# Implementation Notes

Date: 2026-06-04.

Task: initialize the repository documentation context, fill `AGENTS.md`, and add
focused project facts under `docs/`.

## Decisions

- Kept the repository documentation scope docs-only; no code, config, or test
  behavior was changed.
- Preserved the existing bilingual documentation structure by adding both
  `docs/project.md` and `docs/en/project.md`.
- Treated local files as the source of truth before shared wiki context. The
  shared `platforms/godi.md` page matched the local high-level contract, so no
  shared wiki edit was needed.
- Documented `CONTRIBUTING.md` as current doc debt instead of silently relying
  on it, because it still references `goinertia` and nonexistent Make targets.
- Included a local `GOCACHE` fallback in project guidance because a sandboxed
  command hit permission errors against the default Go build cache.

## Verification Snapshot

- `go test ./...` passed before documentation edits.
- `GOCACHE=/Users/amir/dev/projects/my/godi/tmp/gocache go test ./...`
  passed after documentation edits.
