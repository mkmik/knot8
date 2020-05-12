// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

/*
Package splice allows to perform simple edits on a string, byte buffer or a file.

It allows to delete, insert or replace strings in a text buffer.

The core operation is: replace the current content of a given selection with a new string.
Deletion is just replacement with an empty string.
Insertion is just replacement at a zero length selection.

Selections are addressed by unicode character offsets, not byte offsets!.

The edit operation involves one single pass through the input.
*/
package splice

import (
	"io"
	"io/ioutil"

	"golang.org/x/text/transform"
)

// Constructs a splice transformer given one or more operations.
// A splice transformer implements golang.org/x/text/transform.Transform;
// that package contains many useful functions to apply the transformation.
func T(ops ...Op) *Transformer { return NewTransformer(ops...) }

// A Op captures a request to replace a selection with a replacement string.
// An idiomatic way to construct an Op instance is to call With or WithFunc on a Selection.
type Op struct {
	Selection
	Replace func(prev string) (string, error)
}

// A selection selects a range of characters in the input string buffer.
// It's defined to be the range that starts at Start end ends before the End position.
// Positions are  unicode codepoint offsets, not byte offsets.
type Selection struct {
	Start int
	End   int
}

// With returns an operation that captures a replacement of the current selection with a desired replacement string.
func (s Selection) With(r string) Op {
	return s.WithFunc(func(string) (string, error) { return r, nil })
}

// WithFunc returns an operation that will call the f callback on the previous value of the selection
// and replace the selection with the return value of the callback.
func (s Selection) WithFunc(f func(prev string) (string, error)) Op {
	return Op{s, f}
}

// Span constructs a Selection.
func Span(start, end int) Selection { return Selection{start, end} }

// Peek returns a slice of strings for each extent of the input reader.
// The order of the resulting slice matches the order of the provided selection slice
// (which can be in any order; slice provides the necessary sorting to guarantee a single
// scan pass on the reader).
func Peek(r io.Reader, sels ...Selection) ([]string, error) {
	var (
		reps = make([]Op, len(sels))
		res  = make([]string, len(sels))
	)
	for i, sel := range sels {
		i := i
		reps[i] = sel.WithFunc(func(prev string) (string, error) {
			res[i] = prev
			return prev, nil
		})
	}

	if _, err := io.Copy(ioutil.Discard, transform.NewReader(r, T(reps...))); err != nil {
		return nil, err
	}
	return res, nil
}
