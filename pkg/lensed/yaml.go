// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package lensed

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/vmware-labs/go-yaml-edit"
	"github.com/vmware-labs/go-yaml-edit/splice"
	yptr "github.com/vmware-labs/yaml-jsonpointer"
	"golang.org/x/text/transform"
	"gopkg.in/yaml.v3"
)

// YAMLLens implements the "yaml" lens.
type YAMLLens struct{}

// Apply implements the Lens interface.
func (YAMLLens) Apply(src []byte, vals []Setter) ([]byte, error) {
	var root yaml.Node
	if err := yaml.Unmarshal(src, &root); err != nil {
		return nil, err
	}

	var ops []splice.Op
	for _, v := range vals {
		f, err := yptr.Find(&root, v.Pointer)
		if err != nil {
			return nil, err
		}

		b, err := v.Value.Transform([]byte(f.Value))
		if err != nil {
			return nil, err
		}
		ops = append(ops, yamled.Node(f).With(string(b)))
	}

	b, _, err := transform.Bytes(yamled.T(ops...), src)
	return b, err
}

// MultiYAMLLens implements the "yamls" lens.
type MultiYAMLLens struct{}

// Apply implements the Lens interface.
func (MultiYAMLLens) Apply(src []byte, vals []Setter) ([]byte, error) {
	docs, err := parseAllYAMLDocs(src)
	if err != nil {
		return nil, err
	}

	var ops []splice.Op
	for _, v := range vals {
		head, tail, err := chompJSONPointer(v.Pointer)
		if err != nil {
			return nil, err
		}
		n, err := strconv.Atoi(head)
		if err != nil {
			return nil, err
		}

		root := docs[n]
		f, err := yptr.Find(root, tail)
		if err != nil {
			return nil, err
		}

		b, err := v.Value.Transform([]byte(f.Value))
		if err != nil {
			return nil, err
		}
		ops = append(ops, yamled.Node(f).With(string(b)))
	}

	b, _, err := transform.Bytes(yamled.T(ops...), src)
	return b, err

	return nil, fmt.Errorf("not implemented yet")
}

func parseAllYAMLDocs(src []byte) (res []*yaml.Node, err error) {
	dec := yaml.NewDecoder(bytes.NewReader(src))
	for {
		var n yaml.Node
		if err := dec.Decode(&n); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return nil, err
		}
		res = append(res, &n)
	}
	return res, nil
}

// chompJSONPointer splits a JSONPointer into the first component and the tail.
// The tail is a valid JSONPointer (i.e. it retains the leading /).
// If ptr contains only one component, an empty tail is returned.
func chompJSONPointer(ptr string) (head, tail string, err error) {
	if !strings.HasPrefix(ptr, "/") {
		return "", "", fmt.Errorf("%q not valid JSONPointer: doesn't start with '/'", ptr)
	}
	c := strings.SplitN(ptr, "/", 3)
	if len(c) == 2 {
		return c[1], "", nil
	}
	return c[1], "/" + c[2], nil
}
