package kptr_test

import (
	"errors"
	"testing"

	kptr "github.com/mkmik/kno8/pkg/kptr"
	yaml "gopkg.in/yaml.v3"
)

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
	if false {
		if _, err := kptr.Find(&root, "/bad/path"); !errors.Is(err, kptr.ErrNotFound) {
			t.Fatalf("expecting not found error, got: %v", err)
		}
	}

	r, err := kptr.Find(&root, `/spec/template/spec/replicas`)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := r.Line, 5; got != want {
		t.Fatalf("got: %v, want: %v", got, want)
	}
	if got, want := r.Column, 17; got != want {
		t.Fatalf("got: %v, want: %v", got, want)
	}

	r, err = kptr.Find(&root, `/spec/template/spec/containers/0/image`)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := r.Value, "nginx"; got != want {
		t.Fatalf("got: %v, want: %v", got, want)
	}

	r, err = kptr.Find(&root, `/spec/template/spec/containers/~{"name":"app"}/image`)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := r.Value, "nginx"; got != want {
		t.Fatalf("got: %v, want: %v", got, want)
	}

	r, err = kptr.Find(&root, `/spec/template/spec/containers/~[name=app]/image`)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := r.Value, "nginx"; got != want {
		t.Fatalf("got: %v, want: %v", got, want)
	}
}
