// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

/*
Package splice allows to perform simple edits on a string, byte buffer or a file.

It allows to delete, insert or replace strings in a text buffer.

The core operation is: replace the current content of a given selection with a new string.
Deletion is just replacement with an empty string.
Insertion is just replacement at a zero length selection.

Selections can be constructed from absolute start/end positions in the text buffer (unicode character offsets, not byte offsets!), of from line+column numbers (1-based, columns are unicode character offsets, not bytes).

The edit operation involves one single pass through the input.
A second pass through the input is currently necessary when using Line+Column numbers (see Loc).
*/
package splice

import (
	"bufio"
	"errors"
	"io"
	"io/ioutil"
)

func T(ops ...Op) Transformer { return Transformer(ops) }

type Transformer []Op

func (t Transformer) Transform(w io.Writer, r io.ReadSeeker) error {
	re, err := resolvePositions(r, t)
	if err != nil {
		return err
	}
	return splice(w, r, re...)
}

// A Op captures a request to replace a selection with a replacement string.
// An idiomatic way to construct an Op instance is to call With or WithFunc on a Selection.
type Op struct {
	Selection
	Replace func(prev string) (string, error)
}

// A selection selects a range of characters in the input string buffer.
// It's defined to be the range that starts at Start end ends before the End position.
type Selection struct {
	Start Pos
	End   Pos
}

func (s Selection) asExtent() extent {
	return extent{s.Start.offset(), s.End.offset()}
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

// A Pos is a position in the input string buffer.
type Pos interface {
	// absolute position, or 0 if not known
	offset() int
	// return true if current position matches provided line
	match(line, col int) bool
}

// An Offset is an absolute rune position in the input string buffer.
func Offset(pos int) Pos { return offset(pos) }

type offset int

func (a offset) offset() int       { return int(a) }
func (offset) match(int, int) bool { return false }

// A Loc is a line:col position in the input string buffer.
func Loc(line, col int) Pos { return loc{line, col} }

type loc struct {
	line int
	col  int
}

func (loc) offset() int                { return 0 }
func (l loc) match(line, col int) bool { return l.line == line && l.col == col }

// Sel constructs a selection from a start and end position.
func Sel(start, end Pos) Selection { return Selection{start, end} }

// An Span is a shortcut for splice.Sel(splice.Offset(start), splice.Offset(end)).
func Span(start, end int) Selection { return Sel(Offset(start), Offset(end)) }

// Peek returns a slice of strings for each extent of the input reader.
// The order of the resulting slice matches the order of the provided selection slice
// (which can be in any order; slice provides the necessary sorting to guarantee a single
// scan pass on the reader).
func Peek(r io.ReadSeeker, sels ...Selection) ([]string, error) {
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
	if err := T(reps...).Transform(ioutil.Discard, r); err != nil {
		return nil, err
	}
	return res, nil
}

// resolvePositions resolves line:col positions by performing one pass through a reader.
// It's useful because the current transform implementation can only handle absolute rune addresses.
func resolvePositions(in io.ReadSeeker, rs []Op) ([]replacer, error) {
	defer in.Seek(0, 0)

	res := make([]replacer, len(rs))
	for i, r := range rs {
		res[i] = replacer{
			ext:  r.asExtent(),
			repl: r.Replace,
		}
	}

	rbuf := bufio.NewReader(in)
	line, col := 1, 0
	for i := 0; ; i++ {
		ch, _, err := rbuf.ReadRune()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}

		for j := range rs {
			if rs[j].Start.match(line, col) {
				res[j].ext.Start = i
			}
			if rs[j].End.match(line, col) {
				res[j].ext.End = i
			}
		}

		if ch == '\n' {
			line++
			col = 0
		}
		col++
	}
	return res, nil
}