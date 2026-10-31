# Matchings (dig.As)

`dig.As` позволяет экспортировать результат конструктора как один или несколько интерфейсов.

В `godi` есть два способа настроить это:

- на уровне конкретной зависимости через `As[T]` или `WithMatch`
- глобально для контейнера через `WithMatchings`

## As[T]

Для явной привязки вызовите `As[T]()` на значении зависимости:

```go
base := godi.NewDependency(func() *bytes.Buffer { return bytes.NewBufferString("ok") })
reader := base.As[io.Reader]()
readerWriter := reader.As[io.Writer]()
```

`T` должен быть интерфейсом. `As[T]` применяет `WithMatch(new(T))` и следует
тем же правилам валидации. `readerWriter` экспортирует `io.Reader` и `io.Writer`,
но не `*bytes.Buffer`. `base` сохраняет конкретный тип, а `reader` экспортирует
только `io.Reader`: новая цепочка не изменяет предыдущие значения и соседние
цепочки. Регистрируйте нужный вариант зависимости. Интерфейсы автоматически
не привязываются, если вы явно не включили `WithMatchings`.

## WithMatch

Используйте `WithMatch`, когда хотите явно привязать зависимость к интерфейсу.

```go
godi.NewDependency(
  func() *bytes.Buffer { return bytes.NewBufferString("ok") },
  godi.WithMatch(new(io.Reader)),
)
```

## WithMatchings

Используйте `WithMatchings`, чтобы автоматически применять bindings для предоставляемых зависимостей.

### Interface Pointer

Если передать pointer-to-interface, `godi` проверяет тип каждой зависимости и применяет `dig.As`, когда тип реализует интерфейс.

```go
cnt, err := godi.NewContainer(
  godi.WithMatchings(new(io.Reader)),
  godi.WithDependencies(
    godi.NewSingleDependency(func() *bytes.Buffer { return bytes.NewBuffer(nil) }),
  ),
)
```

### NewMatching(origin, interfaces...)

Если передать `Matching`, `godi` применяет `dig.As` только для конкретного origin-типа.

```go
cnt, err := godi.NewContainer(
  godi.WithMatchings(godi.NewMatching(new(bytes.Buffer), new(io.Reader))),
  godi.WithDependencies(
    godi.NewSingleDependency(func() *bytes.Buffer { return bytes.NewBuffer(nil) }),
  ),
)
```

Примечания:

- `origin` обязан быть pointer-типом (например `new(bytes.Buffer)`).
- `interfaces` обязаны быть pointers to interfaces (например `new(io.Reader)`).
- `NewMatching(new(T), ...)` матчится и на конструкторы, возвращающие `T`, и на конструкторы, возвращающие `*T`.
