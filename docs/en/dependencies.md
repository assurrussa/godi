# Dependencies

## Provide

By default, a dependency provides a slot.

```go
dep := godi.NewDependency(func() string { return "base" })
deps := godi.CollectDependencies(dep)
```

If multiple dependencies provide the same slot, container creation fails with a duplicate provider error.

## Replace

`Replace` overrides a slot.

```go
deps := godi.CollectDependencies(
  godi.NewDependency(func() string { return "base" }),
  godi.Replace(func() string { return "override" }),
)
```

Notes:

- Replace is not supported for `group` dependencies.
- Replace participates in slot resolution (it is not "last wins" by list order).

## Decorate

`Decorate` wraps an already-provided slot.

```go
deps := godi.CollectDependencies(
  godi.NewDependency(func() string { return "base" }),
  godi.Decorate(func(s string) string { return s + "!" }),
)
```

Decorator constructors must return:

- `func(...) T`
- `func(...) (T, error)`

### Named Decoration

`Decorate` does not support `WithName`, `WithGroup`, or `WithMatch`.
For named decoration, use `dig.In` for input and `dig.Out` for output:

```go
type out struct {
  dig.Out
  S string `name:"n"`
}

deps := godi.CollectDependencies(
  godi.NewDependency(func() string { return "base" }, godi.WithName("n")),
  godi.Decorate(func(in struct {
    dig.In
    S string `name:"n"`
  }) out {
    return out{S: in.S + "!"}
  }),
)
```

Group outputs from decorator `dig.Out` are currently not supported.

## Options

### WithName

Provides a named slot (maps to `dig.Name`).

```go
godi.NewDependency(func() string { return "x" }, godi.WithName("n"))
```

### WithGroup

Provides into a dig group (maps to `dig.Group`).

```go
godi.NewDependency(func() string { return "a" }, godi.WithGroup("items"))
godi.NewDependency(func() string { return "b" }, godi.WithGroup("items"))
```

### WithMatch

Exposes the dependency under an interface using `dig.As`.

```go
godi.NewDependency(func() *bytes.Buffer { return bytes.NewBufferString("ok") }, godi.WithMatch(new(io.Reader)))
```

### WithKey

Attaches a metadata key used by graph IDs and diagnostics. It does not affect resolution.

### Private

Marks a dependency as private to its module scope (see `docs/en/modules.md`).

## Providing Multiple Outputs With dig.Out

Constructors may return a `dig.Out` struct to provide multiple slots.

```go
type out struct {
  dig.Out
  Named string   `name:"named"`
  Item  []string `group:"items,flatten"`
}

godi.NewDependency(func() out {
  return out{Named: "ok", Item: []string{"a", "b"}}
})
```

Notes:

- `name` / `group` come from the field tags.
- To provide `[]T` into a group as elements, use the `flatten` modifier (example: `group:"items,flatten"`).
- `dig.Out` cannot be combined with `WithName`, `WithGroup`, or `WithMatch` on the same dependency.

## Registration Boundaries

Partial replacement of a multi-output provider is rejected during construction,
including public and private module providers. Replace all of its outputs (one
or several `Replace` constructors), or split the original constructor. Unchanged
outputs are never silently discarded. A provider that also contributes to a group
cannot be replaced this way: groups are additive. `Replace` cannot return group
fields, including fields inside nested `dig.Out` objects.

Nested result objects are expanded recursively for registration, validation,
override detection, and graph metadata. Return result objects by value.
A zero `Dependency` or a typed-nil constructor returns a configuration error;
metadata methods return no type instead of panicking.

Prefer one `CollectDependencies` batch over repeated `Provide` calls: each call
rebuilds the accumulated container. Compare local costs with:

```bash
go test -run '^$' -bench 'BenchmarkContainer' -benchmem -count=3
```

The fixtures cover 10, 100, and 500 independent named slots; registration,
validation, and DOT are measured separately. They do not measure service startup
or constructor execution costs.

## Optional Values

`Optional[T]` resolves an optional `*T`. `Get()` returns a copy of `T`; do not use
it for values containing a used mutex or when object identity matters. Use
`ptr, ok := optional.GetPtr()` to obtain the original pointer without copying.
