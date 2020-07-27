// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package lensed

import (
	"fmt"
	"testing"
)

func TestRegexp(t *testing.T) {
	src := `data: |
  foo
    bar 123
  baz
`
	out := `data: |
  foo
    bar 023
  baz
`

	testCases := []struct {
		src  string
		t    []Mapping
		want string
	}{
		{
			src,
			[]Mapping{
				{"/data/~(regexp)/b.* ([0-9])", "bar 0"},
			},
			out,
		},
		{
			src,
			[]Mapping{
				{"/data/~(regexp)/b.* (?P<num>[0-9])/1", "0"},
			},
			out,
		},
		{
			src,
			[]Mapping{
				{"/data/~(regexp)/b.* (?P<num>[0-9])/num", "0"},
			},
			out,
		},
		{
			"foo:YmFy",
			[]Mapping{
				{"~(regexp)/foo:(.*)/1/~(base64)", "baz"},
			},
			"foo:YmF6",
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
