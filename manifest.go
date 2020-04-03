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

	file string
	raw  interface{}
}

type ObjectMetadata struct {
	Annotations map[string]string `json:"annotations"`
}

func parseManifests(f *os.File) ([]*Manifest, error) {
	d := yaml.NewDecoder(f)

	var res []*Manifest
	for {
		var i interface{}
		if err := d.Decode(&i); err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}

		// quick&dirty way to map an in-memory json tree back to a typed Go struct.
		tmp, err := yaml.Marshal(i)
		if err != nil {
			return nil, err
		}
		var m Manifest
		if err := yaml.Unmarshal(tmp, &m); err != nil {
			return nil, err
		}
		m.raw = i
		m.file = f.Name()

		res = append(res, &m)

	}
	return res, nil
}
