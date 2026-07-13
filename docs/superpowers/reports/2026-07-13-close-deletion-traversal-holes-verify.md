# Verification Report — close-deletion-traversal-holes

**Date:** 2026-07-13 · **Change:** ROADMAP N4 (gate B, engine safety) · Comet tweak · verify_mode full

## Result: PASS

- **F28** — `validateResources` now applies the `local:` plain-name check (shared
  `validateLocalPlainName` helper, called by both subagents and skills/commands so
  they can't drift). Traversal `local:../x` skill/command rejected at load; plain
  `local:x` accepted. Tests: `TestResourcesRejectTraversalLocalSource`,
  `TestResourcesAcceptPlainLocalSource`.
- **F7** — copy-mode prune confined at the `copyfile.Apply` choke point:
  `Apply(ops, pruneRoots)` refuses (retains ownership) any destination resolving
  outside the managed provider roots (incl. `..` traversal); fail-closed on an
  empty root set. Adapters supply `copyPruneRoots()`. Tests:
  `TestApplyConfinesPruneToRoots`, `TestApplyRefusesAllPrunesWithoutRoots`.
- Delta aligned to the shipped root-confinement design (tool-adapters + config-model).

## Evidence
`go test ./internal/config/... ./internal/copyfile/... ./internal/adapter/... -race` → 195 passed;
`go vet` clean; `go build` success; `openspec validate --all` 16/16.

## Standard review
Implemented via TDD subagent; layering choice (choke-point confinement over per-adapter
filter) reviewed and accepted — one enforcement point both adapters route through.

## Out of scope
N5 (remote transactionality), N6 (no-follow writer, locking) — remain open gate-B RC blockers.
