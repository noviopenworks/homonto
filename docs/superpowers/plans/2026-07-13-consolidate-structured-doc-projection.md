---
change: consolidate-structured-doc-projection
design-doc: docs/superpowers/specs/2026-07-13-consolidate-structured-doc-projection-design.md
base-ref: 5146750a14934e3e7cb35fb2a6ba27fbf43029b1
---

# Plan — consolidate structured-doc projection

Safety net: `internal/adapter/conformance` + all claude/opencode tests must
stay green after every step. Any output diff = migration bug, fixed in code.

## Task 1 — shared JSON codec (TDD: new code)
Add `internal/adapter/jsoncodec` implementing `structproj.Codec` over
`internal/jsonutil`. RED: codec_test round-trips Get/Set/Delete/Canonical and
normalizes empty→{}. GREEN: thin delegations. Commit.

## Task 2 — claude settings.json namespace
Route `setting.*` through structproj.Project/Apply/Observe (pathFor = settings
path). Delete that branch of the bespoke loop. Run claude + conformance -race.
Commit.

## Task 3 — claude .claude.json namespaces
Route `mcp./plugin./pluginconfig./marketplace.` through structproj. Delete
those branches. Run claude + conformance -race. Commit.

## Task 4 — opencode opencode.json namespace
Route `mcp./setting.` through structproj. Delete the bespoke loop. Run
opencode + conformance -race. Commit.

## Task 5 — scope confirm + full verify
Confirm file-projection paths untouched. Full: go test ./... -race, vet, build,
openspec validate --all. Commit.
