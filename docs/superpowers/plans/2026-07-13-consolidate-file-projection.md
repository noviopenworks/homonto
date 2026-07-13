---
change: consolidate-file-projection
design-doc: docs/superpowers/specs/2026-07-13-consolidate-file-projection-design.md
base-ref: 7008b18ab961c4b749781da4b44a33aae1e77769
archived-with: 2026-07-13-consolidate-file-projection
---

# Plan — consolidate file-projection

Safety net: `internal/adapter/conformance` + per-adapter link tests
(scope/adopt/observehashes/pruning/robustness) green after every step.

## Task 1 — fileproj contract (TDD, new code)
Add `internal/adapter/fileproj` (Link, Project, Conflicts, ApplyState,
ApplyLinks, Observe + recordedDst + " -> " const). Table-driven unit tests
covering create/relocate/relink/adopt + no-delete + observe. Commit.

## Task 2 — claude skills → commands → subagents
Migrate each namespace onto fileproj (skills canary first). Narrow inline
adopt/delete loop per step; delete it after subagents; drop dead recordedDst.
claude + conformance suites -race green per step. Commit per namespace.

## Task 3 — opencode skills → commands → subagents
Same sequence; drop dead recordedDst. opencode + conformance -race green. Commit.

## Task 4 — verify + scope confirm
Copy-mode + internal/link untouched; generic delete loop unchanged. Full:
go test ./... -race, vet, build, openspec validate --all. Commit.
