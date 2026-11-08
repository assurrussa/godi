package main

import (
	"os"
	"strconv"

	"github.com/assurrussa/godi"
)

func main() {
	module := godi.NewModule("users", godi.CollectDependencies(
		godi.NewDependency(func() string { return "secret" }, godi.Private()),
		godi.NewDependency(func(secret string) int { return len(secret) }),
	))

	cnt, err := godi.NewContainer(godi.WithModules(module))
	if err != nil {
		panic(err)
	}

	n, err := cnt.Resolve[int]()
	if err != nil {
		panic(err)
	}

	if _, err := os.Stdout.WriteString(strconv.Itoa(n) + "\n"); err != nil {
		panic(err)
	}

	if _, err := cnt.Resolve[string](); err == nil {
		panic("expected private provider to be hidden from root")
	}
}
