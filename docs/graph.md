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
- module-private providers

Private-видимость моделируется для каждого выходного слота независимо от `Replace`.
Обычный private-слот скрывает соответствующий глобальный слот; групповые значения
из обоих scopes складываются, в том числе для смешанных выходов `dig.Out`.
Незатенённые выходы глобального multi-output провайдера остаются видимыми.
Полностью скрытые узлы не показываются. Узлы сохраняют объявленный вид операции:
сам по себе `Private()` не превращает провайдер в узел `replace`.

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
