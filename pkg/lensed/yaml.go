package lensed

import (
	"fmt"

	yptr "github.com/vmware-labs/yaml-jsonpointer"
	"github.com/vmware-labs/yaml-jsonpointer/yamled"
	"github.com/vmware-labs/yaml-jsonpointer/yamled/splice"
	"golang.org/x/text/transform"
	"gopkg.in/yaml.v3"
)

// YAMLLens implements the "yaml" lens.
type YAMLLens struct{}

// Apply implements the Lens interface.
func (YAMLLens) Apply(src []byte, vals []Setter) ([]byte, error) {
	var root yaml.Node
	if err := yaml.Unmarshal(src, &root); err != nil {
		return nil, err
	}

	var ops []splice.Op
	for _, v := range vals {
		v := v
		f, err := yptr.Find(&root, v.Pointer)
		if err != nil {
			return nil, err
		}

		b, err := v.Value.Transform([]byte(f.Value))
		if err != nil {
			return nil, err
		}
		ops = append(ops, yamled.Node(f).With(string(b)))
	}

	b, _, err := transform.Bytes(yamled.T(ops...), src)
	return b, err
}

// MultiYAMLLens implements the "yamls" lens.
type MultiYAMLLens struct{}

// Apply implements the Lens interface.
func (MultiYAMLLens) Apply(src []byte, m []Setter) ([]byte, error) {
	return nil, fmt.Errorf("not implemented yet")
}
