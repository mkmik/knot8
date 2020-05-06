// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package yamled

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"
)

// An Replacement structure captures a request to replace Value into a given extent of a yaml file.
type Replacement struct {
	ext   Extent
	value string
}

// NewReplacement constructs a new Replacement structure from a value and a yaml.Node.
func NewReplacement(value string, node *yaml.Node) Replacement {
	return Replacement{NewExtent(node), value}
}

// Replace copies text from r to w while replacing text at given rune extents,
// as specified by the reps slice.
func Replace(w io.Writer, r io.Reader, reps []Replacement) error {
	wbuf, rbuf := bufio.NewWriter(w), bufio.NewReader(r)
	defer wbuf.Flush()

	sreps := make([]Replacement, len(reps))
	copy(sreps, reps)
	sort.Slice(sreps, func(i, j int) bool { return reps[i].ext.Start < reps[j].ext.Start })

	pos := 0
	var prev bytes.Buffer
	for _, rep := range sreps {
		// Copy out the span until the start of the current extent.
		if err := copyRunesN(wbuf, rbuf, rep.ext.Start-pos); err != nil {
			return err
		}

		// Consume the old content of the extent to be replaced.
		// Save it into a buffer because the quoting heuristic needs the previous value.
		if err := copyRunesN(&prev, rbuf, rep.ext.End-rep.ext.Start); err != nil {
			return err
		}

		q, err := quote(rep.value, prev.String())
		if err != nil {
			return err
		}
		prev.Reset()

		// Emit the quoted string
		if _, err := wbuf.WriteString(q); err != nil {
			return err
		}

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

// quote quotes a string into a yaml string.
// It tries to preserve original quotation style, when it's likely to be intentional.
// In particular, if the original value had to be quoted (e.g. a number) and the new value
// doesn't have to be quoted, then the quotes will be dropped. OTOH, if the original value
// didn't have to be quoted in the first place, we'll make sure the new value is also quoted,
// just to avoid frustrating the user who intentionally quoted a string that didn't have to be quoted.
// If the user didn't intentionally quote the string, we're not making the original file style any
// worse than it already was.
//
// TODO: preserve input indentation level
func quote(value, old string) (string, error) {
	indent := 2 // TODO: detect

	if len(old) > 0 {
		q := old[0]
		if q == '"' || q == '\'' {
			reEncoded, err := yamlRoundTrip(old, indent)
			if err != nil {
				return "", err
			}
			if reEncoded[0] != q {
				if q == '"' {
					return jsonMarshalString(value)
				} else {
					return yamlStringTrySingleQuoted(value, indent)
				}
			}
		}
	}

	return yamlString(value, indent)
}

func jsonMarshalString(value interface{}) (string, error) {
	b, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// yamlRoundTrip decodes and a string from YAML and reencodes it into YAML.
func yamlRoundTrip(str string, indent int) (string, error) {
	var parsed string
	if err := yaml.Unmarshal([]byte(str), &parsed); err != nil {
		return "", err
	}
	return yamlString(parsed, indent)
}

// yamlString returns a string quoted with yaml rules.
func yamlString(value string, indent int) (string, error) {
	if value == "" {
		return "", nil
	}
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(indent)
	if err := enc.Encode(value); err != nil {
		return "", err
	}
	s := buf.String()
	return s[:len(s)-1], nil // strip trailing newline emitted by yaml marshaling.
}

func isPrintable(s string) bool {
	for _, r := range s {
		if !unicode.IsPrint(r) {
			return false
		}
	}
	return true
}

// yamlStringTrySingleQuoted will return a single quoted YAML string, unless
// it's impossible to encode it as such (e.g. because it contains non-printable chars),
// in that case it will return whatever encoding yamlString picks.
func yamlStringTrySingleQuoted(s string, indent int) (string, error) {
	if !isPrintable(s) {
		return yamlString(s, indent)
	}
	return fmt.Sprintf("'%s'", strings.ReplaceAll(s, "'", "''")), nil
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
