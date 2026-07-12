---
change: adapter-contract-codex-pilot
design-doc: technical-design.md
base-ref: 9971404
archived-with: 2026-07-12-adapter-contract-codex-pilot
---

# Implementation Plan — Adapter Contract + Codex Pilot

TDD per task group. `go test ./internal/...` after each; full gate at the end.
Codex-on-core proves the exit gate; existing Claude/OpenCode adapters are left
untouched (their tests must stay green) unless task 6's guarded migration keeps
them green.

## Task 1 — tomlutil (TOML codec)
- `internal/tomlutil/tomlutil.go`: `EnsureRoot`, `Get(doc,path)→canonicalJSON`,
  `Set(doc,path,jsonValue)`, `Delete(doc,path)`, `Canonical(jsonValue)`.
  map[string]any tree via go-toml/v2; preserve unmanaged tables; prune emptied
  managed parents on delete.
- Tests first: set/get/delete round-trip, unmanaged preserved, empty/whitespace,
  nested `mcp_servers.x.env`, canonical number/key-order stability.

## Task 2 — structproj (projection core / contract)
- `internal/adapter/structproj/structproj.go`: `Codec` iface; `Project(desired,
  disk, st, ns, codec)`, `Apply(changes, disk, codec, res, st, ns)`,
  `Observe(st, disk, ns, codec)`. Mirror the Claude/OpenCode diff loop
  (create/update/noop/adopt/delete + secret redaction).
- `internal/adapter/structproj/jsoncodec.go`: JSON codec wrapping jsonutil.
- Tests first: JSON-codec-driven create/update/noop/adopt/delete + secret
  redaction, proving fidelity.

## Task 3 — Codex adapter
- `internal/adapter/codex/codex.go`: `New(home)`, `Name()=="codex"`,
  `~/.codex/config.toml`, `desired()` (MCPs targeting codex → `mcp_servers.<n>`),
  Plan/Apply/ObserveHashes delegating to structproj+tomlutil, ns="mcp_servers".
- Tests first: projection, prune, idempotency; secret-safety (plan never prints
  a resolved secret).

## Task 4 — config + engine wiring
- config: add `codex` to known tools (target validation); default targets stay
  claude+opencode (codex opt-in). Test unknown-target still rejected.
- engine.Build: append codex adapter; plan/status/doctor include it.

## Task 5 — compatibility fixture suite
- `internal/adapter/codex/testdata/config.toml` (model + user_owned server).
- `compat_test.go`: apply projects declared + preserves unmanaged; idempotent
  re-plan; de-declare prunes declared, keeps unmanaged; non-managed value not
  clobbered.

## Task 6 — guarded existing-adapter migration (optional)
- Route Claude/OpenCode MCP structured-file slice through structproj IFF every
  existing test stays green; else revert and record follow-up.

## Task 7 — docs, ADR, gate
- ADR: adapter contract (staged in change). Guide/README: Codex target.
- Roadmap item 11 → done (contract + pilot) with evidence.
- Gate: `go test -race ./...`, `./scripts/gate.sh`.

## Verification
Each task: `go test ./internal/... -race`. Final: full gate. Behavior fidelity of
structproj checked against ported JSON adapter tests.
