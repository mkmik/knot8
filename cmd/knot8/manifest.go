// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package main

import (
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

type Manifest struct {
	APIVersion string         `yaml:"apiVersion"`
	Kind       string         `yaml:"kind"`
	Metadata   ObjectMetadata `yaml:"metadata"`

	file      string
	raw       yaml.Node
	fromStdin bool
}

type ObjectMetadata struct {
	Annotations map[string]string `json:"annotations"`
}

func parseManifests(f *os.File, fromStdin bool) ([]*Manifest, error) {
	d := yaml.NewDecoder(f)

	var res []*Manifest
	for {
		var n yaml.Node
		if err := d.Decode(&n); err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}

		var m Manifest
		if err := n.Decode(&m); err != nil {
			return nil, err
		}
		m.raw = n
		m.file = f.Name()
		m.fromStdin = fromStdin

		res = append(res, &m)

	}
	return res, nil
}
