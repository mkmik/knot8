// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package main

import (
	"fmt"
	"reflect"
	"testing"
)

func TestExpandPaths(t *testing.T) {
	testCases := []struct {
		paths    []string
		expanded []string
	}{
		{[]string{"testdata/exf/m1.yaml"}, []string{"testdata/exf/m1.yaml"}},
		{[]string{"testdata/exf/*.yaml"}, []string{"testdata/exf/m1.yaml", "testdata/exf/m2.yaml"}},
		{[]string{"testdata/exf/d1"}, []string{"testdata/exf/d1/m3.yaml", "testdata/exf/d1/m4.yaml"}},
		{[]string{"-"}, []string{"-"}},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			got, err := expandPaths(tc.paths)
			if err != nil {
				t.Fatal(err)
			}
			if want := tc.expanded; !reflect.DeepEqual(got, want) {
				t.Errorf("got: %q, want: %q", got, want)
			}
		})
	}
}
