package main

import (
	"bytes"
	"io"
	"os"
	"slices"
	"strings"

	"go.uber.org/dig"

	"github.com/assurrussa/godi"
)

type out struct {
	dig.Out
	Named string   `name:"named"`
	Items []string `group:"items,flatten"`
}

type inParams struct {
	dig.In
	Reader io.Reader
	Named  string   `name:"named"`
	Items  []string `group:"items"`
}

func main() {
	cnt, err := godi.NewContainer(
		godi.WithMatchings(new(io.Reader)),
		godi.WithConstructors(
			func() *bytes.Buffer { return bytes.NewBufferString("ok") },
			func() out {
				return out{Named: "hello", Items: []string{"a", "b"}}
			},
		),
	)
	if err != nil {
		panic(err)
	}

	resolved, err := cnt.Resolve[inParams]()
	if err != nil {
		panic(err)
	}

	items := slices.Clone(resolved.Items)
	slices.Sort(items)
	outStr := resolved.Named + ":" + strings.Join(items, ",") + "\n"
	if _, err := os.Stdout.WriteString(outStr); err != nil {
		panic(err)
	}
}
