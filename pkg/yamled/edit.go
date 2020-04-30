// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package yamled

import (
	"bytes"
	"fmt"
	"sort"

	"gopkg.in/yaml.v3"
)

// RuneSplicer implementations allow in-place editing of buffers by using rune positions ranges
type RuneSplicer interface {
	// Splice replaces the contents from rune positions start to end with the given string value.
	Splice(value string, start, end int) error
}

// An Edit structure captures a request to splice Value into a given rune range of a buffer.
type Edit struct {
	Extent
	Value string
}

// NewEdit constructs a new Edit structure from a value and a yaml.Node.
func NewEdit(value string, node *yaml.Node) Edit {
	return Edit{NewExtent(node), value}
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

// Splice edits a file in place by replacing each of the given rune ranges in the file
// buf with a given string value.
func Splice(buf RuneSplicer, edits []Edit) error {
	backwards := make([]Edit, len(edits))
	copy(backwards, edits)
	sort.Slice(backwards, func(i, j int) bool { return backwards[i].Start > backwards[j].Start })

	for _, e := range backwards {
		if err := buf.Splice(e.Value, e.Start, e.End); err != nil {
			return err
		}
	}
	return nil
}

// A RuneBuffer is a trivial implementation of a RuneSplicer that uses a rune slice.
type RuneBuffer []rune

func (buf RuneBuffer) boundsCheck(start, end int) error {
	if l := len(buf); start < 0 || start >= l || end < start || end >= l {
		return fmt.Errorf("%d:%d out of bound (buf size %d)", start, end, l)
	}
	return nil
}

func (buf *RuneBuffer) Splice(value string, start, end int) error {
	if err := buf.boundsCheck(start, end); err != nil {
		return err
	}
	*buf = append((*buf)[:start], append(bytes.Runes([]byte(value)), (*buf)[end:]...)...)
	return nil
}
