// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package lensed

import (
	"fmt"
	"regexp"

	"github.com/go-openapi/jsonpointer"
	"github.com/vmware-labs/go-yaml-edit/splice"
	"golang.org/x/text/transform"
)

// LineLens implements the "line" lens.
// The line lens selects a line matching a regexp.
// Like awk's or sed's "/regexp/" construct.
type LineLens struct{}

// Apply implements the Lens interface.
func (LineLens) Apply(src []byte, vals []Setter) ([]byte, error) {
	var ops []splice.Op

	for _, v := range vals {
		p, err := jsonpointer.New(v.Pointer)
		if err != nil {
			return nil, err
		}
		path := p.DecodedTokens()
		if got, want := len(path), 1; got != want {
			return nil, fmt.Errorf("unexpected path len. got: %d, want: %d", got, want)
		}

		r, err := regexp.Compile(fmt.Sprintf(".*%s.*", path[0]))
		if err != nil {
			return nil, err
		}
		matches := r.FindAllIndex(src, -1)
		if len(matches) > 1 {
			return nil, fmt.Errorf("multiple lines match, not supported")
		}
		if len(matches) == 0 {
			return nil, fmt.Errorf("regexp %q didn't match any line", path[0])
		}
		loc := matches[0]
		start, end := loc[0], loc[1]
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