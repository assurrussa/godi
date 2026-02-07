# Getting Started

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
  godi.WithDependencies(
    godi.CollectDependencies(
      godi.NewDependency(func() string { return "value" }),
    ),
  ),
)
```

### Constructor Signature Rules

Constructors must be functions that return:

- `func(...) T`
- `func(...) (T, error)`

Returning only `error` is rejected.

## Invoking

```go
var got string
err := cnt.Invoke(func(v string) {
  got = v
})
```

The first `Invoke` starts the container. After that, `Provide` is rejected.

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

