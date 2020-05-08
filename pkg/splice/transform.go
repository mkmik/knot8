// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package splice

import (
	"bufio"
	"bytes"
	"io"
	"sort"
)

type replacer struct {
	ext  extent
	repl func(prev string) (string, error)
}

type extent struct {
	Start int
	End   int
}

// splice copies text from r to w while replacing text at given rune extents,
// as specified by the reps slice. The text to be replaced is provided via a callback
// function "replace" in the replacer structures.
func splice(w io.Writer, r io.Reader, reps ...replacer) error {
	wbuf, rbuf := bufio.NewWriter(w), bufio.NewReader(r)
	defer wbuf.Flush()

	sorted := make([]replacer, len(reps))
	copy(sorted, reps)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ext.Start < sorted[j].ext.Start })

	pos := 0
	var prev bytes.Buffer
	for _, rep := range sorted {
		// Copy out the span until the start of the current extent.
		if err := copyRunesN(wbuf, rbuf, rep.ext.Start-pos); err != nil {
			return err
		}

		// Consume the old content of the extent to be replaced.
		// Save it into a buffer because the quoting heuristic needs the previous value.
		if err := copyRunesN(&prev, rbuf, rep.ext.End-rep.ext.Start); err != nil {
			return err
		}

		next, err := rep.repl(prev.String())
		if err != nil {
			return err
		}
		if _, err := wbuf.WriteString(next); err != nil {
			return err
		}
		prev.Reset()

		pos = rep.ext.End
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
