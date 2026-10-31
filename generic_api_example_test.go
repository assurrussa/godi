package godi_test

import (
	"bytes"
	"io"
	"os"

	"github.com/assurrussa/godi"
)

func ExampleContainer_Resolve() {
	cnt, err := godi.NewContainer(godi.WithConstructors(
		godi.NewDependency(func() *bytes.Buffer {
			return bytes.NewBufferString("hello world")
		}).As[io.Reader](),
		io.ReadAll,
	))
	if err != nil {
		panic(err)
	}
	message, err := cnt.Resolve[[]byte]()
	if err != nil {
		panic(err)
	}
	if _, err := os.Stdout.Write(append(message, '\n')); err != nil {
		panic(err)
	}
	// Output: hello world
}
