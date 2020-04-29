// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/mattn/go-isatty"
	"github.com/mkmik/multierror"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"gopkg.in/yaml.v3"
)

// A shadowFile is in-memory copy of a file that can be commited back to disk.
type shadowFile struct {
	name string
	buf  []rune
}

func newShadowFile(filename string) (*shadowFile, error) {
	var r io.Reader
	if filename == "-" {
		if isatty.IsTerminal(os.Stdin.Fd()) {
			fmt.Fprintf(os.Stderr, "(reading manifests from standard input; hit ctrl-c if this is not what you wanted)\n")
		}
		r = os.Stdin
	} else {
		f, err := os.Open(filename)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		r = f
	}

	buf, err := readFileRunes(r)
	if err != nil {
		return nil, err
	}
	return &shadowFile{name: filename, buf: buf}, nil
}

func (s *shadowFile) Commit() error {
	b := []byte(string(s.buf))
	var w io.Writer
	if s.name == "-" {
		w = os.Stdout
	} else {
		f, err := os.OpenFile(s.name, os.O_WRONLY|os.O_TRUNC, 0)
		if err != nil {
			return err
		}
		defer f.Close()
		w = f
	}

	_, err := w.Write(b)
	return err
}

type runeRange struct {
	start int
	end   int
}

func (r runeRange) slice(src []rune) []rune {
	return src[r.start:r.end]
}

type edit struct {
	runeRange
	value string
}

func mkEdit(value string, node *yaml.Node) edit {
	return edit{mkRuneRange(node), value}
}

// patch edits a file in place by replacing each of the given rune ranges in the file
// buf with a given string value.
func (f *shadowFile) patch(edits []edit) error {
	backwards := make([]edit, len(edits))
	copy(backwards, edits)
	sort.Slice(backwards, func(i, j int) bool { return backwards[i].start > backwards[j].start })

	for _, e := range backwards {
		f.buf = append(f.buf[:e.start], append(bytes.Runes([]byte(e.value)), f.buf[e.end:]...)...)
	}
	return nil
}

// expandPaths will expand all path entries and return a slice of file paths.
// If an input path points to a directory it will return all *.yaml files contained in it.
// Shell globs are resolved.
func expandPaths(paths []string) ([]string, error) {
	var (
		res  []string
		errs []error
	)
	glob := func(p string) ([]string, bool, error) {
		g, err := filepath.Glob(p)
		if err != nil {
			return nil, false, err
		}
		res, err := onlyFiles(g)
		return res, len(g) > 0, err
	}
	add := func(p string) bool {
		g, found, err := glob(p)
		if err != nil {
			errs = append(errs, err)
		} else {
			res = append(res, g...)
		}
		return found
	}

	for _, p := range paths {
		// special case for stdin pseudo path
		if p == "-" {
			res = append(res, p)
			continue
		}
		if found := add(p); !found {
			errs = append(errs, fmt.Errorf("%q matched no files", p))
		}
		_ = add(p + "/*.yaml")
		_ = add(p + "/*.yml")
	}
	if errs != nil {
		return nil, multierror.Join(errs)
	}
	return res, nil
}

// onlyFiles filter the paths and excludes directories.
// This function assumes all  paths exist.
func onlyFiles(paths []string) ([]string, error) {
	var res []string

	for _, p := range paths {
		st, err := os.Stat(p)
		if err != nil {
			return nil, nil
		}
		if !st.IsDir() {
			res = append(res, p)
		}
	}

	return res, nil
}

// readFileRunes reads a text file encoded as either UTF-8 or UTF-16, both LE and BE
// (which are the supported encodings of YAML), and return an array of runes which
// we can operate on in order to implement rune-addressed in-place edits.
func readFileRunes(r io.Reader) ([]rune, error) {
	t := unicode.BOMOverride(runes.ReplaceIllFormed())
	return readAllRunes(bufio.NewReader(transform.NewReader(r, t)))
}

// readAllRunes returns a slice of runes. API modeled after ioutil.ReadAll but the implementation is inefficient.
func readAllRunes(r io.RuneReader) ([]rune, error) {
	var res []rune
	for {
		ch, _, err := r.ReadRune()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		res = append(res, ch)
	}
	return res, nil
}
