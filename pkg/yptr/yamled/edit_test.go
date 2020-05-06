package yamled_test

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
	"knot8.io/pkg/yptr"
	"knot8.io/pkg/yptr/yamled"
)

func TestReplace(t *testing.T) {
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

func TestExtract(t *testing.T) {
	testCases := []struct {
		in   string
		ex   []yamled.Extent
		want []string
	}{
		{"abcd", []yamled.Extent{{1, 2}}, []string{"b"}},
		{"abcd", []yamled.Extent{{1, 2}, {2, 3}}, []string{"b", "c"}},
		{"abcd", []yamled.Extent{{1, 3}}, []string{"bc"}},
		{"abcd", []yamled.Extent{{0, 4}}, []string{"abcd"}},
		{"abcd", []yamled.Extent{{3, 4}}, []string{"d"}},
		{"abcd", []yamled.Extent{{4, 4}}, []string{""}},
		{"abcd", []yamled.Extent{{1, 3}, {3, 4}}, []string{"bc", "d"}},
		{"abcd", []yamled.Extent{{3, 4}, {1, 3}}, []string{"d", "bc"}},
	}
	for i, tc := range testCases {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			got, err := yamled.Extract(strings.NewReader(tc.in), tc.ex)
			if err != nil {
				t.Fatal(err)
			}
			if want := tc.want; !reflect.DeepEqual(got, want) {
				t.Errorf("got: %q, want: %q", got, want)
			}
		})
	}
}

func TestUpdateInPlace(t *testing.T) {
	tmp, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp.Name())

	fmt.Fprintf(tmp, "foo: bar\n")

	if _, err := tmp.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	var doc yaml.Node
	if err := yaml.NewDecoder(tmp).Decode(&doc); err != nil {
		t.Fatal(err)
	}
	tmp.Close()

	n, err := yptr.Find(&doc, "/foo")
	if err != nil {
		t.Fatal(err)
	}
	if err := yamled.UpdateFile(tmp.Name(), yamled.NewReplacement("baz", n)); err != nil {
		t.Fatal(err)
	}

	b, err := ioutil.ReadFile(tmp.Name())
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(b), "foo: baz\n"; got != want {
		t.Errorf("got: %q, want: %q", got, want)
	}
}
