// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package yptr

import (
	"fmt"
	"testing"

	yaml "gopkg.in/yaml.v3"
)

func TestIsTreeSubset(t *testing.T) {
	testCases := []struct {
		a  string
		b  string
		ok bool
	}{
		{`1`, `1`, true},
		{`1`, `2`, false},
		{`"a"`, `"a"`, true},
		{`"a"`, `"b"`, false},
		{`{"a":"b"}`, `{"a":"b","c":d"}`, true},
		{`{"a":"b"}`, `{"c":d","a":"b"}`, true},
		{`{"a":"x"}`, `{"a":"b","c":d"}`, false},
		{`{"a":"b","c":d"}`, `{"a":"b"}`, false},
		{`{"a":{"b": 1}}`, `{"a":{"b": 1}}`, true},
		{`{"a":{"b": 1}}`, `{"a":{"b": 2}}`, false},
		{`{"a":{"b": 1}}`, `{"a":{"b": 1, "c": 2}}`, true},
		{`[0]`, `[0]`, true},
		{`[0, 0]`, `[0, 0]`, true},
		{`[0]`, `[0, 1]`, true},
		{`[1]`, `[0, 1]`, true},
		{`[0, 0]`, `[0, 1]`, true},
		{`[1, 1]`, `[1, 0]`, true},
		{`[1, 1]`, `[1, 1]`, true},
		{`[1, 1]`, `[0, 1]`, true},
		{`[0]`, `[1, 2]`, false},
		{`{"a":{"b": [1]}}`, `{"a":{"b": [0,1,2], "c": 2}}`, true},
		{`{"a":{"b": [1]}}`, `{"a":{"b": [0,2], "c": 2}}`, false},

		{`[0,2]`, `[0,1,2]`, true},
		{`{"a":{"b":1}}`, `{"x":2, "a":{"c":4, "b":1, "d":5}, "y":1}`, true}, // from isTreeSubset doc comment.
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			var a, b yaml.Node
			if err := yaml.Unmarshal([]byte(tc.a), &a); err != nil {
				t.Fatal(err)
			}
			if err := yaml.Unmarshal([]byte(tc.b), &b); err != nil {
				t.Fatal(err)
			}

			if got, want := isTreeSubset(&a, &b), tc.ok; got != want {
				t.Errorf("got: %v, want: %v", got, want)
			}
		})
	}
}
