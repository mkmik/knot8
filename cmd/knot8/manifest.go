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

	raw    yaml.Node
	source manifestSource
}

type ObjectMetadata struct {
	Annotations map[string]string `json:"annotations"`
}

type manifestSource struct {
	file      string
	fromStdin bool
	streamPos int // position in yaml stream
}

func parseManifests(f *os.File, fromStdin bool) ([]*Manifest, error) {
	d := yaml.NewDecoder(f)

	var res []*Manifest
	for i := 0; ; i++ {
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
		m.source = manifestSource{
			file:      f.Name(),
			fromStdin: fromStdin,
			streamPos: i,
		}

		res = append(res, &m)
	}
	return res, nil
}
