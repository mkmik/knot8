// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package yamled_test

import (
	"fmt"
	"log"
	"testing"

	yptr "github.com/vmware-labs/yaml-jsonpointer"
	"golang.org/x/text/transform"
	"gopkg.in/yaml.v3"
	"knot8.io/pkg/yamled"
)

// ExampleT shows how to use the transformer to edit a YAML source in place.
// It also shows how the quoting style is preserved
func ExampleT() {
	src := `apiVersion: v1
kind: Service
metadata:
  name: "foo"
  namespace: myns
`

	var root yaml.Node
	if err := yaml.Unmarshal([]byte(src), &root); err != nil {
		log.Fatal(err)
	}
	nameNode, err := yptr.Find(&root, "/metadata/name")
	if err != nil {
		log.Fatal(err)
	}
	nsNode, err := yptr.Find(&root, "/metadata/namespace")
	if err != nil {
		log.Fatal(err)
	}

	out, _, err := transform.String(yamled.T(
		yamled.Node(nameNode).With("bar"),
		yamled.Node(nsNode).With("otherns"),
	), src)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(out)

	// Output:
	// apiVersion: v1
	// kind: Service
	// metadata:
	//   name: "bar"
	//   namespace: otherns
}

func TestEdit(t *testing.T) {
	src := `foo: abc
bar: xy
baz: end
`

	testCases := []struct {
		foo string
		bar string
	}{
		{
			foo: "AB",
			bar: "xyz",
		},
		{
			foo: "ABCD",
			bar: "x",
		},
		{
			foo: "ABCD",
			bar: "",
		},
		{
			foo: "",
			bar: "x",
		},
		{
			foo: "",
			bar: "a#b",
		},
		{
			foo: "",
			bar: "a #b",
		},
		{
			foo: "",
			bar: " ",
		},
		{
			foo: "a",
			bar: "2",
		},
		{
			foo: "a\nb\n",
			bar: "ab",
		},
		{
			foo: "\na\nb\n",
			bar: "ab",
		},
		{
			foo: "\na\nb\n\n\n",
			bar: "ab",
		},
		{
			foo: "a",
			bar: "\n",
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			var n yaml.Node
			buf := []byte(src)
			if err := yaml.Unmarshal(buf, &n); err != nil {
				t.Fatal(err)
			}

			foo, err := yptr.Find(&n, "/foo")
			if err != nil {
				t.Fatal(err)
			}

			bar, err := yptr.Find(&n, "/bar")
			if err != nil {
				t.Fatal(err)
			}

			buf, _, err = transform.Bytes(yamled.T(
				yamled.Node(foo).With(tc.foo),
				yamled.Node(bar).With(tc.bar),
			), []byte(src))
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("after:\n%s", string(buf))

			var ne yaml.Node
			if err := yaml.Unmarshal(buf, &ne); err != nil {
				t.Fatal(err)
			}

			check := func(path, want string) {
				f, err := yptr.Find(&ne, path)
				if err != nil {
					t.Fatal(err)
				}
				if got := f.Value; got != want {
					t.Errorf("got: %q, want: %q", got, want)
				}

				if tag := f.Tag; tag != "!!str" && tag != "!!null" {
					t.Errorf("tag for %q must be either string or null, got %q", path, tag)
				}
			}
			check("/foo", tc.foo)
			check("/bar", tc.bar)
		})
	}
}

func TestIndent(t *testing.T) {
	// the long strings here excercise ErrShortSrc and ErrShortDst codepaths in the transformer.
	src := `out:
  foo: abc
  other:
    bar: xy
baz: end
xxx: aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
yyy: yyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyy
www:
  y: zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz
`

	testCases := []struct {
		foo string
		bar string
	}{
		{
			foo: "edit test",
			bar: "long\nzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz",
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			var n yaml.Node
			buf := []byte(src)
			if err := yaml.Unmarshal(buf, &n); err != nil {
				t.Fatal(err)
			}

			foo, err := yptr.Find(&n, "/out/foo")
			if err != nil {
				t.Fatal(err)
			}

			bar, err := yptr.Find(&n, "/out/other/bar")
			if err != nil {
				t.Fatal(err)
			}

			buf, _, err = transform.Bytes(yamled.T(
				yamled.Node(foo).With(tc.foo),
				yamled.Node(bar).With(tc.bar),
			), []byte(src))
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("after:\n%s", string(buf))

			var ne yaml.Node
			if err := yaml.Unmarshal(buf, &ne); err != nil {
				t.Fatal(err)
			}

			check := func(path, want string) {
				f, err := yptr.Find(&ne, path)
				if err != nil {
					t.Fatal(err)
				}
				if got := f.Value; got != want {
					t.Errorf("got: %q, want: %q", got, want)
				}

				if tag := f.Tag; tag != "!!str" && tag != "!!null" {
					t.Errorf("tag for %q must be either string or null, got %q", path, tag)
				}
			}
			check("/out/foo", tc.foo)
			check("/out/other/bar", tc.bar)
		})
	}

}
