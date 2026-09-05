# Graph и диагностика

`godi` умеет строить граф зависимостей и рендерить его в DOT формат.

## Root graph

```go
g := cnt.Graph()
dot := cnt.GraphDOT()
```

Если резолвинг графа не удался (например, из-за дублей), `BuildGraph` делает best-effort fallback и показывает все entries.

## Module graphs

```go
graphs := cnt.GraphModules()      // root + each module scope
dots := cnt.GraphDOTModules()     // DOT for each graph
```

Module graphs включают:

- resolved root providers
- module-private providers (displayed as `replace` in module graph)

## Рендер DOT

Use Graphviz:

```bash
dot -Tsvg graph.dot > graph.svg
```

## Детект overrides

`DetectOverrides` reports explicit replacements (`godi.Replace`) by slot.

```go
overrides := godi.DetectOverrides(deps)
```

## Идентичность и замены

ID провайдеров включают модуль и локальный индекс: одинаковый `WithKey` в разных
модулях не объединяет узлы графа. Формат root ID сохраняется. ID предназначены
для диагностики, а не для постоянных ключей хранения.
`DetectOverrides` использует разрешение слотов контейнера и отражает `Replace`
независимо от порядка в списке. Для некорректного набора зависимостей отчёт пуст;
ошибку конфигурации можно получить через `NewContainer` или `Provide`.
