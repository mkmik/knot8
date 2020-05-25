package lensed

import (
	"fmt"
	"testing"

	yptr "github.com/vmware-labs/yaml-jsonpointer"
)

func TestYAMLS(t *testing.T) {
	src := `---
a: 1
---
a: 2
---
b: 3
`

	testCases := []struct {
		ptr  string
		want string
	}{
		{"~(yamls)/0/a", "1"},
		{"~(yamls)/1/a", "2"},
		{"~(yamls)/2/b", "3"},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			r, err := Get([]byte(src), []string{tc.ptr})
			if err != nil {
				t.Fatal(err)
			}
			if got, want := string(r[0]), tc.want; got != want {
				t.Errorf("got: %q, want: %q", got, want)
			}
		})
	}
}

func TestParseAllYAMLDocs(t *testing.T) {
	src := `---
a: 1
---
b: 2
---
c: 3
`

	d, err := parseAllYAMLDocs([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(d), 3; got != want {
		t.Errorf("got: %d, want: %d", got, want)
	}

	f0, err := yptr.Find(d[0], "/a")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := f0.Value, "1"; got != want {
		t.Errorf("got: %q, want: %q", got, want)
	}

	f1, err := yptr.Find(d[1], "/b")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := f1.Value, "2"; got != want {
		t.Errorf("got: %q, want: %q", got, want)
	}

	f2, err := yptr.Find(d[2], "/c")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := f2.Value, "3"; got != want {
		t.Errorf("got: %q, want: %q", got, want)
	}
}

func TestChompJSONPointer(t *testing.T) {
	testCases := []struct {
		ptr  string
		head string
		tail string
	}{
		{"/a/b/c", "a", "/b/c"},
		{"/a/b", "a", "/b"},
		{"/a", "a", ""},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			head, tail, err := chompJSONPointer(tc.ptr)
			if err != nil {
				t.Fatal(err)
			}
			if got, want := head, tc.head; got != want {
				t.Errorf("got: %q, want: %q", got, want)
			}
			if got, want := tail, tc.tail; got != want {
				t.Errorf("got: %q, want: %q", got, want)
			}
		})
	}

	if _, _, err := chompJSONPointer(""); err == nil {
		t.Errorf("wanted error, got %v", err)
	}

	if _, _, err := chompJSONPointer("a"); err == nil {
		t.Errorf("wanted error, got %v", err)
	}
}
