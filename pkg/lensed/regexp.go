// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package lensed

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/go-openapi/jsonpointer"
	"github.com/vmware-labs/go-yaml-edit/splice"
	"golang.org/x/text/transform"
)

// RegexpLens implements the "regexp" lens.
// The regexp find a selection matching a regexp and exposes
// capturing groups as child nodes.
// Child nodes can be referenced by number or by name
// if the subexpressions is named with the naming syntax "(?P<name>re)").
type RegexpLens struct{}

// Apply implements the Lens interface.
func (RegexpLens) Apply(src []byte, vals []Setter) ([]byte, error) {
	var ops []splice.Op

	for _, v := range vals {
		p, err := jsonpointer.New(v.Pointer)
		if err != nil {
			return nil, err
		}
		path := p.DecodedTokens()
		if l, m := len(path), 2; l > m {
			return nil, fmt.Errorf("unexpected path len. got: %d, max: %d", l, m)
		}

		r, err := regexp.Compile(path[0])
		if err != nil {
			return nil, err
		}

		subn := 0
		if len(path) == 2 {
			sub := path[1]
			subn, err = strconv.Atoi(sub)
			if err != nil {
				sn := r.SubexpNames()
				for i := 0; i < len(sn); i++ {
					if sub == sn[i] {
						subn = i
						break
					}
				}
				if subn == 0 {
					return nil, fmt.Errorf("cannot find subexpression %q", sub)
				}
			}
		}

		matches := r.FindAllIndex(src, -1)
		if l, m := len(matches), 1; l > m {
			return nil, fmt.Errorf("found %d matches, max %d match supported", l, m)
		}
		if len(matches) == 0 {
			return nil, fmt.Errorf("no matches for regexp %q found", path[0])
		}
		loc := r.FindSubmatchIndex(src)
		start, end := loc[2*subn+0], loc[2*subn+1]

		oldval := src[start:end]
		newval, err := v.Value.Transform(oldval)
		if err != nil {
			return nil, err
		}

		ops = append(ops, splice.Span(start, end).With(string(newval)))
	}
	b, _, err := transform.Bytes(splice.T(ops...), src)
	return b, err
}