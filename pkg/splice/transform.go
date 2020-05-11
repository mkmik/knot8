// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package splice

import (
	"bytes"
	"sort"
	"unicode/utf8"

	"golang.org/x/text/transform"
)

// A Transformer transforms some text applying Ops (Replacements on Selections).
type Transformer struct {
	ops []Op         // replacement operations
	op  int          // current op
	off int          // current source offset
	old bytes.Buffer // old content of the span
}

func NewTransformer(ops ...Op) *Transformer {
	sorted := make([]Op, len(ops))
	copy(sorted, ops)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Start < sorted[j].Start })
	t := &Transformer{ops: sorted}
	t.Reset()
	return t
}

func (t *Transformer) Reset() {
	t.op = 0
	t.off = 0
	t.old.Reset()
}

func (t *Transformer) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) {
	defer func() {
		t.off += nSrc
	}()

	inSpan := false
	for {
		for t.op < len(t.ops) {
			op := t.ops[t.op]
			if t.off+nSrc == op.Start {
				inSpan = true
				t.old.Reset()
			}
			if t.off+nSrc == op.End {
				new, err := op.Replace(t.old.String(), "  demo:") // TODO capture context
				if err != nil {
					return nDst, nSrc, err
				}
				if len(new) > len(dst[nDst:]) {
					return nDst, nSrc, transform.ErrShortDst
				}
				copy(dst[nDst:], []byte(new))
				nDst += len(new)
				inSpan = false
				t.op++
			} else {
				break
			}
			// there could be new span starting back to back to this span end, hence looping.
		}
		// spans can address one past the end of the input, hence we first have to do ^^^
		// and check whether to exit the loop only here:
		if nSrc >= len(src) {
			break
		}

		r, size := utf8.DecodeRune(src[nSrc:])
		if r == utf8.RuneError && !atEOF && !utf8.FullRune(src[nSrc:]) {
			return nDst, nSrc, transform.ErrShortSrc
		}
		if inSpan {
			t.old.WriteRune(r)
		} else {
			if size > len(dst[nDst:]) {
				return nDst, nSrc, transform.ErrShortDst
			}
			nDst += utf8.EncodeRune(dst[nDst:], r)
		}
		nSrc += size
	}
	return nDst, nSrc, nil
}
