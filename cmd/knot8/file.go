// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/mattn/go-isatty"
	"github.com/mkmik/multierror"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
)

// A shadowFile is in-memory copy of a file that can be commited back to disk.
type shadowFile struct {
	name string
	buf  []byte
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

	buf, err := readAllTranscode(r)
	if err != nil {
		return nil, err
	}
	return &shadowFile{name: filename, buf: buf}, nil
}

func (f *shadowFile) update(up func(b []byte) ([]byte, error)) error {
	b, err := up(f.buf)
	if err != nil {
		return err
	}
	f.buf = b
	return nil
}

func (f *shadowFile) Commit() error {
	b := []byte(string(f.buf))
	var w io.Writer
	if f.name == "-" {
		w = os.Stdout
	} else {
		file, err := os.OpenFile(f.name, os.O_WRONLY|os.O_TRUNC, 0)
		if err != nil {
			return err
		}
		defer file.Close()
		w = file
	}

	_, err := w.Write(b)
	return err
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

// readAllTranscode reads a text input encoded as either UTF-8 or UTF-16, both LE and BE
// (which are the supported encodings of YAML),
func readAllTranscode(r io.Reader) ([]byte, error) {
	t := unicode.BOMOverride(runes.ReplaceIllFormed())
	return ioutil.ReadAll(transform.NewReader(r, t))
}
