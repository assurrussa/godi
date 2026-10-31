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
- Explicit interface bindings with `As[T]` / `WithMatch`, or automatic bindings via matchings
- `dig.Out` multi-output support (including `name` / `group` tags)
- `Runnable` collection + `Lifecycle` helper
- Dependency graph export (DOT/Graphviz) and override detection

## Install

Requires Go `1.27.0` or newer.

```bash
go get github.com/assurrussa/godi@latest
```

## Quick Start

```go
package main

import (
	"bytes"
	"fmt"
	"io"
	"log"

	"github.com/assurrussa/godi"
)

func main() {
	cnt, err := godi.NewContainer(
		godi.WithConstructors(
			godi.NewDependency(func() *bytes.Buffer {
				return bytes.NewBufferString("world")
			}).As[io.Reader](),
			func(reader io.Reader) (string, error) {
				name, err := io.ReadAll(reader)
				if err != nil {
					return "", err
				}
				return "hello " + string(name), nil
			},
		),
	)
	if err != nil {
		log.Fatal(err)
	}
	if err := cnt.Validate(); err != nil {
		log.Fatal(err)
	}

	greeting, err := cnt.Resolve[string]()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(greeting)
}
```

`WithConstructors` mixes constructor functions and ready `Dependency` values.
`As[io.Reader]()` explicitly exposes the buffer as `io.Reader`; it does not also
register `*bytes.Buffer`. Constructors run lazily when resolved, and their results
are cached. `Validate` does not run them.

`Resolve[T]()` uses the same resolution path as `Invoke`. Either call starts the
container, even on failure, so add all dependencies before resolving. Serialize
container calls. Use `Invoke` and `dig.In` for named/group injection; a `dig.In`
struct may also be the type argument to `Resolve`.

## Documentation

- English: `docs/en/README.md`
- Russian: `docs/README.md`

## Examples

- `go run ./examples/basic`
- `go run ./examples/modules`
- `go run ./examples/digout`

## License

MIT
