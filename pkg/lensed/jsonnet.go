// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package lensed

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/go-openapi/jsonpointer"
	"github.com/google/go-jsonnet"
	"github.com/google/go-jsonnet/ast"
	"github.com/vmware-labs/go-yaml-edit/splice"
	"golang.org/x/text/transform"
)

// Jsonnet implements the "jsonnet" lens.
//
// The jsonnet lens implementation is still in its early stages, it only supports:
// 1. double quoted string scalars
// 2. single level ~{"foo":"bar", "baz":"quz"} matchers
type Jsonnet struct{}

// Apply implements the Lens interface.
func (Jsonnet) Apply(src []byte, vals []Setter) ([]byte, error) {
	var ops []splice.Op

	root, err := jsonnet.SnippetToAST("-", string(src))
	if err != nil {
		return nil, err
	}

	for _, v := range vals {
		p, err := jsonpointer.New(v.Pointer)
		if err != nil {
			return nil, err
		}
		path := p.DecodedTokens()

		n, err := findJsonnetNode(root, path)
		if err != nil {
			return nil, err
		}
		start, err := lineColToPos(src, n.Loc().Begin.Line, n.Loc().Begin.Column+1)
		if err != nil {
			return nil, err
		}
		end, err := lineColToPos(src, n.Loc().End.Line, n.Loc().End.Column+1)
		if err != nil {
			return nil, err
		}

		var oldval []byte
		switch n := n.(type) {
		case *ast.LiteralString:
			oldval = []byte(n.Value)
		case *ast.Import:
			return nil, fmt.Errorf("cannot directly address an import node. Please use .../~file")
		default:
			return nil, fmt.Errorf("unhandled node type %T", n)
		}

		newval, err := v.Value.Transform(oldval)
		if err != nil {
			return nil, err
		}

		ops = append(ops, splice.Span(start, end).With(fmt.Sprintf("%q", string(newval))))
	}
	b, _, err := transform.Bytes(splice.T(ops...), src)
	return b, err
}

func checkImportFile(path []string) error {
	if len(path) == 1 && path[0] == "~file" {
		return nil
	}
	return fmt.Errorf("import nodes only support the ~file field, found %q", path)
}

func findJsonnetNode(root ast.Node, path []string) (ast.Node, error) {
	if len(path) == 0 {
		return root, nil
	}
	p := path[0]

	switch n := root.(type) {
	case *ast.DesugaredObject:
		for _, f := range n.Fields {
			if k, ok := f.Name.(*ast.LiteralString); !ok {
				continue
			} else if p == k.Value {
				return findJsonnetNode(f.Body, path[1:])
			}
		}
	case *ast.Array:
		var exprs []ast.Node
		for _, e := range n.Elements {
			exprs = append(exprs, e.Expr)
		}

		e, err := matchJsonnetArrayItem(p, exprs)
		if err != nil {
			return nil, err
		}
		return findJsonnetNode(e, path[1:])
	case *ast.Local:
		return findJsonnetNode(n.Body, path)
	case *ast.Import:
		if err := checkImportFile(path); err != nil {
			return nil, err
		}
		return n.File, nil
	case *ast.ImportStr:
		if err := checkImportFile(path); err != nil {
			return nil, err
		}
		return n.File, nil
	case *ast.ImportBin:
		if err := checkImportFile(path); err != nil {
			return nil, err
		}
		return n.File, nil
	default:
		return nil, fmt.Errorf("unsupported jsonnet node type: %T", n)
	}

	return nil, fmt.Errorf("cannot find field %q in object", p)
}

func lineColToPos(src []byte, line, column int) (int, error) {
	l, c := 1, 1
	for i, r := range string(src) {
		c++
		if r == '\n' {
			l++
			c = 1
		}
		if l == line && c == column {
			return i, nil
		}
	}
	return 0, io.EOF
}

func matchJsonnetArrayItem(p string, exprs []ast.Node) (ast.Node, error) {
	if strings.HasPrefix(p, "~{") {
		var m map[string]string
		if err := json.Unmarshal([]byte(p[1:]), &m); err != nil {
			return nil, err
		}
		nodes, err := filterJsonnetArrayItems(exprs, isTreeSubsetPred(m))
		if err != nil {
			return nil, err
		}
		if got, want := len(nodes), 1; got != want {
			return nil, fmt.Errorf("bad number of subtree matches: got=%d, want=%d", got, want)
		}
		return nodes[0], nil
	}
	i, err := strconv.Atoi(p)
	if err != nil {
		return nil, err
	}
	return exprs[i], nil
}

type jsonnetNodePredicate func(ast.Node) bool

func isTreeSubsetPred(a map[string]string) jsonnetNodePredicate {
	return func(b ast.Node) bool {
		return isTreeSubset(a, b)
	}
}

func isTreeSubset(a map[string]string, b ast.Node) bool {
	for k, v := range a {
		if !jsonnetObjectHasField(b, k, v) {
			return false
		}
	}
	return true
}

func jsonnetObjectHasField(b ast.Node, name, value string) bool {
	if o, ok := b.(*ast.DesugaredObject); ok {
		for _, f := range o.Fields {
			if n, ok := f.Name.(*ast.LiteralString); ok && n.Value == name {
				v, ok := f.Body.(*ast.LiteralString)
				return ok && v.Value == value
			}
		}
	}
	return false
}

func filterJsonnetArrayItems(nodes []ast.Node, pred jsonnetNodePredicate) ([]ast.Node, error) {
	var res []ast.Node
	for _, n := range nodes {
		if pred(n) {
			res = append(res, n)
		}
	}
	return res, nil
}
