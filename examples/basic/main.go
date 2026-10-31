package main

import (
	"bytes"
	"io"
	"os"

	"github.com/assurrussa/godi"
)

func main() {
	cnt, err := godi.NewContainer(
		godi.WithConstructors(
			godi.NewDependency(func() *bytes.Buffer {
				return bytes.NewBufferString("world")
			}).As[io.Reader](),
			func(reader io.Reader) (string, error) {
				target, readErr := io.ReadAll(reader)
				if readErr != nil {
					return "", readErr
				}
				return "hello " + string(target), nil
			},
		),
	)
	if err != nil {
		panic(err)
	}

	if err := cnt.Validate(); err != nil {
		panic(err)
	}

	greeting, err := cnt.Resolve[string]()
	if err != nil {
		panic(err)
	}

	if _, err := os.Stdout.WriteString(greeting + "\n"); err != nil {
		panic(err)
	}
}
