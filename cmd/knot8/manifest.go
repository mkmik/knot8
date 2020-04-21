// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package main

import (
	"bytes"
	"io"
	"os"

	"github.com/mkmik/multierror"
	"gopkg.in/yaml.v3"
)

type Manifest struct {
	APIVersion string         `yaml:"apiVersion"`
	Kind       string         `yaml:"kind"`
	Metadata   ObjectMetadata `yaml:"metadata"`

	raw    yaml.Node
	source manifestSource
}

type ObjectMetadata struct {
	Annotations map[string]string `json:"annotations"`
}

type manifestSource struct {
	file      shadowFile
	fromStdin bool
	streamPos int // position in yaml stream
}

func parseManifests(f shadowFile, fromStdin bool) (Manifests, error) {
	d := yaml.NewDecoder(bytes.NewReader([]byte(string(f.buf))))

	var res []*Manifest
	for i := 0; ; i++ {
		var m Manifest
		if err := d.Decode(&m.raw); err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}

		if err := m.raw.Decode(&m); err != nil {
			return nil, err
		}
		m.source = manifestSource{
			file:      f,
			fromStdin: fromStdin,
			streamPos: i,
		}

		res = append(res, &m)
	}
	return res, nil
}

func (m *Manifest) Commit() error {
	if err := m.source.file.Commit(); err != nil {
		return err
	}

	if m.source.fromStdin {
		return copyFileInto(os.Stdout, m.source.file.name)
	}
	return nil
}

type Manifests []*Manifest

// Commit saves changes made to the manifests
func (ms Manifests) Commit() error {
	var errs []error
	for _, m := range ms {
		if err := m.Commit(); err != nil {
			errs = append(errs, err)
		}
	}
	if errs != nil {
		return multierror.Join(errs)
	}
	return nil
}
