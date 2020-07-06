// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package main

// allSame returns true if all elements of a sequence of length l are the same.
// The equality of the elements of the slice is evaluated via a caller supplied predicate,
// p that must returns true the ith and the jth element are the same.
func allSame(l int, p func(i, j int) bool) bool {
	for i := 1; i < l; i++ {
		if !p(0, i) {
			return false
		}
	}
	return true
}
