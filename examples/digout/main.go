package main

import (
	"bytes"
	"io"
	"os"
	"sort"
	"strings"

	"go.uber.org/dig"

	"github.com/assurrussa/godi"
)

type out struct {
	dig.Out
	Named string   `name:"named"`
	Items []string `group:"items,flatten"`
}

func main() {
	cnt, err := godi.NewContainer(
		godi.WithMatchings(new(io.Reader)),
		godi.WithDependencies(
			godi.CollectDependencies(
				godi.NewDependency(func() *bytes.Buffer { return bytes.NewBufferString("ok") }),
				godi.NewDependency(func() out {
					return out{Named: "hello", Items: []string{"a", "b"}}
				}),
			),
		),
	)
	if err != nil {
		panic(err)
	}

	var named string
	var items []string
	if err := cnt.Invoke(func(in struct {
		dig.In
		Reader io.Reader
		Named  string   `name:"named"`
		Items  []string `group:"items"`
	},
	) {
		named = in.Named
		items = in.Items
	}); err != nil {
		panic(err)
	}

	sort.Strings(items)
	outStr := named + ":" + strings.Join(items, ",") + "\n"
	if _, err := os.Stdout.WriteString(outStr); err != nil {
		panic(err)
	}
}
