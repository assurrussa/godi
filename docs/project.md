# Проектный паспорт

Актуально на: 2026-06-04.

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

`make check` запускает `tidy`, `generate`, `fmt`, `vet`, `lint`, `test`,
`test-race` и `cover-html`. Команды `fmt`, `lint` и `cover-html` могут менять
файлы; `cover.html` игнорируется git.

Если окружение не даёт доступ к системному Go build cache, используйте локальный
cache:

```bash
mkdir -p tmp/gocache
GOCACHE=$PWD/tmp/gocache go test ./...
```

## Текущий doc debt

`CONTRIBUTING.md` сейчас выглядит скопированным из `goinertia`: в нём есть
чужое имя проекта и команды, отсутствующие в текущем `Makefile`. Пока файл не
обновлён, источниками истины для разработки являются `AGENTS.md`, `README.md`,
`docs/`, `go.mod`, `Makefile`, `.golangci.yml`, CI workflow, код и тесты.
