package splice_test

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"knot8.io/pkg/splice"
)

func ExampleOp() {
	fmt.Printf("%T", splice.Span(3, 4).With("foo"))
	// Output:
	// splice.Op
}

func Example() {
	src := "abcd"

	res, err := splice.String(src, splice.Span(1, 2).With("B"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(res)

	// Output:
	// aBcd
}

func Example_multiple() {
	src := "abcd"

	aBcD, err := splice.String(src,
		splice.Span(1, 2).With("B"),
		splice.Span(3, 4).With("D"),
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(aBcD)

	aBaDa, err := splice.String(src,
		splice.Span(1, 2).With("Ba"),
		splice.Span(2, 3).With(""),
		splice.Span(3, 4).With("Da"),
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(aBaDa)

	// Output:
	// aBcD
	// aBaDa
}

func Example_lineCol() {
	src := "abcd\nefgh"

	res, err := splice.String(src, splice.Sel(splice.Loc(2, 2), splice.Loc(2, 3)).With("X"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(res)
	// Output:
	// abcd
	// eXgh
}

func Example_insert() {
	src := "abcd"

	res, err := splice.String(src, splice.Span(2, 2).With("X"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(res)

	// Output:
	// abXcd
}

func Example_delete() {
	src := "abcd"

	res, err := splice.String(src, splice.Span(2, 3).With(""))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(res)

	// Output:
	// abd
}

func ExampleBytes() {
	buf := []byte("abcd")
	aBcd, err := splice.Bytes(buf, splice.Span(1, 2).With("B"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(aBcd))

	// Output:
	// aBcd
}

func ExampleSwapBytes() {
	buf := []byte("abcd")
	if err := splice.SwapBytes(&buf, splice.Span(1, 2).With("B")); err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(buf))

	// Output:
	// aBcd
}

func ExampleFile() {
	tmp, err := ioutil.TempFile("", "")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tmp.Name())

	fmt.Fprintf(tmp, "abcd")
	tmp.Close()

	if err := splice.File(tmp.Name(), splice.Span(1, 3).With("X")); err != nil {
		log.Fatal(err)
	}

	b, err := ioutil.ReadFile(tmp.Name())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(b))

	// Output:
	// aXd
}
