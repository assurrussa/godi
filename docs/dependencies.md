# Зависимости (Dependencies)

## Provide

По умолчанию зависимость "предоставляет" (provide) слот.

```go
dep := godi.NewDependency(func() string { return "base" })
deps := godi.CollectDependencies(dep)
```

Если несколько зависимостей provide один и тот же слот, создание контейнера завершится ошибкой "duplicate provider".

## Replace

`Replace` переопределяет (override) слот.

```go
deps := godi.CollectDependencies(
  godi.NewDependency(func() string { return "base" }),
  godi.Replace(func() string { return "override" }),
)
```

Примечания:

- Replace не поддерживается для `group` зависимостей.
- Replace участвует в резолвинге слотов (это не "последний в списке победил").

## Decorate

`Decorate` оборачивает уже предоставленный слот.

```go
deps := godi.CollectDependencies(
  godi.NewDependency(func() string { return "base" }),
  godi.Decorate(func(s string) string { return s + "!" }),
)
```

Конструктор-декоратор обязан возвращать:

- `func(...) T`
- `func(...) (T, error)`

### Named Decoration

`Decorate` не поддерживает `WithName`, `WithGroup`, `WithMatch`.
Чтобы декорировать named слот, используйте `dig.In` на входе и `dig.Out` на выходе:

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

Group outputs из `dig.Out` у декоратора пока не поддерживаются.

## Options

### WithName

Предоставляет named слот (маппится на `dig.Name`).

```go
godi.NewDependency(func() string { return "x" }, godi.WithName("n"))
```

### WithGroup

Добавляет значение в dig group (маппится на `dig.Group`).

```go
godi.NewDependency(func() string { return "a" }, godi.WithGroup("items"))
godi.NewDependency(func() string { return "b" }, godi.WithGroup("items"))
```

### WithMatch

Экспортирует зависимость как интерфейс через `dig.As`.

```go
godi.NewDependency(func() *bytes.Buffer { return bytes.NewBufferString("ok") }, godi.WithMatch(new(io.Reader)))
```

### WithKey

Добавляет метаданные `key`, которые используются в graph IDs и диагностике. На резолвинг не влияет.

### Private

Делает зависимость приватной внутри модуля (см. `docs/modules.md`).

## Providing Multiple Outputs With dig.Out

Конструктор может возвращать `dig.Out` struct и тем самым provide несколько слотов.

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

Примечания:

- `name` / `group` берутся из тегов поля.
- Чтобы `[]T` попало в группу как элементы, используйте модификатор `flatten` (пример: `group:"items,flatten"`).
- `dig.Out` нельзя комбинировать с `WithName`, `WithGroup`, `WithMatch` в рамках одной зависимости.
