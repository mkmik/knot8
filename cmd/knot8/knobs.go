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
	"sort"
	"strings"

	"github.com/mkmik/multierror"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"knot8.io/pkg/yptr"
)

const (
	annoPrefix = "field.knot8.io/"
)

type Knob struct {
	Name     string
	Pointers []Pointer
}

type Pointer struct {
	Expr     string
	Manifest *Manifest
}

func parseKnobs(manifests []*Manifest) (map[string]Knob, error) {
	res := map[string]Knob{}
	var errs []error
	for _, m := range manifests {
		for k, v := range m.Metadata.Annotations {
			if strings.HasPrefix(k, annoPrefix) {
				if err := addKnob(res, m, strings.TrimPrefix(k, annoPrefix), v); err != nil {
					errs = append(errs, err)
				}
			}
		}
	}
	if errs != nil {
		return nil, multierror.Join(errs)
	}
	return res, nil
}

func addKnob(r map[string]Knob, m *Manifest, n, e string) error {
	k := r[n]
	k.Name = n
	k.Pointers = append(k.Pointers, Pointer{Expr: e, Manifest: m})
	r[n] = k
	return nil
}

func setKnob(knobs map[string]Knob, n, v string) error {
	k, ok := knobs[n]
	if !ok {
		return fmt.Errorf("knob %q not found", n)
	}

	updates := map[string][]runeRange{}

	var errs []error
	for _, p := range k.Pointers {
		f, err := yptr.Find(&p.Manifest.raw, p.Expr)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		updates[p.Manifest.file] = append(updates[p.Manifest.file], runeRange{f.Index, f.IndexEnd})
	}
	if errs != nil {
		return multierror.Join(errs)
	}

	for f, positions := range updates {
		if err := patchFile(f, v, positions); err != nil {
			errs = append(errs, fmt.Errorf("patching file %q: %w", f, err))
		}
	}
	if errs != nil {
		return multierror.Join(errs)
	}
	return nil
}

type runeRange struct {
	start int
	end   int
}

// patchFile edits a file in place by replacing each of the given rune ranges in the file
// with a given string value.
//
// All valid yaml encodings are supported (UTF-8, UTF16-LE, UTF16-BE) but the input
// encoding is not currently preserved when writing the output file.
func patchFile(filename, value string, positions []runeRange) error {
	backwards := make([]runeRange, len(positions))
	copy(backwards, positions)
	sort.Slice(backwards, func(i, j int) bool { return positions[i].start > positions[j].start })

	r, err := readFileRunes(filename)
	if err != nil {
		return err
	}
	rvalue := bytes.Runes([]byte(value))

	for _, pos := range backwards {
		r = append(r[:pos.start], append(rvalue, r[pos.end:]...)...)
	}

	return writeFileRunes(filename, r)
}

// readFileRunes reads a text file encoded as either UTF-8 or UTF-16, both LE and BE
// (which are the supported encodings of YAML), and return an array of runes which
// we can operate on in order to implement rune-addressed in-place edits.
func readFileRunes(filename string) ([]rune, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	t := unicode.BOMOverride(runes.ReplaceIllFormed())
	r := bufio.NewReader(transform.NewReader(f, t))

	return readAllRunes(r)
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
