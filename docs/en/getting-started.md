# Getting Started

Requires Go `1.27.0` or newer.

## Concepts

- `Dependency`: a constructor function plus optional options (name/group/as/key).
- `Dependencies`: a list of `Dependency` values.
- `Container`: a thin wrapper around `dig.Container` that adds:
  - slot resolution (`Provide` vs `Replace`)
  - modules (scopes) and privacy
  - validation without running constructors
  - dependency graph export

## Creating A Container

```go
cnt, err := godi.NewContainer(
  godi.WithConstructors(func() string { return "value" }),
)
```

`WithConstructors(items ...any)` accepts constructor functions and ready
`Dependency` values, including `Replace`, `Decorate`, and their options. Pass
values, not `*Dependency` pointers. Each function is wrapped with `NewDependency`.
Repeated `WithConstructors` and `WithDependencies` options append registrations.
Invalid items fail `NewContainer` through the usual dependency validation.
Neither creating the option, building the container, nor `Validate` runs constructors.

### Constructor Signature Rules

Constructors must be functions that return:

- `func(...) T`
- `func(...) (T, error)`

Returning only `error` is rejected.

## Resolving A Value

```go
got, err := cnt.Resolve[string]()
if err != nil {
  return err
}
```

`Resolve[T]()` is a thin wrapper around `Invoke`: it returns the resolved value
or the zero value of `T` and the original error. Constructors remain lazy and
cached; resolving the same pointer dependency preserves its identity. It does
not infer interface bindings: register them explicitly with `As[T]` or matchings.

Use `Invoke` with `dig.In` for named/group inputs and more complex injection.
`Resolve` can also resolve a `dig.In` struct; it adds no separate name/group options.

## Invoking

```go
var got string
err := cnt.Invoke(func(v string) {
  got = v
})
```

The first `Invoke`, `Resolve`, or `Runnables` call starts the container, even if
resolution fails. After that, `Provide` is rejected. Serialize all container calls;
`Container` is not safe for concurrent use.

## Adding Dependencies Incrementally

```go
err := cnt.Provide(godi.CollectDependencies(
  godi.NewDependency(func() int { return 1 }),
))
```

`Provide` is transactional: if the added dependencies make the container invalid, the container state is not polluted.

## Validate Without Running Constructors

`Validate` runs a dry build and checks that all exposed slots are resolvable.

```go
if err := cnt.Validate(); err != nil {
  // missing deps / invalid slots / invalid decorators, etc.
}
```

