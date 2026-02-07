# Modules

`godi` supports module scopes (implemented using `dig.Scope`).

Use cases:

- split dependencies by domain/package
- hide internal providers while still exporting higher-level services

## Defining A Module

```go
module := godi.Module{
  Name: "users",
  Dependencies: godi.CollectDependencies(
    // Private provider is visible only inside the module scope.
    godi.NewDependency(func() string { return "secret" }, godi.Private()),

    // Public provider can be exported and resolved from root.
    godi.NewDependency(func(secret string) int { return len(secret) }),
  ),
}

cnt, err := godi.NewContainer(godi.WithModules(module))
```

## Private Providers

- Root scope cannot resolve private providers.
- Module public providers can depend on private ones.

## Resolution Model

At build time, `godi` resolves slot winners across:

- root dependencies
- public module dependencies (private ones are excluded from global resolution)

Then, for each module, winning public providers are registered inside that module scope and exported to root using `dig.Export(true)`.

This means:

- if multiple modules export the same slot, container creation fails (duplicate provider) unless you explicitly model override via `Replace`
- private providers never leak to root

