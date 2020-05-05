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

	"github.com/mkmik/argsort"
	"gopkg.in/yaml.v3"
)

// RuneSplicer implementations allow in-place editing of buffers by using rune positions ranges
type RuneSplicer interface {
	// Splice replaces the contents from rune positions start to end with the given string value.
	Splice(value string, start, end int) error
	// Slice retrieves the existing contents from rune positions start to end.
	Slice(start, end int) (string, error)
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

// Transform implements a streaming transform by applying the non-overlapping edits.
func Transform(w io.Writer, r io.Reader, edits []Edit) error {
	edmap := argsort.SortSlice(edits, func(i, j int) bool { return edits[i].ext.Start < edits[j].ext.Start })

	wbuf, rbuf := bufio.NewWriter(w), bufio.NewReader(r)
	defer wbuf.Flush()

	for i, e := 0, 0; e < len(edits); i++ {
		ch, _, err := rbuf.ReadRune()
		if err != nil {
			return err
		}

		ed := edits[edmap[e]]
		if ed.ext.Start == i {
			l := ed.ext.End - ed.ext.Start - 1
			var old []rune
			for j := 0; j < l; j++ {
				och, _, err := rbuf.ReadRune()
				if err != nil {
					return err
				}
				old = append(old, och)
			}
			i += l
			e++

			q, err := quote(ed.value, string(old))
			if err != nil {
				return err
			}

			if _, err := wbuf.WriteString(q); err != nil {
				return err
			}

			continue
		}

		if _, err := wbuf.WriteRune(ch); err != nil {
			return err
		}
	}
	_, err := io.Copy(wbuf, rbuf)
	return err
}

// Splice edits a file in place by performing a set of edits.
func Splice(buf RuneSplicer, edits []Edit) error {
	backwards := make([]Edit, len(edits))
	copy(backwards, edits)
	sort.Slice(backwards, func(i, j int) bool { return backwards[i].ext.Start > backwards[j].ext.Start })

	for _, e := range backwards {
		o, err := buf.Slice(e.ext.Start, e.ext.End)
		if err != nil {
			return err
		}
		q, err := quote(e.value, o)
		if err != nil {
			return err
		}
		if err := buf.Splice(q, e.ext.Start, e.ext.End); err != nil {
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

// A RuneBuffer is a trivial implementation of a RuneSplicer that uses a rune slice.
type RuneBuffer []rune

func (buf *RuneBuffer) Splice(value string, start, end int) error {
	*buf = append((*buf)[:start], append(bytes.Runes([]byte(value)), (*buf)[end:]...)...)
	return nil
}

func (buf *RuneBuffer) Slice(start, end int) (string, error) {
	return string((*buf)[start:end]), nil
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
