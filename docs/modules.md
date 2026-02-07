# Модули (Modules)

`godi` поддерживает модульные scopes (реализовано через `dig.Scope`).

Сценарии:

- группировать зависимости по доменам/пакетам
- скрывать внутренние провайдеры, экспортируя наружу только верхнеуровневые сервисы

## Определение модуля

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

## Private провайдеры

- Root scope не может резолвить private провайдеры.
- Публичные провайдеры внутри модуля могут зависеть от private.

## Модель резолвинга

На этапе сборки `godi` выбирает "победителей" слотов среди:

- root dependencies
- public module dependencies (private исключены из global resolution)

Затем, для каждого модуля, публичные провайдеры-победители регистрируются внутри module scope и экспортируются в root через `dig.Export(true)`.

Следствия:

- если несколько модулей экспортируют один и тот же слот, создание контейнера упадет с duplicate provider (если явно не моделировать override через `Replace`)
- private провайдеры никогда не "протекают" в root
