# Lifecycle и Runnables

## Lifecycle

`Lifecycle` это небольшой хелпер, который хранит start/stop hooks.

- `Start` запускает hooks по порядку
- `Stop` запускает hooks в обратном порядке
- если `Start` падает, уже запущенные hooks будут остановлены (best-effort)

```go
l := godi.NewLifecycle()
l.Append(godi.Hook{
  OnStart: func(ctx context.Context) error { return nil },
  OnStop:  func(ctx context.Context) error { return nil },
})
```

### Default Lifecycle In Container

Можно автоматически зарегистрировать `*Lifecycle` в контейнере:

```go
cnt, err := godi.NewContainer(godi.WithDefaultLifecycle())
```

## Runnable

`Runnable` is a simple struct with start/stop callbacks.

Если вы предоставляете зависимости, которые возвращают `godi.Runnable`, они автоматически попадают в группу.
Используйте `Container.Runnables()`, чтобы собрать их:

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

- `Runnables()` запускает контейнер (после этого `Provide` запрещен).
- `Runnable` нельзя комбинировать с `WithGroup`.
