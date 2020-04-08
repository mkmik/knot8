// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package kptr_test

import (
	"errors"
	"fmt"
	"testing"

	kptr "github.com/mkmik/knot8/pkg/kptr"
	yaml "gopkg.in/yaml.v3"
)

func ExampleFind() {
	src := `
a:
  b:
    c: 42
d:
- e
- f
`
	var n yaml.Node
	yaml.Unmarshal([]byte(src), &n)

	r, _ := kptr.Find(&n, `/a/b/c`)
	fmt.Printf("Scalar %q at %d:%d\n", r.Value, r.Line, r.Column)

	r, _ = kptr.Find(&n, `/d/0`)
	fmt.Printf("Scalar %q at %d:%d\n", r.Value, r.Line, r.Column)
	// Output: Scalar "42" at 4:8
	// Scalar "e" at 6:3
}

func ExampleFind_extension() {
	src := `kind: Deployment
apiVersion: apps/v1
metadata:
  name: foo
spec:
  template:
    spec:
      replicas: 1
      containers:
      - name: app
        image: nginx
      - name: sidecar
        image: mysidecar
`
	var n yaml.Node
	yaml.Unmarshal([]byte(src), &n)

	r, _ := kptr.Find(&n, `/spec/template/spec/containers/1/image`)
	fmt.Printf("Scalar %q at %d:%d\n", r.Value, r.Line, r.Column)

	r, _ = kptr.Find(&n, `/spec/template/spec/containers/~{"name":"app"}/image`)
	fmt.Printf("Scalar %q at %d:%d\n", r.Value, r.Line, r.Column)

	// Output: Scalar "mysidecar" at 13:16
	// Scalar "nginx" at 11:16
}

func ExampleFind_jsonPointerCompat() {
	// the array item match syntax doesn't accidentally match a field that just happens
	// to contain the same characters.
	src := `a:
  "{\"b\":\"c\"}": d
`
	var n yaml.Node
	yaml.Unmarshal([]byte(src), &n)

	r, _ := kptr.Find(&n, `/a/{"b":"c"}`)

	fmt.Printf("Scalar %q at %d:%d\n", r.Value, r.Line, r.Column)

	// Output: Scalar "d" at 2:20
}

func TestParse(t *testing.T) {
	src := `
spec:
  template:
    spec:
      replicas: 1
      containers:
      - name: app
        image: nginx
`
	var root yaml.Node
	if err := yaml.Unmarshal([]byte(src), &root); err != nil {
		t.Fatal(err)
	}
	if _, err := kptr.Find(&root, "/bad/path"); !errors.Is(err, kptr.ErrNotFound) {
		t.Fatalf("expecting not found error, got: %v", err)
	}

	testCases := []struct {
		ptr    string
		value  string
		line   int
		column int
	}{
		{`/spec/template/spec/replicas`, "1", 5, 17},
		{`/spec/template/spec/containers/0/image`, "nginx", 8, 16},
		{`/spec/template/spec/containers/~{"name":"app"}/image`, "nginx", 8, 16},
		{`/spec/template/spec/containers/~[name=app]/image`, "nginx", 8, 16},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			r, err := kptr.Find(&root, tc.ptr)
			if err != nil {
				t.Fatal(err)
			}
			if got, want := r.Value, tc.value; got != want {
				t.Fatalf("got: %v, want: %v", got, want)
			}
			if got, want := r.Line, tc.line; got != want {
				t.Errorf("got: %v, want: %v", got, want)
			}
			if got, want := r.Column, tc.column; got != want {
				t.Errorf("got: %v, want: %v", got, want)
			}
		})
	}

	errorCases := []struct {
		ptr string
		err error
	}{
		{"a", fmt.Errorf(`JSON pointer must be empty or start with a "/`)},
		{"/a", kptr.ErrNotFound},
	}
	for i, tc := range errorCases {
		t.Run(fmt.Sprint("error", i), func(t *testing.T) {
			_, err := kptr.Find(&root, tc.ptr)
			if err == nil {
				t.Fatal("error expected")
			}
			if got, want := err, tc.err; got.Error() != want.Error() && !errors.Is(got, want) {
				t.Errorf("got: %v, want: %v", got, want)
			}
		})
	}
}
