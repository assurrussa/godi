# Project Passport

Current as of: 2026-06-04.

`godi` is a Go library module: `github.com/assurrussa/godi`. It provides a thin
layer over `go.uber.org/dig` for DI dependency registration, module scopes,
explicit override/decorate scenarios, lifecycle hooks, optional dependencies,
and dependency graph diagnostics.

## Project Boundaries

- This is a library, not an HTTP service or long-running application runtime.
- The public contract lives in exported types and functions from the root
  `godi` package.
- Examples under `examples/` should compile like external package consumers.
- User-facing documentation is maintained in Russian under `docs/` and in
  English under `docs/en/`.

## Key Files

- `container.go`: container, `Provide`, `Invoke`, `Validate`, module build path.
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

## Verification

Fast check:

```bash
go test ./...
```

Full local gate from `Makefile`:

```bash
make check
```

`make check` runs `tidy`, `generate`, `fmt`, `vet`, `lint`, `test`,
`test-race`, and `cover-html`. The `fmt`, `lint`, and `cover-html` targets may
rewrite files; `cover.html` is ignored by git.

If the environment cannot access the system Go build cache, use a local cache:

```bash
mkdir -p tmp/gocache
GOCACHE=$PWD/tmp/gocache go test ./...
```

## Current Doc Debt

`CONTRIBUTING.md` currently appears copied from `goinertia`: it contains the
wrong project name and commands that are absent from this repository's
`Makefile`. Until it is refreshed, the sources of truth for development are
`AGENTS.md`, `README.md`, `docs/`, `go.mod`, `Makefile`, `.golangci.yml`, the
CI workflow, code, and tests.
