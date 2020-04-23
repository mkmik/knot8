// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package main

import (
	"fmt"
	"testing"
)

func TestAllSameString(t *testing.T) {
	testCases := []struct {
		ok bool
		s  []string
	}{
		{true, []string{"a", "a", "a"}},
		{true, []string{"a"}},
		{true, []string{}},
		{true, []string{"", ""}},
		{false, []string{"a", "b"}},
		{false, []string{"a", "a", "b"}},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			ok := allSame(len(tc.s), func(i, j int) bool { return tc.s[i] == tc.s[j] })
			if got, want := ok, tc.ok; got != want {
				t.Errorf("got: %v, want: %v", got, want)
			}
		})
	}
}
