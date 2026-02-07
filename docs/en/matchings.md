# Matchings (dig.As)

`dig.As` allows exposing a constructor result as one or more interfaces.

`godi` offers two ways to configure it:

- per-dependency via `WithMatch`
- globally via `WithMatchings`

## WithMatch

Use `WithMatch` when you want to explicitly bind a dependency to an interface.

```go
godi.NewDependency(
  func() *bytes.Buffer { return bytes.NewBufferString("ok") },
  godi.WithMatch(new(io.Reader)),
)
```

## WithMatchings

Use `WithMatchings` to apply interface bindings automatically for provided dependencies.

### Interface Pointer

If you pass a pointer to an interface, `godi` checks each dependency type and applies `dig.As` when it implements the interface.

```go
cnt, err := godi.NewContainer(
  godi.WithMatchings(new(io.Reader)),
  godi.WithDependencies(
    godi.NewSingleDependency(func() *bytes.Buffer { return bytes.NewBuffer(nil) }),
  ),
)
```

### NewMatching(origin, interfaces...)

If you pass a `Matching`, `godi` applies `dig.As` only for a specific origin type.

```go
cnt, err := godi.NewContainer(
  godi.WithMatchings(godi.NewMatching(new(bytes.Buffer), new(io.Reader))),
  godi.WithDependencies(
    godi.NewSingleDependency(func() *bytes.Buffer { return bytes.NewBuffer(nil) }),
  ),
)
```

Notes:

- `origin` must be a pointer type (for example `new(bytes.Buffer)`).
- `interfaces` must be pointers to interfaces (for example `new(io.Reader)`).
- `NewMatching(new(T), ...)` matches both constructors returning `T` and `*T`.

