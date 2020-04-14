// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"sort"
	"strings"

	"github.com/mkmik/multierror"
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
			errs = append(errs, err)
		}
	}
	return nil
}

type runeRange struct {
	start int
	end   int
}

// patchFile edits a file in place by replacing each of the given rune ranges in the file
// with a given string value.
func patchFile(filename, value string, positions []runeRange) error {
	backwards := make([]runeRange, len(positions))
	copy(backwards, positions)
	sort.Slice(backwards, func(i, j int) bool { return positions[i].start > positions[j].start })

	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	rvalue := bytes.Runes([]byte(value))
	r := bytes.Runes(b)

	for _, pos := range backwards {
		r = append(r[:pos.start], append(rvalue, r[pos.end:]...)...)
	}

	return ioutil.WriteFile(filename, []byte(string(r)), 0)
}
