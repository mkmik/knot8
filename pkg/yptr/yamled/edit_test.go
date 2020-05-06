package yamled_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
	"knot8.io/pkg/yptr"
	"knot8.io/pkg/yptr/yamled"
)

func TestSplice(t *testing.T) {
	src := `foo: abc
bar: xy
baz: end
`

	testCases := []struct {
		foo  string
		bar  string
		want string
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
			buf := bytes.NewBuffer([]byte(src))
			var n yaml.Node
			if err := yaml.Unmarshal(buf.Bytes(), &n); err != nil {
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

			edits := []yamled.Replacement{
				yamled.NewReplacement(tc.foo, foo),
				yamled.NewReplacement(tc.bar, bar),
			}

			var tmp bytes.Buffer
			if err := yamled.Replace(&tmp, strings.NewReader(src), edits); err != nil {
				t.Fatal(err)
			}
			t.Logf("after:\n%s", tmp.String())

			*buf = tmp

			var ne yaml.Node
			if err := yaml.Unmarshal(buf.Bytes(), &ne); err != nil {
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
