// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"

	"github.com/mattn/go-isatty"
	"github.com/mkmik/multierror"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
)

// A shadowFile is in-memory copy of a file that can be commited back to disk.
type shadowFile struct {
	name string
	buf  []rune
}

func newShadowFile(f *os.File) (shadowFile, error) {
	r, err := readFileRunes(f)
	if err != nil {
		return shadowFile{}, err
	}
	return shadowFile{name: f.Name(), buf: r}, nil
}

func (m *shadowFile) Commit() error {
	return writeFileRunes(m.name, m.buf)
}

type runeRange struct {
	start int
	end   int
}

func (r runeRange) slice(src []rune) []rune {
	return src[r.start:r.end]
}

// patch edits a file in place by replacing each of the given rune ranges in the file
// buf with a given string value.
func (f *shadowFile) patch(value string, positions []runeRange) error {
	backwards := make([]runeRange, len(positions))
	copy(backwards, positions)
	sort.Slice(backwards, func(i, j int) bool { return positions[i].start > positions[j].start })

	rvalue := bytes.Runes([]byte(value))

	for _, pos := range backwards {
		f.buf = append(f.buf[:pos.start], append(rvalue, f.buf[pos.end:]...)...)
	}

	return nil
}

// openFiles opens all files referenced by the paths slice.
// If a path points to a directory, openFiles will open all *.yaml files contained in it.
func openFiles(paths []string) ([]*os.File, error) {
	var (
		files []*os.File
		errs  []error
	)

	for _, p := range paths {
		fs, err := openManifestsAt(p)
		if err != nil {
			errs = append(errs, err)
		} else {
			files = append(files, fs...)
		}
	}

	if errs != nil {
		return nil, multierror.Join(errs)
	}
	return files, nil
}

// openManifestsAt will open the file p and return it if it's a simple file,
// otherwise, if it's a directory, it will open all the K8s manifest files contained in it (see manifestsInDir).
func openManifestsAt(p string) ([]*os.File, error) {
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}

	if st, err := f.Stat(); err != nil {
		return nil, err
	} else if st.IsDir() {
		paths, err := manifestsInDir(f)
		if err != nil {
			return nil, err
		}
		return openFiles(paths)
	}

	return []*os.File{f}, nil
}

// manifestsInDir returns all potential K8s manifest files in a directory.
func manifestsInDir(dir *os.File) ([]string, error) {
	names, err := dir.Readdirnames(-1)
	if err != nil {
		return nil, err
	}

	var res []string
	for _, n := range names {
		if ok, err := matchExts(n, "yaml", "yml", "json"); err != nil {
			return nil, err
		} else if ok {
			res = append(res, filepath.Join(dir.Name(), n))
		}
	}
	return res, nil
}

// copyFileInto reads filename and copies it into the writer w.
func copyFileInto(w io.Writer, filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(w, f)
	return err
}

func matchExts(filename string, exts ...string) (bool, error) {
	for _, e := range exts {
		if ok, err := filepath.Match(fmt.Sprintf("*.%s", e), filename); err != nil {
			return false, err
		} else if ok {
			return true, nil
		}
	}
	return false, nil
}

// slurpStdin reads stdin fully and saves into a temporary file, whose path name is returned.
func slurpStdin() (string, error) {
	if isatty.IsTerminal(os.Stdin.Fd()) {
		fmt.Fprintf(os.Stderr, "(reading manifests from standard input; hit ctrl-c if this is not what you wanted)\n")
	}

	tmp, err := ioutil.TempFile("", "stdin")
	if err != nil {
		return "", err
	}
	_, err = io.Copy(tmp, os.Stdin)
	if err != nil {
		return "", err
	}
	return tmp.Name(), nil
}

// readFileRunes reads a text file encoded as either UTF-8 or UTF-16, both LE and BE
// (which are the supported encodings of YAML), and return an array of runes which
// we can operate on in order to implement rune-addressed in-place edits.
func readFileRunes(r io.Reader) ([]rune, error) {
	t := unicode.BOMOverride(runes.ReplaceIllFormed())
	return readAllRunes(bufio.NewReader(transform.NewReader(r, t)))
}

func writeFileRunes(filename string, runes []rune) error {
	return ioutil.WriteFile(filename, []byte(string(runes)), 0)
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
