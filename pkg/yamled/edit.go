// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package yamled

import (
	"io"
	"io/ioutil"

	"gopkg.in/yaml.v3"
	"knot8.io/pkg/splice"
)

// Replacer replaces nodes in a yaml file.
type Replacer struct {
	Replacements []Replacement
}

func NewReplacer(rs ...Replacement) Replacer {
	return Replacer{rs}
}

// Bytes applies the replacer on a byte buffer.
func (r Replacer) Bytes(b []byte) ([]byte, error) {
	return splice.Bytes(b, r.asOps()...)
}

func (r Replacer) File(filename string) error {
	return splice.File(filename, r.asOps()...)
}

func (r Replacer) asOps() []splice.Op {
	reps := make([]splice.Op, len(r.Replacements))
	for i, rep := range r.Replacements {
		reps[i] = rep.asOp()
	}
	return reps
}

// Extent is a pair of start+end rune indices.
type Extent struct {
	Start int
	End   int
}

func (e Extent) AsSelection() splice.Selection { return splice.Span(e.Start, e.End) }

// NewExtent returns a Extent that covers the extent of a given yaml.Node.
func NewExtent(n *yaml.Node) Extent {
	// IndexEnd incorrectly includes trailing newline when strings are multiline.
	// TODO(mkm): remove hack once upstream is patched
	d := 0
	if n.Style&(yaml.LiteralStyle|yaml.FoldedStyle) != 0 {
		d = 1
	}
	return Extent{n.Index, n.IndexEnd - d}
}

// An Replacement structure captures a request to replace Value into a given extent of a yaml file.
type Replacement struct {
	ext   Extent
	value string
}

func (r Replacement) asOp() splice.Op {
	return r.ext.AsSelection().WithFunc(func(prev string) (string, error) {
		return quote(r.value, prev)
	})
}

// NewReplacement constructs a new Replacement structure from a value and a yaml.Node.
func NewReplacement(value string, node *yaml.Node) Replacement {
	return Replacement{NewExtent(node), value}
}

// Extract returns a slice of strings for each extent of the input reader.
// The order of the resulting slice matches the order of the provided exts slice
// (which can be in any order; extract provides the necessary sorting to guarantee a single
// scan pass on the reader).
func Extract(r io.ReadSeeker, exts ...Extent) ([]string, error) {
	var (
		reps = make([]splice.Op, len(exts))
		res  = make([]string, len(exts))
	)
	for i, ext := range exts {
		i := i
		reps[i] = ext.AsSelection().WithFunc(func(prev string) (string, error) {
			res[i] = prev
			return prev, nil
		})
	}
	if err := splice.Transform(ioutil.Discard, r, reps...); err != nil {
		return nil, err
	}
	return res, nil
}
