# Verification Report — adapter-contract-codex-pilot

- **Date:** 2026-07-12
- **Change:** adapter-contract-codex-pilot (roadmap item 11 — Ecosystem Expansion)
- **Mode:** full
- **Base-ref:** `9971404` → HEAD (`feature/20260712/adapter-contract-codex-pilot`)

## Result: PASS

### 1. Tasks complete

All 7 task groups in `tasks.md` are `[x]` (task 6 recorded as a deferred
same-behavior follow-up). `grep -c '- \[ \]' tasks.md` → 0.

### 2. Implementation matches design

Follows `openspec/changes/.../design.md` and the Design Doc: `internal/tomlutil`
(TOML codec), `internal/adapter/structproj` (the Codec-parameterized projection
contract), `internal/adapter/codex` (pilot on the contract), config + engine
wiring. Codex is a thin adapter (file path + desired() + codec); all
plan/apply/observe control flow lives in the contract.

### 3. Spec scenario coverage

| Delta spec requirement | Verifying test |
|---|---|
| adapter-contract: projection core | `TestProjectCreateThenNoop`, `TestProjectUpdateAndDelete`, `TestProjectAdoptsPreexisting`, `TestProjectSecretRedaction` |
| adapter-contract: Codec abstracts format | tomlutil + structproj suites (TOML-driven core) |
| adapter-contract: compatibility fixture | `TestCodexCompatibilityFixture` |
| codex-adapter: MCP projection | `TestCodexProjectsMCP`, `TestCodexDottedMCPName` |
| codex-adapter: prune / idempotency | `TestCodexPrunesDeDeclared`, `TestCodexDeleteDoesNotRecreateAbsentFile` |
| codex-adapter: secret safety | `TestCodexPlanDoesNotRevealSecret` |
| tool-adapters: codex target (opt-in) | `TestCodexTargetAccepted`, `TestCodexRejectedForNonMCPKinds` |

### 4. proposal.md goals satisfied

The adapter contract is published (structproj + Codec, JSON and TOML codecs); a
third adapter (Codex) ships on it without duplicating the Claude/OpenCode control
flow; the compatibility fixture proves surgical merge, idempotency, prune, and
unmanaged-content preservation. Codex is opt-in (MCP-only); defaults unchanged.

### 5. Security / review

High-effort workflow review completed; all correctness findings fixed
(`1404ecd`): codex MCP-only target scope (avoids unloadable config), dotted-name
quoting, and delete-not-recreating-absent-file; plus dedup cleanups. Secret
guarantees preserved (plan never resolves a secret). No CRITICAL findings into
verify.

### 6. Gate evidence

`scripts/gate.sh` re-run after the review fixes → govulncheck clean (pinned
go1.26.5), `go test -race ./...` → 25 packages green, dual-binary Docker E2E →
ALL SUITES PASS; `openspec validate adapter-contract-codex-pilot` → valid.

### Scope note (not a gap)

Catalog governance and migrating Claude/OpenCode onto the contract are tracked
follow-ups (task 6); the exit gate — adapter contract published + one pilot
adapter green — is met by Codex-on-contract.
