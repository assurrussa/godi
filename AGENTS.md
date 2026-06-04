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

Use `$project-context-router` when a task needs cross-project context
from a local shared wiki.

Do not hard-code machine-local absolute paths in this public repository.
If a local shared wiki is available, expose its root through
`AGENT_CONTEXT_ROOT` or let `$project-context-router` resolve it for the
current session.

Local docs and code in this repository remain the source of truth for
commands, public APIs, config keys, supported imports, runtime behavior,
and release gates. Read this repo's `AGENTS.md`, `README.md`, `docs/`
or `reference/`, task files, code, tests, and configs before shared wiki
pages.

When shared context is available, read these paths from the resolved wiki
root:
- `streams/wiki/index.md`
- `streams/wiki/glossary.md`
- `streams/wiki/platforms/godi.md`

If local verified docs/code conflict with the shared wiki, treat the wiki
as stale and update the relevant platform page after verification. Do not
copy whole README files into wiki; keep shared pages concise and
contract-focused.
