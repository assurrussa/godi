# Lifecycle And Runnables

## Lifecycle

`Lifecycle` is a small helper that stores start/stop hooks.

- `Start` activates hooks in order; `Stop` cleans active hooks in reverse order.
- Only successfully started hooks receive `OnStop`. A hook whose `OnStart`
  fails owns cleanup of resources it acquired before returning the error.
- Stop-only hooks represent resources acquired externally. They run even if
  `Stop` is called before `Start`, or startup fails before reaching their position.
- On startup failure, rollback uses startup context values with an independent
  30-second deadline. Hooks must respect context cancellation; the timeout does
  not forcibly interrupt callbacks. Startup and rollback errors are joined.
- This is one lifecycle cycle: repeated successful `Start` is a no-op; repeated
  failed `Start` returns its saved error; restarting after shutdown is rejected.
- Each cleanup is attempted once, including on errors. Repeated `Stop` returns
  the saved cleanup error without running hooks again.
- Register hooks before the first `Start` or `Stop`. Late `Append` panics.
- State is synchronized, but overlapping or reentrant `Start`/`Stop` calls during
  a transition return an error. Callbacks run without holding the state mutex.

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

