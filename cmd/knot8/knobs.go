// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package main

import (
	"strings"

	"github.com/mkmik/multierror"
)

const (
	annoPrefix = "knot8.vmware.com/"
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
	k.Pointers = append(k.Pointers, Pointer{Expr: e, Manifest: m})
	r[n] = k
	return nil
}
