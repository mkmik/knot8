// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package lensed

import (
	"fmt"
	"testing"
)

func TestTOML(t *testing.T) {
	testCases := []struct {
		src  string
		t    []Mapping
		want string
	}{
		{
			`foo: |
  [s1]
  k1 =  "v1" # a comment
  k2 = "v2"
`,
			[]Mapping{
				{"/foo/~(toml)/s1/k1", "V1"},
			},
			`foo: |
  [s1]
  k1 =  "V1" # a comment
  k2 = "v2"
`,
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
