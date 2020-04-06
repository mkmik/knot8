// Package kptr implements a simple extension to the JSONPointers standard that handles pointers into k8s manifests
// which usually contain arrays whose elements are objects with a field that uniquely specifies the array entry (e.g. "name"
package kptr

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/go-openapi/jsonpointer"
	yaml "gopkg.in/yaml.v3"
)

var (
	ErrTooManyResults = fmt.Errorf("too many results")
	ErrNotFound       = fmt.Errorf("not found")
)

// FindAll finds all locations in the json/yaml tree pointed by root that match the extended
// JSONPointer passed in ptr.
func FindAll(root *yaml.Node, ptr string) ([]*yaml.Node, error) {
	p, err := jsonpointer.New(ptr)
	if err != nil {
		return nil, err
	}
	toks := p.DecodedTokens()

	res, err := find(root, toks)
	if err != nil {
		return nil, fmt.Errorf("%q: %w", ptr, err)
	}
	return res, nil
}

// Like FindAll but returns ErrTooManyResults if multiple matches are located.
func Find(root *yaml.Node, ptr string) (*yaml.Node, error) {
	res, err := FindAll(root, ptr)
	if err != nil {
		return nil, err
	}
	if len(res) > 1 {
		return nil, fmt.Errorf("Got %d matches: %w", len(res), ErrTooManyResults)
	}
	if len(res) == 0 {
		return nil, fmt.Errorf("bad state while finding %q: res is empty but error is: %v", ptr, err)
	}
	return res[0], nil
}

func find(root *yaml.Node, toks []string) ([]*yaml.Node, error) {
	next, err := match(root, toks[0])
	if err != nil {
		return nil, err
	}
	if len(toks) == 1 {
		return next, nil
	}

	var res []*yaml.Node
	for _, n := range next {
		f, err := find(n, toks[1:])
		if err != nil {
			return nil, err
		}
		res = append(res, f...)
	}
	return res, nil
}

func match(root *yaml.Node, tok string) ([]*yaml.Node, error) {
	if false {
		rendered, _ := yaml.Marshal(root)
		log.Printf("searching %q in %q:\n%s\n---", tok, root.Tag, string(rendered))
	}

	c := root.Content
	switch root.Kind {
	case yaml.MappingNode:
		if l := len(c); l%2 != 0 {
			return nil, fmt.Errorf("yaml.Node invariant broken, found %d map content", l)
		}

		for i := 0; i < len(c); i += 2 {
			key, value := c[i], c[i+1]
			if tok == key.Value {
				return []*yaml.Node{value}, nil
			}
		}
	case yaml.SequenceNode:
		switch {
		case strings.HasPrefix(tok, "~{"): // subtree match: ~{"name":"app"}
			var mtree interface{}
			if err := yaml.Unmarshal([]byte(tok[1:]), &mtree); err != nil {
				return nil, err
			}
			// TODO: implement json subtree match this is just a hack
			var key, value string
			for k, v := range mtree.(map[string]interface{}) {
				key = k
				value = v.(string)
				break
			}
			return simpleMatchArray(c, key, value)
		case strings.HasPrefix(tok, "~["): // alternative syntax: ~[name=app]
			s := strings.SplitN(strings.TrimSuffix(strings.TrimPrefix(tok, "~]"), "]"), "=", 2)
			if len(s) != 2 {
				return nil, fmt.Errorf("syntax error, expecting ~[key=value]")
			}
			key, value := s[0], s[1]
			return simpleMatchArray(c, key, value)
		default:
			i, err := strconv.Atoi(tok)
			if err != nil {
				return nil, err
			}
			if i < 0 || i >= len(c) {
				return nil, fmt.Errorf("out of bounds")
			}
			return c[i : i+1], nil
		}
	case yaml.DocumentNode:
		// skip document nodes.
		return match(c[0], tok)
	default:
		return nil, fmt.Errorf("unhandled node type: %v (%v)", root.Kind, root.Tag)
	}
	return nil, fmt.Errorf("%q: %w", tok, ErrNotFound)
}

func simpleMatchArray(c []*yaml.Node, key, value string) ([]*yaml.Node, error) {
	var ma []*yaml.Node
	for _, a := range c {
		e, err := match(a, key)
		if errors.Is(err, ErrNotFound) {
			continue
		} else if err != nil {
			return nil, err
		}
		if e[0].Value == value {
			ma = append(ma, a)
		}
	}
	return ma, nil
}