// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package splice_test

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"strings"
	"testing"

	"golang.org/x/text/transform"
	"knot8.io/pkg/atomicfile"
	"knot8.io/pkg/splice"
)

func ExampleOp() {
	fmt.Printf("%T", splice.Span(3, 4).With("foo"))
	// Output:
	// splice.Op
}

func Example() {
	src := "abcd"

	res, _, err := transform.String(splice.T(splice.Span(1, 2).With("B")), src)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(res)

	// Output:
	// aBcd
}

func Example_multiple() {
	src := "abcd"

	t := splice.T(
		splice.Span(1, 2).With("B"),
		splice.Span(3, 4).With("D"),
	)
	aBcD, _, err := transform.String(t, src)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(aBcD)

	t = splice.T(
		splice.Span(1, 2).With("Ba"),
		splice.Span(2, 3).With(""),
		splice.Span(3, 4).With("Da"),
	)
	aBaDa, _, err := transform.String(t, src)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(aBaDa)

	// Output:
	// aBcD
	// aBaDa
}

func Example_insert() {
	src := "abcd"

	t := splice.T(splice.Span(2, 2).With("X"))
	res, _, err := transform.String(t, src)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(res)

	// Output:
	// abXcd
}

func Example_delete() {
	src := "abcd"

	t := splice.T(splice.Span(2, 3).With(""))
	res, _, err := transform.String(t, src)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(res)

	// Output:
	// abd
}

func ExampleBytes() {
	buf := []byte("abcd")

	t := splice.T(splice.Span(1, 2).With("B"))
	aBcd, _, err := transform.Bytes(t, buf)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(aBcd))

	// Output:
	// aBcd
}

func Example_file() {
	tmp, err := ioutil.TempFile("", "")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tmp.Name())

	fmt.Fprintf(tmp, "abcd")
	tmp.Close()

	t := splice.T(splice.Span(1, 3).With("X"))
	if err := atomicfile.Transform(t, tmp.Name()); err != nil {
		log.Fatal(err)
	}

	b, err := ioutil.ReadFile(tmp.Name())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(b))

	// Output:
	// aXd
}

func TestOps(t *testing.T) {
	rep := func(ops ...splice.Op) []splice.Op { return ops }
	testCases := []struct {
		in   string
		want string
		ops  []splice.Op
	}{
		{"abcd", "abXcd", rep(splice.Span(2, 2).With("X"))},
		{"abcd", "abd", rep(splice.Span(2, 3).With(""))},
		{"abcd", "abYd", rep(splice.Span(2, 3).With("Y"))},
		{"abcd", "ab x d", rep(splice.Span(2, 3).With(" x "))},
		{"ab x d", "abcd", rep(splice.Span(2, 5).With("c"))},
		{"abcd", "abcd$", rep(splice.Span(4, 4).With("$"))},
		{"abcd", "^abcd", rep(splice.Span(0, 0).With("^"))},
		{"abcd", "", rep(splice.Span(0, 4).With(""))},
		{"", "abcd", rep(splice.Span(0, 0).With("abcd"))},
		{
			"abcde xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx from xyz",
			"aBCdE xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx from xyz",
			rep(
				splice.Span(1, 2).With("B"),
				splice.Span(2, 3).With("C"),
				splice.Span(4, 5).With("E"),
			),
		},
	}
	for i, tc := range testCases {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			tr := splice.T(tc.ops...)
			got, _, err := transform.String(tr, tc.in)
			if err != nil {
				t.Fatal(err)
			}
			if want := tc.want; got != want {
				t.Errorf("got: %q, want: %q", got, want)
			}
		})
	}
}

func TestPeek(t *testing.T) {
	s := func(s ...splice.Selection) []splice.Selection { return s }
	testCases := []struct {
		in   string
		sel  []splice.Selection
		want []string
	}{
		{"abcd", s(splice.Span(1, 2)), []string{"b"}},
		{"abcd", s(splice.Span(1, 2), splice.Span(2, 3)), []string{"b", "c"}},
		{"abcd", s(splice.Span(1, 3)), []string{"bc"}},
		{"abcd", s(splice.Span(0, 4)), []string{"abcd"}},
		{"abcd", s(splice.Span(3, 4)), []string{"d"}},
		{"abcd", s(splice.Span(4, 4)), []string{""}},
		{"abcd", s(splice.Span(1, 3), splice.Span(3, 4)), []string{"bc", "d"}},
		{"abcd", s(splice.Span(3, 4), splice.Span(1, 3)), []string{"d", "bc"}},
	}
	for i, tc := range testCases {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			got, err := splice.Peek(strings.NewReader(tc.in), tc.sel...)
			if err != nil {
				t.Fatal(err)
			}
			if want := tc.want; !reflect.DeepEqual(got, want) {
				t.Errorf("got: %q, want: %q", got, want)
			}
		})
	}
}
