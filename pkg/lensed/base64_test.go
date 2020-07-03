// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package lensed

import (
	"fmt"
	"testing"
)

func TestBase64(t *testing.T) {
	testCases := []struct {
		src  string
		t    []Mapping
		want string
	}{
		{
			`foo: YmFy`,
			[]Mapping{
				{"/foo/~(base64)", "baz"},
			},
			`foo: YmF6`,
		},
		{
			`foo: Zm9vOiBhCmJhcjogYgo=`,
			[]Mapping{
				{"/foo/~(base64)/~(yaml)/foo", "A"},
			},
			`foo: Zm9vOiBBCmJhcjogYgo=`,
		},
		{
			`foo: ""`,
			[]Mapping{
				{"/foo/~(base64)", "baz"},
			},
			`foo: "YmF6"`,
		},
		{
			`foo: ''`,
			[]Mapping{
				{"/foo/~(base64)", "baz"},
			},
			`foo: 'YmF6'`,
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
