package kptr_test

import (
	"errors"
	"fmt"
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
}
