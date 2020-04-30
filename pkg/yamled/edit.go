// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package yamled

import (
	"bytes"
	"sort"

	"gopkg.in/yaml.v3"
)

// RuneSplicer implementations allow in-place editing of buffers by using rune positions ranges
type RuneSplicer interface {
	// Splice replaces the contents from rune positions start to end with the given string value.
	Splice(value string, start, end int) error
}

// An Edit structure captures a request to splice Value into a given extent of a buffer.
type Edit struct {
	ext   Extent
	value string
}

// NewEdit constructs a new Edit structure from a value and a yaml.Node.
func NewEdit(value string, node *yaml.Node) Edit {
	return Edit{NewExtent(node), value}
}

// Splice edits a file in place by performing a set of edits.
func Splice(buf RuneSplicer, edits []Edit) error {
	backwards := make([]Edit, len(edits))
	copy(backwards, edits)
	sort.Slice(backwards, func(i, j int) bool { return backwards[i].ext.Start > backwards[j].ext.Start })

	for _, e := range backwards {
		q, err := quote(e.value)
		if err != nil {
			return err
		}
		if err := buf.Splice(q, e.ext.Start, e.ext.End); err != nil {
			return err
		}
	}
	return nil
}

// quote quotes a yaml string.
// TODO: try to preserve previous quoting style and indentation level
func quote(value string) (string, error) {
	if value == "" {
		return "", nil
	}
	b, err := yaml.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(b)[:len(b)-1], nil
}

// A RuneBuffer is a trivial implementation of a RuneSplicer that uses a rune slice.
type RuneBuffer []rune

func (buf *RuneBuffer) Splice(value string, start, end int) error {
	*buf = append((*buf)[:start], append(bytes.Runes([]byte(value)), (*buf)[end:]...)...)
	return nil
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
