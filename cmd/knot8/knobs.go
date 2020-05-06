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
	"knot8.io/pkg/yptr/yamled"
)

const (
	annoPrefix   = "field.knot8.io/"
	originalAnno = "knot8.io/original"
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

type Knobs map[string]Knob

func parseKnobs(manifests []*Manifest) (Knobs, error) {
	res := Knobs{}
	var errs []error
	for _, m := range manifests {
		for k, v := range m.Metadata.Annotations {
			if strings.HasPrefix(k, annoPrefix) {
				if err := res.addKnob(m, strings.TrimPrefix(k, annoPrefix), v); err != nil {
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

func (ks Knobs) addKnob(m *Manifest, n, e string) error {
	k := ks[n]
	k.Name = n
	k.Pointers = append(k.Pointers, Pointer{Expr: e, Manifest: m})
	ks[n] = k
	return nil
}

// Names returns a sorted slice of knob names.
func (ks Knobs) Names() []string {
	var names []string
	for n := range ks {
		names = append(names, n)
	}

	sort.Strings(names)
	return names
}

type KnobTarget struct {
	value string
	ptr   Pointer
	line  int
	loc   yamled.Extent
	raw   string
}

func checkKnobValues(values []KnobTarget) bool {
	return allSame(len(values), func(i, j int) bool { return values[i].value == values[j].value })
}

func (ks Knobs) GetAll(n string) ([]KnobTarget, error) {
	k, ok := ks[n]
	if !ok {
		return nil, fmt.Errorf("knob %q not found", n)
	}
	return k.GetAll()
}

func (ks Knobs) GetValue(n string) (string, error) {
	values, err := ks.GetAll(n)
	if err != nil {
		return "", err
	}
	if !checkKnobValues(values) {
		return "", fmt.Errorf("values pointed by field %q are not unique", n)
	}
	return values[0].value, nil
}

func (k Knob) GetAll() ([]KnobTarget, error) {
	var (
		errs []error
		res  []KnobTarget
	)
	for _, p := range k.Pointers {
		f, err := p.findNode()
		if err != nil {
			errs = append(errs, err)
			continue
		}

		loc := yamled.NewExtent(f)
		raw, err := yamled.Extract(p.Manifest.source.reader(), loc)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		res = append(res, KnobTarget{f.Value, p, f.Line, loc, raw[0]})
	}
	if errs != nil {
		return nil, multierror.Join(errs)
	}
	return res, nil
}

// An EditBatch holds requests to edit knobs, added via the Set method
// and performed in the right order by the Commit method.
type EditBatch struct {
	ks    Knobs
	edits map[*shadowFile][]yamled.Replacement

	committed bool
}

func (ks Knobs) NewEditBatch() EditBatch {
	return EditBatch{
		ks:    ks,
		edits: map[*shadowFile][]yamled.Replacement{},
	}
}

func (b EditBatch) Set(n, v string) error {
	if b.committed {
		return fmt.Errorf("batch already committed")
	}

	k, ok := b.ks[n]
	if !ok {
		return fmt.Errorf("knob %q not found", n)
	}

	var errs []error
	for _, p := range k.Pointers {
		f, err := p.findNode()
		if err != nil {
			errs = append(errs, err)
			continue
		}
		file := p.Manifest.source.file
		b.edits[file] = append(b.edits[file], yamled.NewReplacement(v, f))
	}
	if errs != nil {
		return multierror.Join(errs)
	}

	return nil
}

// Commit performs the edits in bulk.
func (b EditBatch) Commit() error {
	var errs []error
	for f, edits := range b.edits {
		up := func(b []byte) ([]byte, error) { return yamled.UpdateBuffer(b, edits...) }
		if err := f.update(up); err != nil {
			errs = append(errs, fmt.Errorf("patching file %q: %w", f, err))
		}
	}
	if errs != nil {
		return multierror.Join(errs)
	}
	b.committed = true
	return nil
}

func allManifests(knobs Knobs) []*Manifest {
	var res []*Manifest
	for _, k := range knobs {
		for _, p := range k.Pointers {
			res = append(res, p.Manifest)
		}
	}
	return res
}

func findOriginal(knobs Knobs) (map[string]string, error) {
	ms := allManifests(knobs)
	for _, m := range ms {
		if o, ok := m.Metadata.Annotations[originalAnno]; ok {
			var res map[string]string
			if err := yaml.Unmarshal([]byte(o), &res); err != nil {
				return nil, err
			}
			return res, nil
		}
	}
	return nil, fmt.Errorf("cannot find %s annotation", originalAnno)
}
