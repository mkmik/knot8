// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"gopkg.in/yaml.v3"
)

type Manifest struct {
	VersionKind `yaml:",inline"`
	Metadata    ObjectMetadata `yaml:"metadata"`

	raw    yaml.Node
	source manifestSource
}

func (m Manifest) FQN() FQN {
	return FQN{m.VersionKind, m.Metadata.NamespacedName}
}

type VersionKind struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
}

type ObjectMetadata struct {
	NamespacedName `yaml:",inline"`
	Annotations    map[string]string `json:"annotations"`
}

type NamespacedName struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

// FQN is a fully qualified name
type FQN struct {
	VersionKind
	NamespacedName
}

func (f FQN) String() string {
	b, _ := json.Marshal(f)
	return string(b)
}

type manifestSource struct {
	file      *shadowFile
	streamPos int // position in yaml stream
}

func parseManifests(f *shadowFile) (Manifests, error) {
	d := yaml.NewDecoder(bytes.NewReader([]byte(string(f.buf))))

	var res []*Manifest
	for i := 0; ; i++ {
		var m Manifest
		if err := d.Decode(&m.raw); err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("parsing %q: %w", f.name, err)
		}

		if err := m.raw.Decode(&m); err != nil {
			return nil, fmt.Errorf("parsing %q: %w", f.name, err)
		}
		// Skip YAML files that are not K8s manifests.
		if m.APIVersion == "" && m.Kind == "" {
			continue
		}

		for k := range m.Metadata.Annotations {
			if !isOurAnnotation(k) {
				delete(m.Metadata.Annotations, k)
			}
		}

		m.source = manifestSource{
			file:      f,
			streamPos: i,
		}

		res = append(res, &m)
	}
	return res, nil
}

type Manifests []*Manifest

// Commit saves changes made to the manifests
func (ms Manifests) Commit() error {
	uniq := map[*shadowFile]struct{}{}

	var errs []error
	for _, m := range ms {
		if _, found := uniq[m.source.file]; found {
			continue
		}
		uniq[m.source.file] = struct{}{}

		if err := m.source.file.Commit(); err != nil {
			errs = append(errs, err)
		}
	}

	if errs != nil {
		return errors.Join(errs...)
	}
	return nil
}

// Intersect return the set of manifests in the receiver that
// also exist in src. Equality is matched used the FQN method.
func (ms Manifests) Intersect(src Manifests) Manifests {
	exists := map[FQN]bool{}
	for _, s := range src {
		exists[s.FQN()] = true
	}

	var res Manifests
	for _, d := range ms {
		if exists[d.FQN()] {
			res = append(res, d)
		}
	}
	return res
}

// MergeAnnotations copies annotations from the src into ms if they exist
func (ms Manifests) MergeAnnotations(src Manifests) {
	anns := map[FQN]map[string]string{}
	for _, s := range src {
		anns[s.FQN()] = s.Metadata.Annotations
	}

	for _, d := range ms {
		for k, v := range anns[d.FQN()] {
			if isOurAnnotation(k) {
				if d.Metadata.Annotations == nil {
					d.Metadata.Annotations = map[string]string{}
				}
				d.Metadata.Annotations[k] = v
			}
		}
	}
}

func isOurAnnotation(a string) bool {
	c := strings.SplitN(a, "/", 2)
	return strings.HasSuffix(c[0], annoDomain)
}
