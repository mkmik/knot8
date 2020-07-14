// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package lensed

import (
	"fmt"
	"strings"
)

var (
	// Default is the default map of lenses.
	Default = LensMap{
		"":            YAMLLens{},
		"yaml":        YAMLLens{},
		"yamls":       MultiYAMLLens{},
		"toml":        TOMLLens{},
		"base64":      Base64Lens{},
		"line":        LineLens{},
		"regexp":      RegexpLens{},
		"ociImageRef": OCIImageRef{},
	}
)

// A Mapping is a request to replace the value pointed by pointer with a replacement string.
type Mapping struct {
	Pointer     string
	Replacement string
}

// A Setter is like a mapping but uses a Replacer to update the existing value pointed by the pointer.
type Setter struct {
	Pointer string
	Value   Replacer
}

// A Lens knows how to perform in-place edits of parts of a text.
// The parts are addressed using JSONPointer pointers.
// The new value is provided via a transformer which allows, among other things, to nest lenses.
type Lens interface {
	Apply(src []byte, m []Setter) ([]byte, error)
}

// A Replacer transforms a byte slice into another byte slice.
type Replacer interface {
	Transform(src []byte) ([]byte, error)
}

// A LensMap is a collection of named lenses.
type LensMap map[string]Lens

// Apply invokes Apply on the Default lens map.
func Apply(src []byte, m []Mapping) ([]byte, error) {
	return Default.Apply(src, m)
}

// Apply applies a slice of mappings on a source byte slice, resolving lens names
// from the lens map.
func (lm LensMap) Apply(src []byte, ms []Mapping) ([]byte, error) {
	var setters []Setter
	for _, m := range ms {
		setters = append(setters, Setter{m.Pointer, leafReplacer([]byte(m.Replacement))})
	}
	return lm.ApplySetters(src, setters)
}

func (lm LensMap) ApplySetters(src []byte, setters []Setter) ([]byte, error) {
	als, err := lm.appliedLenses(setters)
	if err != nil {
		return nil, err
	}

	for _, t := range als {
		src, err = t.Transform(src)
		if err != nil {
			return nil, err
		}
	}
	return src, nil
}

// Get invokes Get on the Default lens map.
func Get(src []byte, ptrs []string) ([][]byte, error) {
	return Default.Get(src, ptrs)
}

// Get returns a slice of byte slices, each one containing the value of a field
// addressed by a pointer.
func (lm LensMap) Get(src []byte, ptrs []string) ([][]byte, error) {
	res := make([][]byte, len(ptrs))
	setters := make([]Setter, len(ptrs))
	for i, p := range ptrs {
		setters[i] = Setter{p, captureReplacer{&res[i]}}
	}
	_, err := lm.ApplySetters(src, setters)
	if err != nil {
		return nil, err
	}
	return res, err
}

func (lm LensMap) appliedLenses(setters []Setter) ([]appliedLens, error) {
	var res []appliedLens
	for _, setter := range setters {
		type lensPointer struct {
			lens    Lens
			pointer string
		}

		var pairs []lensPointer
		rest := normalize(setter.Pointer)
		for rest != "" {
			var (
				lens, ptr string
				err       error
			)
			lens, ptr, rest, err = split(rest)
			if err != nil {
				return nil, err
			}
			l, ok := lm[lens]
			if !ok {
				return nil, fmt.Errorf("lens %q not defined", lens)
			}

			pairs = append(pairs, lensPointer{l, ptr})
		}

		var value Replacer = setter.Value
		var a appliedLens
		for j := len(pairs) - 1; j >= 0; j-- {
			a = appliedLens{pairs[j].lens, []Setter{{pairs[j].pointer, value}}}
			value = a
		}
		res = append(res, a)
	}

	return res, nil
}

// Replacer returns a Replacer implementation that applies mappings to its input.
func (lm LensMap) Replacer(ms []Mapping) AppliedLensMap {
	return AppliedLensMap{lm, ms}
}

// An AppliedLensMap is a Replacer that applied mappings to its inputs.
type AppliedLensMap struct {
	lm LensMap
	ms []Mapping
}

// Transform implements the Replacer interface.
func (a AppliedLensMap) Transform(src []byte) ([]byte, error) {
	return a.lm.Apply(src, a.ms)
}

// normalize normalizes pointer expressions.
func normalize(ptr string) string {
	if ptr == "" {
		return "~()/"
	}
	if strings.HasPrefix(ptr, "/") {
		ptr = fmt.Sprintf("~()%s", ptr)
	}

	ptr = strings.TrimSuffix(ptr, "/")
	if s := strings.Split(ptr, "/"); strings.HasPrefix(s[len(s)-1], "~(") {
		ptr += "/"
	}
	return ptr
}

func isLens(s string) (string, bool) {
	if strings.HasPrefix(s, "~(") {
		return strings.TrimPrefix(strings.TrimSuffix(s, ")"), "~("), true
	}
	return s, false
}

func split(src string) (lens string, pointer string, rest string, err error) {
	if src == "" {
		return "", "", "", nil
	}
	c := strings.Split(src, "/")
	lens, ok := isLens(c[0])
	if !ok {
		return "", "", "", fmt.Errorf("broken promise, %q doesn't start with a lens", src)
	}
	for i := 1; i < len(c); i++ {
		if _, ok := isLens(c[i]); ok {
			if pointer == "" {
				pointer = "/"
			}
			return lens, pointer, strings.Join(c[i:], "/"), nil
		}
		pointer += "/" + c[i]
	}
	return lens, pointer, "", nil
}

type appliedLens struct {
	lens    Lens
	setters []Setter
}

func (a appliedLens) Transform(src []byte) ([]byte, error) {
	return a.lens.Apply(src, a.setters)
}

type leafReplacer []byte

func (l leafReplacer) Transform(src []byte) ([]byte, error) {
	return l, nil
}

type captureReplacer struct{ b *[]byte }

func (c captureReplacer) Transform(src []byte) ([]byte, error) {
	*c.b = src
	return src, nil
}
