package yamled_test

import (
	"testing"

	"gopkg.in/yaml.v3"
	"knot8.io/pkg/yamled"
	"knot8.io/pkg/yptr"
)

func TestSplice(t *testing.T) {
	src := `foo: abc
bar: xy
baz: end
`

	buf := yamled.RuneBuffer(src)
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
		yamled.NewEdit("AB", foo),
		yamled.NewEdit("xyz", bar),
	}
	if err := yamled.Splice(&buf, edits); err != nil {
		t.Fatal(err)
	}

	want := `foo: AB
bar: xyz
baz: end
`

	if got := string(buf); got != want {
		t.Errorf("got: %q, want: %q", got, want)
	}
}
