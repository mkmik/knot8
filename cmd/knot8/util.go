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
