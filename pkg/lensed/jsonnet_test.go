// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package lensed

import (
	"fmt"
	"testing"
)

func TestJsonnet(t *testing.T) {
	testCases := []struct {
		src  string
		t    []Mapping
		want string
	}{
		{
			`{foo:{bar:"xyz"}}`,
			[]Mapping{
				{"~(jsonnet)/foo/bar", "abc"},
			},
			`{foo:{bar:"abc"}}`,
		},
		{
			`{foo:
  {bar:"xyz"}}`,
			[]Mapping{
				{"~(jsonnet)/foo/bar", "abc"},
			},
			`{foo:
  {bar:"abc"}}`,
		},
		{
			`{foo:["xyz"]}`,
			[]Mapping{
				{"~(jsonnet)/foo/0", "abc"},
			},
			`{foo:["abc"]}`,
		},
		{
			`{foo:[{bar:"xyz"}]}`,
			[]Mapping{
				{"~(jsonnet)/foo/0/bar", "abc"},
			},
			`{foo:[{bar:"abc"}]}`,
		},
		{
			`{foo:[{name:"wrong",bar:"ppp"},{name:"right",bar:"xyz"}]}`,
			[]Mapping{
				{`~(jsonnet)/foo/~{"name":"right"}/bar`, "abc"},
			},
			`{foo:[{name:"wrong",bar:"ppp"},{name:"right",bar:"abc"}]}`,
		},
		{
			`{local a="b",foo:{bar:"xyz"}}`,
			[]Mapping{
				{"~(jsonnet)/foo/bar", "abc"},
			},
			`{local a="b",foo:{bar:"abc"}}`,
		},
		{
			`local a=1; {foo:{bar:"xyz"}}`,
			[]Mapping{
				{"~(jsonnet)/foo/bar", "abc"},
			},
			`local a=1; {foo:{bar:"abc"}}`,
		},
		{
			`{foo:{bar:import "xyz"}}`,
			[]Mapping{
				{"~(jsonnet)/foo/bar/~file", "abc"},
			},
			`{foo:{bar:import "abc"}}`,
		},
		{
			`{foo:{bar:importstr "xyz"}}`,
			[]Mapping{
				{"~(jsonnet)/foo/bar/~file", "abc"},
			},
			`{foo:{bar:importstr "abc"}}`,
		},
		{
			`{foo:{bar:importbin "xyz"}}`,
			[]Mapping{
				{"~(jsonnet)/foo/bar/~file", "abc"},
			},
			`{foo:{bar:importbin "abc"}}`,
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
