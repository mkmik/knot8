package kptr

import (
	yaml "gopkg.in/yaml.v3"
)

// isTreeSubset returns true if all elements of the json tree a exist in the json tree b.
func isTreeSubset(a, b *yaml.Node) bool {
	if a.Kind == yaml.DocumentNode {
		a = a.Content[0]
	}
	if b.Kind == yaml.DocumentNode {
		b = b.Content[0]
	}

	if a.Kind != b.Kind {
		return false
	}

	if a.Value != b.Value {
		return false
	}

	switch a.Kind {
	case yaml.MappingNode:
		x, y := a.Content, b.Content
		for i := 0; i < len(x); i += 2 {
			keyA, valueA := x[i].Value, x[i+1]
			found := false
			for j := 0; j < len(y); j += 2 {
				keyB, valueB := y[j].Value, y[j+1]
				if keyA == keyB && isTreeSubset(valueA, valueB) {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	case yaml.SequenceNode:
		x, y := a.Content, b.Content
		for i := 0; i < len(x); i++ {
			elA := x[i]
			found := false
			for j := 0; j < len(y); j++ {
				elB := y[j]
				if isTreeSubset(elA, elB) {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}

	return true
}
