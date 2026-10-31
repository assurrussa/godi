# Project Passport

Current as of: 2026-10-31.

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

Requires Go `1.27.0` or newer; the underlying DI library is `go.uber.org/dig`
`v1.19.0`.

## Key Files

- `container.go`: container, `Provide`, `Invoke`, `Resolve`, `Validate`, module build path.
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

`make check` verifies module tidiness and formatting, runs vet, lint, and race
tests without rewriting repository files. `make fix` explicitly applies tidy,
generation, formatting, and lint fixes; `make cover-html` generates coverage.
CI tests the Go version from `go.mod` and stable in one bounded job, and pins
`golangci-lint` to `v2.13.2`. See `CONTRIBUTING.md` for development commands.

Use the configured shared Go build and module caches for local checks. If the
sandbox restricts cache access, obtain access to those caches rather than
creating per-checkout caches.

## Concurrency Contract

Container construction, mutation, resolution, validation, and graph inspection
must be serialized by the caller. `Container` is not safe for concurrent use.
Resolve service instances during startup and then use those instances directly;
their own concurrency contracts apply. `Provide` is forbidden after the first
`Invoke`, `Resolve`, or `Runnables` call, even when resolution fails. `Validate`
is a dry run and does not cross that boundary. Lifecycle state synchronization is documented
separately in `lifecycle.md`.
