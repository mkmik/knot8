// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package yamled

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
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
	buf := bytes.NewBuffer(make([]byte, 0, len(b)))
	if err := r.transform(buf, bytes.NewReader(b)); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (r Replacer) File(filename string) error {
	in, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := ioutil.TempFile(filepath.Dir(filename), ".*~")
	if err != nil {
		return err
	}
	defer os.RemoveAll(out.Name())

	if err := r.transform(out, in); err != nil {
		return err
	}
	out.Close()

	return os.Rename(out.Name(), filename)
}

func (r Replacer) transform(w io.Writer, in io.Reader) error {
	reps := make([]replacer, len(r.Replacements))
	for i, rep := range r.Replacements {
		reps[i] = rep.asReplacer()
	}
	return transform(w, in, reps)
}

// Extent is a pair of start+end rune indices.
type Extent struct {
	Start int
	End   int
}

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

func (r Replacement) asReplacer() replacer {
	return replacer{r.ext, func(prev string) (string, error) { return quote(r.value, prev) }}
}

// NewReplacement constructs a new Replacement structure from a value and a yaml.Node.
func NewReplacement(value string, node *yaml.Node) Replacement {
	return Replacement{NewExtent(node), value}
}

// Extract returns a slice of strings for each extent of the input reader.
// The order of the resulting slice matches the order of the provided exts slice
// (which can be in any order; extract provides the necessary sorting to guarantee a single
// scan pass on the reader).
func Extract(r io.Reader, exts ...Extent) ([]string, error) {
	var (
		reps = make([]replacer, len(exts))
		res  = make([]string, len(exts))
	)
	for i, ext := range exts {
		i := i
		reps[i] = replacer{ext, func(prev string) (string, error) {
			res[i] = prev
			return prev, nil
		}}
	}
	if err := transform(ioutil.Discard, r, reps); err != nil {
		return nil, err
	}
	return res, nil
}
