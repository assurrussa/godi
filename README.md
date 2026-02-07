# godi

[![Go Reference](https://pkg.go.dev/badge/github.com/assurrussa/godi.svg)](https://pkg.go.dev/github.com/assurrussa/godi)
[![Go Report Card](https://goreportcard.com/badge/github.com/assurrussa/godi)](https://goreportcard.com/report/github.com/assurrussa/godi)
[![Go](https://github.com/assurrussa/godi/actions/workflows/go.yml/badge.svg)](https://github.com/assurrussa/godi/actions/workflows/go.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## Overview

godi is a lightweight DI helper built on top of `go.uber.org/dig`.

## Features

- `Provide` / `Replace` / `Decorate` over dependency "slots"
- Module scopes with `Private()` providers
- Automatic `dig.As(...)` bindings via matchings
- `dig.Out` multi-output support (including `name` / `group` tags)
- `Runnable` collection + `Lifecycle` helper
- Dependency graph export (DOT/Graphviz) and override detection

## Install

```bash
go get github.com/assurrussa/godi@latest
```

## Quick Start

```go
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
				}) string {
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
```

## Documentation

- English: `docs/en/README.md`
- Russian: `docs/README.md`

## Examples

- `go run ./examples/basic`
- `go run ./examples/modules`
- `go run ./examples/digout`

## License

MIT
