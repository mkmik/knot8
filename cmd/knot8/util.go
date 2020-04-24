// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package main

import (
	"fmt"
	"strings"
)

// reindent moves a block of text leftwards until the first line is not indented
// and then adds l indentation spaces to the whole block.
func reindent(s string, i int) string {
	f := strings.IndexFunc(s, func(r rune) bool { return r != ' ' })
	s = trimIndent(s, f)
	return addIndent(s, i)
}

// trimIndent removes i levels of indentation from string s.
func trimIndent(s string, i int) string {
	return mapLines(s, func(l string) string {
		return l[i:]
	})
}

// addIndent adds i levels of indentation to string i.
func addIndent(s string, i int) string {
	return mapLines(s, func(l string) string {
		return fmt.Sprintf("%s%s", strings.Repeat(" ", i), l)
	})
}

// mapLines invokes function on every line in s.
func mapLines(s string, f func(string) string) string {
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = f(lines[i])
	}
	return strings.Join(lines, "\n")
}

type sliceEq interface {
	Len() int
	Equals(i, j int) bool
}

// allSame returns true if all elements of a sequence of lenght l are the same.
// The equality of the elements of the slice is evaluated via a caller supplied predicate,
// p that must returns true the ith and the jth element are the same.
func allSame(iface sliceEq) bool {
	l := iface.Len()
	for i := 1; i < l; i++ {
		if !iface.Equals(0, i) {
			return false
		}
	}
	return true
}

type sliceEqS struct {
	l int
	p func(i, j int) bool
}

// sliceEqFunc implements a sliceEq interface using an explicit length and a callback.
func sliceEqFunc(l int, p func(i, j int) bool) sliceEqS {
	return sliceEqS{l, p}
}

func (s sliceEqS) Len() int             { return s.l }
func (s sliceEqS) Equals(i, j int) bool { return s.p(i, j) }
