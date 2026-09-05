# Проектный паспорт

Актуально на: 2026-09-05.

`godi` — библиотечный Go-модуль `github.com/assurrussa/godi`. Проект даёт
тонкий слой над `go.uber.org/dig` для регистрации DI-зависимостей, модульных
scope, явных override/decorate-сценариев, lifecycle hooks, optional-зависимостей
и диагностического графа зависимостей.

## Границы проекта

- Это библиотека, а не HTTP-сервис и не приложение с постоянным runtime.
- Публичный контракт живёт в экспортируемых типах и функциях root package
  `godi`.
- Примеры под `examples/` должны компилироваться как внешние пользователи
  пакета.
- Пользовательская документация ведётся на русском в `docs/` и на английском в
  `docs/en/`.

## Основные файлы

- `container.go`: контейнер, `Provide`, `Invoke`, `Validate`, module build path.
- `dependency.go`, `dependency_options.go`: dependency model and options.
- `module.go`: module scope contract.
- `lifecycle.go`: ordered start/stop hooks.
- `graph.go`: graph model and DOT export.
- `matching.go`: automatic `dig.As(...)` matching model.
- `optional.go`: typed optional dependency wrapper.
- `overrides.go`: explicit replacement diagnostics.
- `Makefile`: local development gates.
- `.golangci.yml`: lint policy.
- `.github/workflows/go.yml`: CI lint/build/race-test workflow.

## Проверка

Быстрая проверка:

```bash
go test ./...
```

Полная локальная проверка из `Makefile`:

```bash
make check
```

`make check` проверяет tidy и форматирование, запускает vet, lint и race-тесты
без изменения файлов репозитория. `make fix` явно применяет tidy, генерацию,
форматирование и исправления линтера; `make cover-html` создаёт coverage-отчёт.
CI проверяет Go из `go.mod` и stable в одном job с ограничением времени; версия
`golangci-lint` закреплена на `v2.13.1`. Команды разработки описаны в `CONTRIBUTING.md`.

Если окружение не даёт доступ к системному Go build cache, используйте локальный
cache:

```bash
mkdir -p tmp/gocache
GOCACHE=$PWD/tmp/gocache go test ./...
```

## Контракт конкурентного использования

Создание, изменение, разрешение зависимостей, валидацию и чтение графа контейнера
вызывающая сторона должна выполнять последовательно. `Container` не безопасен
для конкурентного использования. Получайте экземпляры сервисов при запуске,
затем используйте их напрямую с учётом их собственных правил конкурентности.
`Provide` запрещён после первого `Invoke` или `Runnables`, даже при ошибке
разрешения. `Validate` работает в dry-run режиме и не пересекает эту границу.
Синхронизация состояния Lifecycle описана отдельно в `lifecycle.md`.
