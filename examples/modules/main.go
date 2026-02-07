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

	var n int
	if err := cnt.Invoke(func(v int) { n = v }); err != nil {
		panic(err)
	}

	if _, err := os.Stdout.WriteString(strconv.Itoa(n) + "\n"); err != nil {
		panic(err)
	}

	if err := cnt.Invoke(func(_ string) {}); err == nil {
		panic("expected private provider to be hidden from root")
	}
}
