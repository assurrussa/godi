# Matchings (dig.As)

`dig.As` allows exposing a constructor result as one or more interfaces.

`godi` offers two ways to configure it:

- per-dependency via `As[T]` or `WithMatch`
- globally via `WithMatchings`

## As[T]

For an explicit binding, chain `As[T]()` on a dependency value:

```go
base := godi.NewDependency(func() *bytes.Buffer { return bytes.NewBufferString("ok") })
reader := base.As[io.Reader]()
readerWriter := reader.As[io.Writer]()
```

`T` must be an interface. `As[T]` applies `WithMatch(new(T))` and follows the
same validation rules. `readerWriter` exports `io.Reader` and `io.Writer`, but
not `*bytes.Buffer`. `base` stays concrete and `reader` still exports only
`io.Reader`; branching a chain does not change earlier values or sibling chains.
Register the variant you intend to use. Interfaces are not bound automatically
unless you opt into `WithMatchings`.

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

