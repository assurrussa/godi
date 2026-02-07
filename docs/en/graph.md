# Graph And Diagnostics

`godi` can build a dependency graph and render it in DOT format.

## Root Graph

```go
g := cnt.Graph()
dot := cnt.GraphDOT()
```

If graph resolution fails (for example, due to duplicates), `BuildGraph` falls back to showing all entries best-effort.

## Module Graphs

```go
graphs := cnt.GraphModules()      // root + each module scope
dots := cnt.GraphDOTModules()     // DOT for each graph
```

Module graphs include:

- resolved root providers
- module-private providers (displayed as `replace` in module graph)

## Rendering DOT

Use Graphviz:

```bash
dot -Tsvg graph.dot > graph.svg
```

## Override Detection

`DetectOverrides` reports explicit replacements (`godi.Replace`) by slot.

```go
overrides := godi.DetectOverrides(deps)
```

