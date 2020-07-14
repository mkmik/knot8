// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package lensed

import (
	"fmt"
	"testing"
)

func TestOCIImageRef(t *testing.T) {
	testCases := []struct {
		src  string
		t    []Mapping
		want string
	}{
		{
			"image: foo/bar:baz",
			[]Mapping{
				{"/image/~(ociImageRef)/tag", "quz"},
			},
			"image: foo/bar:quz",
		},
		{
			"image: foo/bar:baz",
			[]Mapping{
				{"/image/~(ociImageRef)/image", "boo/far"},
			},
			"image: boo/far:baz",
		},
		{
			"image: foo/bar",
			[]Mapping{
				{"/image/~(ociImageRef)/tag", "baz"},
			},
			"image: foo/bar:baz",
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			got, err := Default.Apply([]byte(tc.src), tc.t)
			if err != nil {
				t.Fatal(err)
			}

			if got, want := string(got), tc.want; got != want {
				t.Errorf("got: %q, want: %q", got, want)
			}
		})
	}
}
