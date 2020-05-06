// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package yamled

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

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

// Replace copies text from r to w while replacing text at given rune extents,
// as specified by the reps slice.
func Replace(w io.Writer, r io.Reader, rs ...Replacement) error {
	reps := make([]replacer, len(rs))
	for i := range rs {
		reps[i] = rs[i].asReplacer()
	}
	return transform(w, r, reps)
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

// UpdateFie updates a file in place.
func UpdateFile(filename string, rs ...Replacement) error {
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

	if err := Replace(out, in, rs...); err != nil {
		return err
	}
	out.Close()

	return os.Rename(out.Name(), filename)
}
