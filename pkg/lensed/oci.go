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

// OCIImageRef implements the "ociImageRef" lens.
// The current implementation is very rough and only supports image name and tag.
// The image name cannot contain a port number.
type OCIImageRef struct{}

// Apply implements the Lens interface.
func (OCIImageRef) Apply(src []byte, vals []Setter) ([]byte, error) {
	var ops []splice.Op

	for _, v := range vals {
		p, err := jsonpointer.New(v.Pointer)
		if err != nil {
			return nil, err
		}
		path := p.DecodedTokens()
		if l, m := len(path), 1; l != m {
			return nil, fmt.Errorf("unexpected path len. got: %d, max: %d", l, m)
		}

		r, err := regexp.Compile("^([^:]*)(:(.*))?$")
		if err != nil {
			return nil, err
		}
		indices := r.FindSubmatchIndex(src)

		var comp int
		switch p := path[0]; p {
		case "image":
			comp = 1
		case "tag":
			comp = 3
		default:
			return nil, fmt.Errorf("unknown ociImageRef field %q", p)
		}

		start, end := indices[2*comp+0], indices[2*comp+1]

		var oldval []byte
		if start == -1 {
			oldval = []byte("latest")
		} else {
			oldval = src[start:end]
		}
		newval, err := v.Value.Transform(oldval)
		if err != nil {
			return nil, err
		}

		if start == -1 {
			start, end = indices[1], indices[1]
			newval = append([]byte(":"), newval...)
		}

		ops = append(ops, splice.Span(start, end).With(string(newval)))
	}
	b, _, err := transform.Bytes(splice.T(ops...), src)
	return b, err
}
