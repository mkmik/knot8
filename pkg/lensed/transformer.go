// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package lensed

import (
	"golang.org/x/text/transform"
)

// NewTransformer returns a transform.Transform implementation for a given replacer.
func NewTransformer(r Replacer) ReplacerTransformer {
	return ReplacerTransformer{r: r}
}

// A ReplacerTransformer is a transform.Transformer that applies a Replacer.
type ReplacerTransformer struct {
	r Replacer
	b []byte
}

// Reset implements the golang.org/x/text/transform.Transformer interface.
func (t *ReplacerTransformer) Reset() {
	t.b = nil
}

// Transform implements the golang.org/x/text/transform.Transformer interface.
func (t *ReplacerTransformer) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) {
	if !atEOF {
		return 0, 0, transform.ErrShortSrc
	}
	if t.b != nil {
		b, err := t.r.Transform(src)
		if err != nil {
			return 0, 0, err
		}
		t.b = b
	}
	if len(dst) < len(t.b) {
		return 0, len(src), transform.ErrShortDst
	}
	copy(dst, t.b)
	return len(dst), len(src), nil
}
