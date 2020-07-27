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
				{"/image/~(oci)/tag", "quz"},
			},
			"image: foo/bar:quz",
		},
		{
			"image: foo/bar:baz",
			[]Mapping{
				{"/image/~(oci)/image", "boo/far"},
			},
			"image: boo/far:baz",
		},
		{
			"image: foo/bar",
			[]Mapping{
				{"/image/~(oci)/tag", "baz"},
			},
			"image: foo/bar:baz",
		},
		{
			"image: foo/bar@sha256:cbbf2f9a99b47fc460d422812b6a5adff7dfee951d8fa2e4a98caa0382cfbdbf",
			[]Mapping{
				{"/image/~(oci)/digest", "7173b809ca12ec5dee4506cd86be934c4596dd234ee82c0662eac04a8c2c71dc"},
			},
			"image: foo/bar@sha256:7173b809ca12ec5dee4506cd86be934c4596dd234ee82c0662eac04a8c2c71dc",
		},
		{
			"image: foo/bar:1.0@sha256:cbbf2f9a99b47fc460d422812b6a5adff7dfee951d8fa2e4a98caa0382cfbdbf",
			[]Mapping{
				{"/image/~(oci)/digest", "7173b809ca12ec5dee4506cd86be934c4596dd234ee82c0662eac04a8c2c71dc"},
			},
			"image: foo/bar:1.0@sha256:7173b809ca12ec5dee4506cd86be934c4596dd234ee82c0662eac04a8c2c71dc",
		},
		{
			"image: foo/bar",
			[]Mapping{
				{"/image/~(oci)/digest", "7173b809ca12ec5dee4506cd86be934c4596dd234ee82c0662eac04a8c2c71dc"},
			},
			"image: foo/bar@sha256:7173b809ca12ec5dee4506cd86be934c4596dd234ee82c0662eac04a8c2c71dc",
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
