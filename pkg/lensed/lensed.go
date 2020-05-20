package lensed

import (
	"fmt"
	"strings"
)

var (
	// Default is the default map of lenses.
	Default = LensMap{
		"":       YAMLLens{},
		"yaml":   YAMLLens{},
		"yamls":  MultiYAMLLens{},
		"base64": Base64Lens{},
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

// Apply applies a slice of mappings on a source byte slice, resolving lens names
// from the lens map.
func (lm LensMap) Apply(src []byte, m []Mapping) ([]byte, error) {
	for i := range m {
		m[i].Pointer = normalize(m[i].Pointer)
	}
	als, err := lm.appliedLenses(m)
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

func (lm LensMap) appliedLenses(ms []Mapping) ([]appliedLens, error) {
	var res []appliedLens
	for _, m := range ms {
		type lensPointer struct {
			lens    Lens
			pointer string
		}

		var pairs []lensPointer
		rest := m.Pointer
		for rest != "" {
			var lens, ptr string
			lens, ptr, rest = split(rest)
			l, ok := lm[lens]
			if !ok {
				return nil, fmt.Errorf("lens %q not defined", lens)
			}

			pairs = append(pairs, lensPointer{l, ptr})
		}

		var value Replacer = leafReplacer([]byte(m.Replacement))
		var a appliedLens
		for i := len(pairs) - 1; i >= 0; i-- {
			a = appliedLens{pairs[i].lens, []Setter{{pairs[i].pointer, value}}}
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

func split(src string) (lens string, pointer string, rest string) {
	if src == "" {
		return "", "", ""
	}
	c := strings.Split(src, "/")
	lens, ok := isLens(c[0])
	if !ok {
		panic(fmt.Errorf("broken promise, %q doesn't start with a lens", src))
	}
	for i := 1; i < len(c); i++ {
		if _, ok := isLens(c[i]); ok {
			if pointer == "" {
				pointer = "/"
			}
			return lens, pointer, strings.Join(c[i:], "/")
		}
		pointer += "/" + c[i]
	}
	return lens, pointer, ""
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
