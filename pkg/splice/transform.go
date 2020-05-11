// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package splice

import (
	"bufio"
	"bytes"
	"io"
	"sort"

	"golang.org/x/text/transform"
)

// Transformer is a crazy inefficient implementation of the splice transformer
// that first reads the whole input in a buffer, and then performs one transformation
// pass using the old splice(w io.Writer, r io.Reader, reps ...Op) API.
type Transformer struct {
	buf  []byte
	copy func(w io.Writer, r io.Reader) error
}

func (t *Transformer) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) {
	if t.buf != nil {
		if len(dst) < len(t.buf) {
			return 0, 0, transform.ErrShortDst
		}
		copy(dst, t.buf)
		return len(t.buf), len(src), nil
	}
	if !atEOF {
		return 0, 0, transform.ErrShortSrc
	}

	var buf bytes.Buffer
	if err := t.copy(&buf, bytes.NewReader(src)); err != nil {
		return 0, 0, err
	}

	t.buf = buf.Bytes()
	if len(dst) < len(t.buf) {
		return 0, 0, transform.ErrShortDst
	}
	copy(dst, t.buf)
	return len(t.buf), len(src), nil
}

func (t *Transformer) Reset() {
	t.buf = nil
}

// splice copies text from r to w while replacing text at given rune extents,
// as specified by the reps slice. The text to be replaced is provided via a callback
// function "replace" in the replacer structures.
func splice(w io.Writer, r io.Reader, reps ...Op) error {
	wbuf, rbuf := bufio.NewWriter(w), bufio.NewReader(r)
	defer wbuf.Flush()

	sorted := make([]Op, len(reps))
	copy(sorted, reps)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Start < sorted[j].Start })

	pos := 0
	var prev bytes.Buffer
	for _, rep := range sorted {
		// Copy out the span until the start of the current extent.
		if err := copyRunesN(wbuf, rbuf, rep.Start-pos); err != nil {
			return err
		}

		// Consume the old content of the extent to be replaced.
		// Save it into a buffer because the quoting heuristic needs the previous value.
		if err := copyRunesN(&prev, rbuf, rep.End-rep.Start); err != nil {
			return err
		}

		next, err := rep.Replace(prev.String(), "  demo:") // TODO capture context
		if err != nil {
			return err
		}
		if _, err := wbuf.WriteString(next); err != nil {
			return err
		}
		prev.Reset()

		pos = rep.End
	}

	// Copy out the trailing span.
	_, err := io.Copy(wbuf, rbuf)
	return err

}

type runeWriter interface {
	WriteRune(r rune) (size int, err error)
}

func copyRunesN(w runeWriter, r io.RuneReader, n int) error {
	for i := 0; i < n; i++ {
		ch, _, err := r.ReadRune()
		if err != nil {
			return err
		}
		if _, err := w.WriteRune(ch); err != nil {
			return err
		}
	}
	return nil
}
