# Repository Guidelines

## Project Role

`godi` is a Go library module: `github.com/assurrussa/godi`.
It is a lightweight dependency-injection helper built on top of
`go.uber.org/dig`, not a service or runnable product host.

Keep changes centered on the public library contract:

- dependency registration through `NewDependency`, `Replace`, `Decorate`, and
  `CollectDependencies`
- container construction and mutation through `NewContainer`, `Provide`,
  `Invoke`, `Validate`, `Runnables`, graph helpers, and container options
- module scopes through `NewModule`, `WithModules`, and `Private`
- matching/interface binding through `WithMatch`, `WithMatchings`, and
  `NewMatching`
- lifecycle helpers, optional dependency wrappers, override detection, and DOT
  graph export

## Source Of Truth

Read local truth before shared context:

1. `AGENTS.md`
2. `README.md`
3. `docs/README.md`, `docs/en/README.md`, and the topic files under `docs/`
4. `go.mod`, `Makefile`, `.golangci.yml`, `.github/workflows/go.yml`
5. package code and tests in the repository root
6. examples under `examples/`

Known local doc drift: `CONTRIBUTING.md` currently contains copied `goinertia`
wording and Make targets that do not exist in this repository. Do not treat it
as authoritative until it is refreshed.

## Shared Agent Context

Use `$project-context-router` for tasks that need cross-project context or the
shared wiki.

Shared wiki root: `/Users/amir/agents/agent-context`.

After local grounding, read:

- `/Users/amir/agents/agent-context/streams/wiki/index.md`
- `/Users/amir/agents/agent-context/streams/wiki/glossary.md`
- `/Users/amir/agents/agent-context/streams/wiki/platforms/godi.md`

If verified local docs/code conflict with the shared wiki, treat the wiki as
stale. When the task changes a public contract, supported workflow, lifecycle
behavior, or cross-project integration, update the relevant shared platform page
after verification. Do not copy whole README files into wiki; keep shared pages
concise and contract-focused.

## Repository Layout

- Root `*.go`: package `godi` implementation and tests.
- `docs/`: user-facing Russian documentation.
- `docs/en/`: user-facing English documentation.
- `examples/basic`, `examples/modules`, `examples/digout`: runnable examples.
- `Makefile`: local development and verification commands.
- `.github/workflows/go.yml`: CI build, lint, and race-test workflow.

## Public Contract Notes

- Constructors must be functions returning `T` or `(T, error)`. A constructor
  returning only `error` is invalid.
- The first `Invoke` or `Runnables` call marks the container as started; after
  that, `Provide` must fail.
- `Provide` is transactional: a failed append must not poison existing
  container state.
- `Replace` is an explicit slot override. Do not model it as accidental
  last-provider-wins behavior.
- `Replace` is not supported for group dependencies.
- `Decorate` requires an existing slot. It does not support `WithName`,
  `WithGroup`, or `WithMatch` directly; named decoration should be modeled with
  `dig.In` input and `dig.Out` output.
- Decorator group outputs from `dig.Out` are currently rejected.
- `dig.Out` providers can expose multiple slots; do not combine `dig.Out` with
  `WithName`, `WithGroup`, or `WithMatch` on the same dependency.
- `WithMatch` and `NewMatching` require pointers to interfaces, matching
  `dig.As(...)` expectations.
- `WithKey` is diagnostic metadata only. It must not affect resolution.
- `Private` providers are visible only inside their module scope; only public
  module providers participate in root/global resolution.
- `Lifecycle.Start` runs hooks in append order; `Stop` runs them in reverse
  order and joins stop errors.
- If `Lifecycle.Start` fails, already-started hooks are stopped in reverse order.
- Graph helpers are diagnostic. `BuildGraph` has best-effort fallback behavior
  when full resolution fails.

## Development Commands

- `go test ./...`: fast package and example compilation check.
- `go test -race -count=5 ./...`: race detector check used by `make check`.
- `make check`: full local gate: `tidy`, `generate`, `fmt`, `vet`, `lint`,
  `test`, `test-race`, and `cover-html`.
- `make fmt`: runs `go fmt`, `gofumpt`, and `gci`; it rewrites files.
- `make lint`: runs `golangci-lint run -v --fix --timeout=5m ./...`; it may
  rewrite files.
- `make bench-all`: runs all benchmarks with allocation stats.
- `make cover-html`: writes `cover.html`; this artifact is ignored by git.
- `go run ./examples/basic`, `go run ./examples/modules`, and
  `go run ./examples/digout`: runnable smoke examples.

If the sandbox cannot access the default Go build cache, use a local cache:

```bash
mkdir -p tmp/gocache
GOCACHE=$PWD/tmp/gocache go test ./...
```

## Dependencies And Tooling

- The module currently targets the Go version declared in `go.mod`.
- Runtime dependency is intentionally small: `go.uber.org/dig`.
- Test dependency currently comes through `github.com/stretchr/testify`.
- `make fmt` requires `gofumpt` and `gci`.
- `make lint` requires `golangci-lint`.
- Avoid dependency upgrades unless the task explicitly asks for them or a
  verified fix requires them.

## Documentation Rules

- Keep user-facing behavior documented in both `docs/` and `docs/en/` when a
  public API or workflow changes.
- Keep project-maintenance facts in `docs/project.md` and
  `docs/en/project.md`; avoid copying full README content into those files.
- Update examples when changing public usage patterns.
- Keep generated coverage, local caches, `.DS_Store`, IDE folders, and other
  machine-local artifacts out of new commits.

## Verification Expectations

For docs-only changes, verify links and run at least `go test ./...` when
practical. For public behavior changes, add or update focused tests and run the
narrowest relevant package test plus the broader check that proves the affected
surface. Use `make check` before release-oriented changes when the required
format/lint tools are available.
