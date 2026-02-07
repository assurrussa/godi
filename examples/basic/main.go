package main

import (
	"os"

	"go.uber.org/dig"

	"github.com/assurrussa/godi"
)

func main() {
	cnt, err := godi.NewContainer(
		godi.WithDependencies(
			godi.CollectDependencies(
				godi.NewDependency(func() string { return "world" }, godi.WithName("target")),
				godi.NewDependency(func(in struct {
					dig.In
					Target string `name:"target"`
				},
				) string {
					return "hello " + in.Target
				}),
			),
		),
	)
	if err != nil {
		panic(err)
	}

	if err := cnt.Validate(); err != nil {
		panic(err)
	}

	var greeting string
	if err := cnt.Invoke(func(s string) { greeting = s }); err != nil {
		panic(err)
	}

	if _, err := os.Stdout.WriteString(greeting + "\n"); err != nil {
		panic(err)
	}
}
