# Brainstorm Summary

- Change: homonto-v1-core
- Date: 2026-07-03

## Confirmed Technical Approach

Normalized desired-state model + per-tool adapters (Claude, OpenCode) with shared
services (secret resolver, symlink linker, surgical JSON/JSONC merge, hashed state
store, plan printer). Six-stage apply pipeline: parse → read → plan → confirm →
resolve-all → write (state last). Implements the existing 14-task TDD plan.

Core design decision (user-approved): **hashed-state secret idempotency**. State
stores `{desired: <unresolved token>, applied: sha256(resolved value)}` per managed
key. Plan is a noop for a secret-backed key iff its token matches state and
`sha256(on-disk)` matches the stored hash; otherwise an update. Any change on a
secret-bearing key redacts `Change.Old` to `«secret»`, so plan output and
`state.json` never contain plaintext.

## Key Trade-offs and Risks

- Chose hash + drift detection over token-only (simpler but no drift) and over
  resolve-at-plan / plaintext-in-state (both violate safety rules).
- JSONC comment loss in rewritten `opencode.jsonc` regions — documented caveat.
- sha256 of low-entropy secrets — fine for real API keys; documented.
- Value-formatting false diffs — mitigated by JSON normalization before compare/hash.

## Testing Strategy

TDD, table-driven per package; golden-file surgical-merge tests; secret-safety
tests covering both create and the drift-of-a-secret-key update path; two-phase
abort test; e2e that asserts a second apply (including a secret-backed MCP) is a
no-op. Full suite: `go test ./... && go vet ./... && go build ./...`.

## Spec Patches

None — delta specs authored fresh this phase (config-model, apply-pipeline,
secret-references, tool-adapters, cli-commands); `openspec validate` passes.
