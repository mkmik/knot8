// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mkmik/multierror"
	"gopkg.in/yaml.v3"
	"knot8.io/pkg/lensed"
)

const (
	annoDomain   = "knot8.io"
	annoPrefix   = "field.knot8.io/"
	originalAnno = "knot8.io/original"
)

type ManifestSet struct {
	Manifests Manifests
	Fields    Fields
}

type Field struct {
	Name     string
	Pointers []Pointer
}

type Pointer struct {
	Expr     string
	Manifest *Manifest
}

func (p Pointer) Abs() string {
	return fmt.Sprintf("~(yamls)/%d%s", p.Manifest.source.streamPos, p.Expr)
}

type Fields map[string]Field

func parseFields(manifests []*Manifest) (Fields, error) {
	res := Fields{}
	var errs []error
	for _, m := range manifests {
		for k, v := range m.Metadata.Annotations {
			if strings.HasPrefix(k, annoPrefix) {
				if err := res.addField(m, strings.TrimPrefix(k, annoPrefix), v); err != nil {
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

func (ks Fields) addField(m *Manifest, n, e string) error {
	k := ks[n]
	k.Name = n
	k.Pointers = append(k.Pointers, Pointer{Expr: e, Manifest: m})
	ks[n] = k
	return nil
}

// Names returns a sorted slice of field names.
func (ks Fields) Names() []string {
	var names []string
	for n := range ks {
		names = append(names, n)
	}

	sort.Strings(names)
	return names
}

// Rebase updates the manifest pointer inside each Pointer value
// so that it points to the manifest for the matching resource (namespace+name)
func (ks Fields) Rebase(dst Manifests) error {
	nn := map[FQN]*Manifest{}
	for _, m := range dst {
		nn[m.FQN()] = m
	}
	var errs []error
	for n := range ks {
		for i, p := range ks[n].Pointers {
			r := p.Manifest.FQN()
			d, found := nn[r]
			if !found {
				errs = append(errs, fmt.Errorf("cannot find resource %s", r))
				continue
			}
			u := ks[n]
			u.Pointers[i].Manifest = d
			ks[n] = u
		}
	}
	if errs != nil {
		return multierror.Join(errs)
	}
	return nil
}

// MergeSchema merges the field definitions from other into the receiver.
func (ks Fields) MergeSchema(other Fields) {
	for n := range other {
		k := ks[n]

		ptrs := map[Pointer]struct{}{}
		for _, p := range k.Pointers {
			ptrs[p] = struct{}{}
		}
		for _, p := range other[n].Pointers {
			if _, found := ptrs[p]; !found {
				k.Pointers = append(k.Pointers, p)
			}
		}

		ks[n] = k
	}
}

type FieldTarget struct {
	value string
	ptr   Pointer
}

func checkFieldValues(values []FieldTarget) bool {
	return allSame(len(values), func(i, j int) bool { return values[i].value == values[j].value })
}

func (ks Fields) GetAll(n string) ([]FieldTarget, error) {
	k, ok := ks[n]
	if !ok {
		return nil, fmt.Errorf("field %q not found", n)
	}
	return k.GetAll()
}

func (ks Fields) GetValue(n string) (string, error) {
	values, err := ks.GetAll(n)
	if err != nil {
		return "", err
	}
	if !checkFieldValues(values) {
		return "", fmt.Errorf("values pointed by field %q are not unique", n)
	}
	return values[0].value, nil
}

func (k Field) GetAll() ([]FieldTarget, error) {
	var (
		errs []error
		res  []FieldTarget
	)
	for _, p := range k.Pointers {
		r, err := lensed.Get(p.Manifest.source.file.buf, []string{p.Abs()})
		if err != nil {
			errs = append(errs, err)
			continue
		}
		v := string(r[0])
		res = append(res, FieldTarget{ptr: p, value: v})
	}
	if errs != nil {
		return nil, multierror.Join(errs)
	}
	return res, nil
}

// An EditBatch holds requests to edit fields, added via the Set method
// and performed in the right order by the Commit method.
type EditBatch struct {
	ks    Fields
	edits map[*shadowFile][]lensed.Mapping

	committed bool
}

func (ks Fields) NewEditBatch() EditBatch {
	return EditBatch{
		ks:    ks,
		edits: map[*shadowFile][]lensed.Mapping{},
	}
}

func (b EditBatch) Set(n, v string) error {
	if b.committed {
		return fmt.Errorf("batch already committed")
	}

	k, ok := b.ks[n]
	if !ok {
		return fmt.Errorf("field %q not found", n)
	}

	var errs []error
	for _, p := range k.Pointers {
		file := p.Manifest.source.file
		b.edits[file] = append(b.edits[file], lensed.Mapping{p.Abs(), v})
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
		if b, err := lensed.Apply(f.buf, edits); err != nil {
			errs = append(errs, fmt.Errorf("patching file %q: %w", f, err))
		} else {
			f.buf = b
		}
	}
	if errs != nil {
		return multierror.Join(errs)
	}
	b.committed = true
	return nil
}

func findOriginal(ms *ManifestSet) (map[string]string, error) {
	for _, m := range ms.Manifests {
		if o, ok := m.Metadata.Annotations[originalAnno]; ok {
			var res map[string]string
			if err := yaml.Unmarshal([]byte(o), &res); err != nil {
				return nil, err
			}
			return res, nil
		}
	}
	return map[string]string{}, nil
}
