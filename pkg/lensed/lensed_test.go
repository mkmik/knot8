package lensed

import (
	"fmt"
	"testing"
)

func TestLenses(t *testing.T) {
	src := []byte(`sc: alar
foo: |
  bar: a
  baz: b
  quz: '{"x": "y"}'
`)

	testCases := []struct {
		t    []Mapping
		want string
	}{
		{
			[]Mapping{
				{"/sc", "otty"},
			},
			`sc: otty
foo: |
  bar: a
  baz: b
  quz: '{"x": "y"}'
`,
		},
		{
			[]Mapping{
				{"/sc", "otty"},
				{"/foo/~(yaml)/bar", "A"},
				{"/foo/~(yaml)/baz", "B"},
			},
			`sc: otty
foo: |
  bar: A
  baz: B
  quz: '{"x": "y"}'
`,
		},
		{
			[]Mapping{
				{"/foo/~(yaml)/quz/~(yaml)/x", "Y"},
			},
			`sc: alar
foo: |
  bar: a
  baz: b
  quz: '{"x": "Y"}'
`,
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			got, err := Default.Apply(src, tc.t)
			if err != nil {
				t.Fatal(err)
			}

			if got, want := string(got), tc.want; got != want {
				t.Errorf("got: %q, want: %q", got, want)
			}
		})
	}
}

func TestSplit(t *testing.T) {
	testCases := []struct {
		src  string
		lens string
		ptr  string
		rest string
	}{
		{"~(l0)/a/b/~(l1)/c/d", "l0", "/a/b", "~(l1)/c/d"},
		{"~(l0)/a/b/~(l1)/c/d/~(l2)/", "l0", "/a/b", "~(l1)/c/d/~(l2)/"},
		{"~(l0)/", "l0", "/", ""},
		{"~(l0)/~(l1)/", "l0", "/", "~(l1)/"},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			lens, ptr, rest := split(tc.src)

			if got, want := lens, tc.lens; got != want {
				t.Errorf("got: %q, want: %q", got, want)
			}
			if got, want := ptr, tc.ptr; got != want {
				t.Errorf("got: %q, want: %q", got, want)
			}
			if got, want := rest, tc.rest; got != want {
				t.Errorf("got: %q, want: %q", got, want)
			}
		})
	}

}

func TestNormalize(t *testing.T) {
	testCases := []struct {
		ptr  string
		want string
	}{
		{"/foo/bar", "~()/foo/bar"},
		{"/", "~()/"},
		{"~()/foo/bar", "~()/foo/bar"},
		{"~(yaml)/foo/bar", "~(yaml)/foo/bar"},
		{"~(yaml)/foo/bar/", "~(yaml)/foo/bar"},
		{"~(yaml)/foo/bar/~(baz)", "~(yaml)/foo/bar/~(baz)/"},
		{"~(yaml)/foo/bar/~(baz)/", "~(yaml)/foo/bar/~(baz)/"},
		{"/~(yaml)", "~()/~(yaml)/"},
		{"/(notalens)", "~()/(notalens)"},
		{"", "~()/"},
	}
	for i, tc := range testCases {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			if got, want := normalize(tc.ptr), tc.want; got != want {
				t.Errorf("got: %q, want: %q", got, want)
			}
		})
	}
}
