// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package yamled

import (
	"gopkg.in/yaml.v3"
	"knot8.io/pkg/splice"
)

// Quoted transforms a splice.Op into an op that quotes the replacement string according to yaml rules.
func Quoted(op splice.Op) splice.Op {
	o := op
	saved := o.Replace
	o.Replace = func(prev string) (string, error) {
		v, err := saved(prev)
		if err != nil {
			return "", err
		}
		return quote(v, prev)
	}
	return o
}

// Node returns a selection that spans over a yaml node.
func Node(n *yaml.Node) Selection {
	// IndexEnd incorrectly includes trailing newline when strings are multiline.
	// TODO(mkm): remove hack once upstream is patched
	d := 0
	if n.Style&(yaml.LiteralStyle|yaml.FoldedStyle) != 0 {
		d = 1
	}
	return Selection{splice.Span(n.Index, n.IndexEnd-d)}
}

type Selection struct {
	Selection splice.Selection
}

func (s Selection) With(v string) splice.Op {
	return Quoted(s.Selection.With(v))
}