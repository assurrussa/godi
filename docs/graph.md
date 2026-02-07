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
