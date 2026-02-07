# Начало работы

## Термины

- `Dependency`: конструктор (функция) + опциональные опции (name/group/as/key).
- `Dependencies`: список `Dependency`.
- `Container`: тонкая обертка над `dig.Container`, которая добавляет:
  - резолвинг "слотов" (`Provide` vs `Replace`)
  - модули (scopes) и приватность
  - валидацию без запуска конструкторов
  - экспорт графа зависимостей

## Создание контейнера

```go
cnt, err := godi.NewContainer(
  godi.WithDependencies(
    godi.CollectDependencies(
      godi.NewDependency(func() string { return "value" }),
    ),
  ),
)
```

### Правила сигнатуры конструктора

Конструктор обязан быть функцией и возвращать:

- `func(...) T`
- `func(...) (T, error)`

Вариант "только `error`" запрещен.

## Invoke

```go
var got string
err := cnt.Invoke(func(v string) {
  got = v
})
```

Первый `Invoke` запускает контейнер. После этого `Provide` запрещен.

## Добавление зависимостей после создания

```go
err := cnt.Provide(godi.CollectDependencies(
  godi.NewDependency(func() int { return 1 }),
))
```

`Provide` работает транзакционно: если новый набор зависимостей делает контейнер невалидным, состояние контейнера не "портится".

## Validate без запуска конструкторов

`Validate` выполняет dry-run сборку и проверяет, что все exposed слоты резолвятся.

```go
if err := cnt.Validate(); err != nil {
  // missing deps / invalid slots / invalid decorators, etc.
}
```
