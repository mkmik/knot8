package atomicfile

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"unicode"

	"golang.org/x/text/runes"
)

func TestWrite(t *testing.T) {
	const testMode = os.FileMode(0664)

	tmp, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp.Name())

	fmt.Fprintf(tmp, "abcd")
	tmp.Close()
	if err := os.Chmod(tmp.Name(), testMode); err != nil {
		t.Fatal(err)
	}

	if err := WriteFile(tmp.Name(), []byte("ABCD"), 0); err != nil {
		t.Fatal(err)
	}

	b, err := ioutil.ReadFile(tmp.Name())
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(b), "ABCD"; got != want {
		t.Errorf("got: %q, want: %q", got, want)
	}

	st, err := os.Stat(tmp.Name())
	if err != nil {
		t.Fatal(err)
	}

	if got, want := st.Mode(), testMode; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

func TestTransform(t *testing.T) {
	tmp, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp.Name())

	fmt.Fprintf(tmp, "abcd")
	tmp.Close()

	if err := Transform(runes.Map(unicode.ToUpper), tmp.Name()); err != nil {
		t.Fatal(err)
	}

	b, err := ioutil.ReadFile(tmp.Name())
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(b), "ABCD"; got != want {
		t.Errorf("got: %q, want: %q", got, want)
	}
}