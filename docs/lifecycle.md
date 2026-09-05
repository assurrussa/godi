# Lifecycle и Runnables

## Lifecycle

`Lifecycle` это небольшой хелпер, который хранит start/stop hooks.

- `Start` активирует hooks по порядку, `Stop` останавливает активные hooks в обратном порядке.
- `OnStop` вызывается только после успешного `OnStart`. Hook с ошибкой `OnStart`
  самостоятельно очищает ресурсы, которые успел создать перед возвратом ошибки.
- Stop-only hooks описывают ресурсы, созданные снаружи. Они выполняются даже при
  `Stop` до `Start` или ошибке запуска до достижения их позиции в списке.
- При ошибке запуска rollback сохраняет значения startup-контекста, но получает
  независимый deadline 30 секунд. Hooks должны учитывать отмену контекста:
  таймаут не прерывает callback принудительно. Ошибки запуска и отката объединяются.
- Lifecycle рассчитан на один цикл: повторный успешный `Start` ничего не делает;
  повторный неуспешный возвращает сохранённую ошибку; перезапуск после остановки запрещён.
- Каждая очистка вызывается один раз, в том числе при ошибке. Повторный `Stop`
  возвращает сохранённую ошибку очистки без повторного выполнения hooks.
- Регистрируйте hooks до первого `Start` или `Stop`. Поздний `Append` вызывает panic.
- Состояние синхронизировано, но одновременный или вложенный `Start`/`Stop` во время
  перехода возвращает ошибку. Callbacks выполняются без удержания mutex состояния.

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
