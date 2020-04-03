package main

import "fmt"

type Knob struct {
	Name     string
	Pointer  string
	Manifest *Manifest
}

func parseKnobs(manifests []*Manifest) (map[string]Knob, error) {
	return nil, fmt.Errorf("not implemented yet")
}
