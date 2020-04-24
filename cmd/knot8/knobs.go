// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mkmik/multierror"
	"gopkg.in/yaml.v3"
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

func (p Pointer) findNode() (*yaml.Node, error) {
	n, err := yptr.Find(&p.Manifest.raw, p.Expr)
	if n.Kind != yaml.ScalarNode {
		return nil, fmt.Errorf("only scalar nodes are supported, found: %s", n.ShortTag())
	}
	return n, err
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

func knobNames(knobs map[string]Knob) []string {
	var names []string
	for k := range knobs {
		names = append(names, k)
	}

	sort.Strings(names)
	return names
}

type knobValue struct {
	value string
	ptr   Pointer
	line  int
	loc   runeRange
}

type knobValues []knobValue

func (s knobValues) Len() int             { return len(s) }
func (s knobValues) Equals(i, j int) bool { return s[i].value == s[j].value }

func getKnob(knobs map[string]Knob, n string) (knobValues, error) {
	k, ok := knobs[n]
	if !ok {
		return nil, fmt.Errorf("knob %q not found", n)
	}

	var (
		errs []error
		res  knobValues
	)
	for _, p := range k.Pointers {
		f, err := p.findNode()
		if err != nil {
			errs = append(errs, err)
			continue
		}

		res = append(res, knobValue{f.Value, p, f.Line, mkRuneRange(f)})
	}
	if errs != nil {
		return nil, multierror.Join(errs)
	}
	return res, nil
}

func setKnob(knobs map[string]Knob, n, v string) error {
	k, ok := knobs[n]
	if !ok {
		return fmt.Errorf("knob %q not found", n)
	}

	updatesByFile := map[*shadowFile][]runeRange{}

	var errs []error
	for _, p := range k.Pointers {
		f, err := p.findNode()
		if err != nil {
			errs = append(errs, err)
			continue
		}
		file := p.Manifest.source.file
		updatesByFile[file] = append(updatesByFile[file], mkRuneRange(f))
	}
	if errs != nil {
		return multierror.Join(errs)
	}

	for f, positions := range updatesByFile {
		if err := f.patch(v, positions); err != nil {
			errs = append(errs, fmt.Errorf("patching file %q: %w", f, err))
		}
	}
	if errs != nil {
		return multierror.Join(errs)
	}
	return nil
}

func mkRuneRange(n *yaml.Node) runeRange {
	// IndexEnd incorrectly includes trailing newline when strings are multiline.
	// TODO(mkm): remove hack once upstream is patched
	d := 0
	if n.Style&(yaml.LiteralStyle|yaml.FoldedStyle) != 0 {
		d = 1
	}
	return runeRange{n.Index, n.IndexEnd - d}
}
