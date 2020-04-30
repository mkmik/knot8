package yamled_test

import (
	"fmt"
	"testing"

	"gopkg.in/yaml.v3"
	"knot8.io/pkg/yamled"
	"knot8.io/pkg/yptr"
)

func TestSplice(t *testing.T) {
	src1 := `foo: abc
bar: xy
baz: end
`

	testCases := []struct {
		src  string
		foo  string
		bar  string
		want string

		fooStyle yaml.Style
		barStyle yaml.Style
	}{
		{
			src: src1,
			foo: "AB",
			bar: "xyz",
		},
		{
			src: src1,
			foo: "ABCD",
			bar: "x",
		},
		{
			src: src1,
			foo: "ABCD",
			bar: "",
		},
		{
			src: src1,
			foo: "",
			bar: "x",
		},
		{
			src: src1,
			foo: "",
			bar: "a#b",
		},
		{
			src:      src1,
			foo:      "",
			bar:      "a #b",
			barStyle: yaml.SingleQuotedStyle,
		},
		{
			src:      src1,
			foo:      "",
			bar:      " ",
			barStyle: yaml.SingleQuotedStyle,
		},
		{
			src:      src1,
			foo:      "a",
			bar:      "2",
			barStyle: yaml.DoubleQuotedStyle,
		},
		{
			src:      src1,
			foo:      "a\nb\n",
			fooStyle: yaml.LiteralStyle,
			bar:      "ab",
		},
		{
			src:      src1,
			foo:      "\na\nb\n",
			fooStyle: yaml.LiteralStyle,
			bar:      "ab",
		},
		{
			src:      src1,
			foo:      "\na\nb\n\n\n",
			fooStyle: yaml.LiteralStyle,
			bar:      "ab",
		},
		{
			src:      src1,
			foo:      "a",
			bar:      "\n",
			barStyle: yaml.LiteralStyle,
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			buf := yamled.RuneBuffer(tc.src)
			var n yaml.Node
			if err := yaml.Unmarshal([]byte(string(buf)), &n); err != nil {
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

			edits := []yamled.Edit{
				yamled.NewEdit(tc.foo, foo),
				yamled.NewEdit(tc.bar, bar),
			}
			if err := yamled.Splice(&buf, edits); err != nil {
				t.Fatal(err)
			}

			t.Logf("after:\n%s", string(buf))

			var ne yaml.Node
			if err := yaml.Unmarshal([]byte(string(buf)), &ne); err != nil {
				t.Fatal(err)
			}

			check := func(path, want string, style yaml.Style) {
				f, err := yptr.Find(&ne, path)
				if err != nil {
					t.Fatal(err)
				}
				if got := f.Value; got != want {
					t.Errorf("got: %q, want: %q", got, want)
				}
				if got, want := f.Style, style; got != want {
					t.Errorf("got: %d, want: %d", got, want)
				}

				if tag := f.Tag; tag != "!!str" && tag != "!!null" {
					t.Errorf("tag for %q must be either string or null, got %q", path, tag)
				}
			}
			check("/foo", tc.foo, tc.fooStyle)
			check("/bar", tc.bar, tc.barStyle)
		})
	}
}
