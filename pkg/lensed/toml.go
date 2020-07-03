// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package lensed

import (
	"fmt"
	"strings"

	"github.com/go-openapi/jsonpointer"
	"github.com/pelletier/go-toml"
	"github.com/vmware-labs/go-yaml-edit/splice"
	"golang.org/x/text/transform"
)

// TOMLLens implements the "toml" lens.
type TOMLLens struct{}

// Apply implements the Lens interface.
func (TOMLLens) Apply(src []byte, vals []Setter) ([]byte, error) {
	t, err := toml.LoadBytes(src)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(src), "\n")
	lineStarts := make([]int, len(lines))
	prev := 0
	for i := range lines {
		lineStarts[i] = prev
		prev += len(lines[i]) + 1 // trailing newline
	}

	var ops []splice.Op

	for _, v := range vals {
		p, err := jsonpointer.New(v.Pointer)
		if err != nil {
			return nil, err
		}
		path := p.DecodedTokens()
		pos := t.GetPositionPath(path)
		if pos.Invalid() {
			return nil, fmt.Errorf("cannot find position of %s", v.Pointer)
		}
		old, ok := t.GetPath(path).(string)
		if !ok {
			return nil, fmt.Errorf("type not supported %T", t.GetPath(path))
		}

		start, end, err := findTOMLValue(lines[pos.Line-1])
		if err != nil {
			return nil, err
		}

		start += lineStarts[pos.Line-1]
		end += lineStarts[pos.Line-1]

		newval, err := v.Value.Transform([]byte(old))
		if err != nil {
			return nil, err
		}
		// TODO figure out if Go %q quoting is compatible with TOML.
		ops = append(ops, splice.Span(start, end).With(fmt.Sprintf("%q", newval)))
	}

	b, _, err := transform.Bytes(splice.T(ops...), src)
	return b, err
}

// findTOMLValue is a quick&dirty parser of a TOML assignment expression that
// returns the accurate start/end position of the value component
func findTOMLValue(line string) (start int, end int, err error) {
	type stateFn func(r rune) stateFn
	line += " " // the state machine needs to run one char after the closing quote of the value

	var (
		outkey, outval, over stateFn

		done bool
		i    int
	)

	esc := func(next stateFn) stateFn {
		var s stateFn
		s = func(r rune) stateFn {
			return s
		}
		return s
	}
	str := func(next stateFn) stateFn {
		var s stateFn
		s = func(r rune) stateFn {
			if r == '\\' {
				return esc(s)
			}
			if r == '"' {
				return next
			}
			return s
		}
		return s
	}

	outkey = func(r rune) stateFn {
		if r == '"' {
			return str(outkey)
		}
		if r == '=' {
			return outval
		}
		return outkey
	}

	outval = func(r rune) stateFn {
		if r == '"' {
			start = i
			return str(over)
		}
		return outval
	}

	over = func(r rune) stateFn {
		done = true
		return over
	}

	var (
		state = outkey
		r     rune
	)
	for i, r = range line {
		state = state(r)
		if done {
			break
		}
	}
	return start, i, nil
}
