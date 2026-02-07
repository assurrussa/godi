# Lifecycle And Runnables

## Lifecycle

`Lifecycle` is a small helper that stores start/stop hooks.

- `Start` runs hooks in order
- `Stop` runs hooks in reverse order
- if `Start` fails, already-started hooks are stopped (best-effort)

```go
l := godi.NewLifecycle()
l.Append(godi.Hook{
  OnStart: func(ctx context.Context) error { return nil },
  OnStop:  func(ctx context.Context) error { return nil },
})
```

### Default Lifecycle In Container

You can register a `*Lifecycle` automatically:

```go
cnt, err := godi.NewContainer(godi.WithDefaultLifecycle())
```

## Runnable

`Runnable` is a simple struct with start/stop callbacks.

If you provide dependencies that return `godi.Runnable`, they are automatically grouped.
Use `Container.Runnables()` to collect them:

```go
cnt, err := godi.NewContainer(godi.WithDependencies(
  godi.NewSingleDependency(func() godi.Runnable {
    return godi.Runnable{
      OnStart: func(context.Context) error { return nil },
      OnStop:  func(context.Context) error { return nil },
    }
  }),
))

r, err := cnt.Runnables()
```

Notes:

- `Runnables()` starts the container (after that `Provide` is rejected).
- `Runnable` cannot be combined with `WithGroup`.

