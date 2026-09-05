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


## Identity And Overrides

Provider IDs include module scope as well as the local index, so reusing `WithKey`
in different modules does not merge graph nodes. Root IDs retain their existing
format. Treat IDs as diagnostic identifiers rather than persistent storage keys.
`DetectOverrides` uses the same slot resolution as the container and reports
`Replace` regardless of list order. Invalid dependency sets return no override
report; use `NewContainer` or `Provide` to obtain the configuration error.
