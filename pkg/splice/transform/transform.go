// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

/*
Package transform is a temporary pseudo-drop-in replacement for golang.org/x/text/transform
until we make the splice transformer compatible with golang.org/x/text/transform.Transform
*/
package transform

import (
	"bytes"
	"io"
	"strings"
)

type Transformer interface {
	Transform(w io.Writer, r io.ReadSeeker) error
}

func String(t Transformer, s string) (string, int, error) {
	var out strings.Builder
	r := strings.NewReader(s)
	if err := t.Transform(&out, r); err != nil {
		return "", 0, err
	}
	return out.String(), out.Len(), nil
}

func Bytes(t Transformer, b []byte) ([]byte, int, error) {
	var out bytes.Buffer
	r := bytes.NewReader(b)
	if err := t.Transform(&out, r); err != nil {
		return nil, 0, err
	}
	return out.Bytes(), out.Len(), nil
}
