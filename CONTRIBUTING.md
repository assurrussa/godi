# Contributing to godi

`godi` is the Go library module `github.com/assurrussa/godi`.
Keep changes focused on its public DI, module, lifecycle, and diagnostic contracts.

## Prerequisites

- Go at least `1.25.4`, as declared in `go.mod`.
- `golangci-lint` `v2.13.1`, matching CI. Its bundled formatters are used by Make.

## Development

Create a branch, add regression coverage for behavior changes, and update both
Russian (`docs/`) and English (`docs/en/`) documentation when contracts change.
Tests should exercise the public package API.

- `make check`: verify module tidiness, formatting, vet, lint, and race tests;
  does not rewrite repository files.
- `make fix`: run tidy, generation, formatting, and automatic lint fixes.
- `make test`: run all tests once.
- `make test-race`: run all tests five times with the race detector.
- `make fmt` / `make fmt-check`: apply / check formatting.
- `make lint` / `make lint-fix`: inspect / automatically fix lint findings.
- `make cover-html`: explicitly generate the ignored `cover.html` report.
- `make bench-all`: run benchmarks with allocation measurements.

Run examples directly:

```bash
go run ./examples/basic
go run ./examples/modules
go run ./examples/digout
```

CI tests both the exact Go version in `go.mod` and current stable, with automatic
toolchain switching disabled. Lint runs once on stable with the pinned version.
The job has a 15-minute timeout and superseded runs for the same ref are cancelled.
See [setup-go inputs](https://github.com/actions/setup-go/blob/v6/action.yml) and
[workflow syntax](https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-syntax).

## Pull Requests

Run `make check` before submitting a pull request to `master`. Describe the
behavior change and relevant validation. Do not include generated coverage files.

Contributions are licensed under the project's MIT License.
