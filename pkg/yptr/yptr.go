// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

/*
Package yptr is a JSONPointer implementation that can walk though a yaml.Node tree.

yaml.Nodes preserve comments and locations in the source and can be useful to implement editing in-place
functionality that uses JSONPointer to locate the fields to be edited.

It also implements a simple extension to the JSONPointers standard that handles pointers into k8s manifests
which usually contain arrays whose elements are objects with a field that uniquely specifies the array entry
(e.g. "name").

For example, given a JSON/YAML input document:

    {"a": [{"k":"x", "v": 42}, {"k":"y", "v": 77}]}

If "k" is a field that contains a key that uniquiely identifies an element in a given array,
we can select the node with the scalar 42 by first selecting the array element for which "k"
has the value of "x", and then by walking to the field "v":

    /a/~{"k":"x"}/v

The "~" token accepts an argument which is interpreted as JSON value to be used as "query-by-example" filter
against elements of an array.
The array element is selected if the query-by-example object is a (recursive) subset of the element.

The ~{...} extension can potentially locate multiple matches. For example, "~{}" effectively acts as a wildcard.
This library offers an API to retrieve multiple matches ("FindAll") or to fetch only one match and error if multiple matches are found ("Find").

JSONPointer is designed to locate exactly one node in the tree. This can be achieved only if the effective
schema of the JSON/YAML document mandates that there is an identifying key in each array element you want to point to.
Using the "Find" function effectively performs a dynamic check of that invariant.

*/
package yptr

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/go-openapi/jsonpointer"
	yaml "gopkg.in/yaml.v3"
)

var (
	// ErrTooManyResults means a pointer matches too many results (usually more than one expected result).
	ErrTooManyResults = fmt.Errorf("too many results")
	// ErrNotFound a pointer failed to find a match.
	ErrNotFound = fmt.Errorf("not found")
)

// FindAll finds all locations in the json/yaml tree pointed by root that match the extended
// JSONPointer passed in ptr.
func FindAll(root *yaml.Node, ptr string) ([]*yaml.Node, error) {
	// TODO: remove dependency on jsonpointer since we only use it to split and unescape the pointer, which is trivial and well defined by the spec.
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

// Find is like FindAll but returns ErrTooManyResults if multiple matches are located.
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

// find recursively matches a token against a yaml node.
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

// match matches a JSONPointer token against a yaml Node.
//
// If root is a map, it performs a field lookup using tok as field name,
// and if found it will return a singleton slice containing the value contained
// in that field.
//
// If root is an array and tok is a number i, it will return the ith element of that array.
// If tok is ~{...}, it will parse the {...} object as a JSON object
// and use it to filter the array using a treeSubsetPred.
// If tok is ~[key=value] it will use keyValuePred to filter the array.
func match(root *yaml.Node, tok string) ([]*yaml.Node, error) {
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
			var mtree yaml.Node
			if err := yaml.Unmarshal([]byte(tok[1:]), &mtree); err != nil {
				return nil, err
			}
			return filter(c, treeSubsetPred(&mtree))
		case strings.HasPrefix(tok, "~["): // alternative syntax: ~[name=app]
			s := strings.SplitN(strings.TrimSuffix(strings.TrimPrefix(tok, "~["), "]"), "=", 2)
			if len(s) != 2 {
				return nil, fmt.Errorf("syntax error, expecting ~[key=value]")
			}
			key, value := s[0], s[1]
			return filter(c, keyValuePred(key, value))
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

type nodePredicate func(*yaml.Node) bool

// filter applies a nodePredicate to each input node and returns only those for which the predicate
// function returns true.
func filter(nodes []*yaml.Node, pred nodePredicate) ([]*yaml.Node, error) {
	var res []*yaml.Node
	for _, n := range nodes {
		if pred(n) {
			res = append(res, n)
		}
	}
	return res, nil
}

// A treeSubsetPred is a node predicate that returns true if tree a is a subset of tree b.
func treeSubsetPred(a *yaml.Node) nodePredicate {
	return func(b *yaml.Node) bool {
		return isTreeSubset(a, b)
	}
}

// keyValuePred is a node predicate that returns true if a node is an object that has a field
// called key whose value is value.
func keyValuePred(key, value string) nodePredicate {
	a := &yaml.Node{
		Kind: yaml.MappingNode, Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: key},
			{Kind: yaml.ScalarNode, Value: value},
		},
	}
	return treeSubsetPred(a)
}
