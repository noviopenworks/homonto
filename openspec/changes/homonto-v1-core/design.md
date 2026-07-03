## Context

`homonto` v1 is a personal Go CLI: the single declarative source of truth for AI
coding-tool config, projecting `homonto.toml` into Claude Code and OpenCode. The
detailed technical RFC and the six-stage pipeline live in
`docs/superpowers/specs/2026-06-24-homonto-design.md`; a task-by-task TDD plan
exists in `docs/superpowers/plans/2026-06-24-homonto.md`. This document records
the **high-level architecture decisions** for the change; the deep Design Doc and
delta capability specs are produced in the Comet design phase.

## Architecture

Normalized desired-state model + per-tool adapters with shared services:

```
homonto.toml ──▶ Parse ──▶ DesiredState ──▶ [ ClaudeAdapter, OpenCodeAdapter ]
                                                   │ Read → Plan → Apply
shared: SecretResolver · ContentLinker · Planner/Printer · StateStore
```

Everything downstream operates on the tool-agnostic `Config`/`DesiredState`, never
on raw TOML. Adding a tool later = implement one `Adapter`, no engine changes.

## Key Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Core model | Declarative source of truth; tools are generated outputs | Reproducible, reviewable, one source |
| Secrets | Reference-only (`${pass:…}` primary, `${ENV}` fallback), resolve after confirm, all-at-once before any write | Plans/logs safe to share; no half-apply |
| Merge safety | Surgical: write only managed keys, preserve unmanaged | Never destroy a user's hand-tuned config |
| Writes | Atomic temp+rename; `state.json` written last | Crash-safe; next apply reconciles |
| Owned content | Symlinks (not copies) in v1 | Edit once, live everywhere; `--copy` later |
| Config format / stack | TOML; cobra, go-toml/v2, sjson/gjson, hujson | Matches design spec |
| **Idempotency of secrets** | State stores `{desired: <unresolved token>, applied: sha256(resolved)}` per managed key | Roadmap-required; keeps repeat plans no-op, detects drift, never stores/prints plaintext |

## Secret-idempotency model (the change's core design point)

Problem: the original plan compared the *unresolved* desired value against the
*resolved* on-disk value, so any secret-backed key showed a spurious `~ update`
every run, and a drift/update would have printed the resolved secret into `Change.Old`.

Resolution — per managed key, at plan time:

```
disk absent                                                   → create
desired has NO secret ref:  disk == desired ? noop : update   (direct compare)
desired HAS secret ref:     in-state
                            && state.desired == desired
                            && state.applied == sha256(disk)  ? noop : update
```

- On `apply`, after resolving each change's value, store
  `state.Set(tool, key, {desired: unresolved, applied: sha256(resolved)})`.
- For **any** change on a secret-bearing key, `Change.Old` is redacted (`«secret»`),
  never the on-disk resolved value — so `plan` output stays plaintext-free.
- `state.json` holds only unresolved tokens + hashes → safe to share.

## Alternatives considered

- **Token-match only, no hashing** — simpler, but loses drift detection for
  secret-backed values (can't tell if the on-disk key changed). Rejected: the
  hash is cheap and the roadmap explicitly asks for drift-safe idempotency.
- **Resolve secrets at plan time to compare** — violates "plan never touches
  `pass` / never resolves". Rejected.
- **Store plaintext resolved value in state** — violates "nothing secret in the
  repo/state; `state.json` is shareable". Rejected.

## Scope boundaries

In: config model, apply pipeline (two-phase, atomic, idempotent, drift), secret
references + hashed state, Claude + OpenCode adapters (surgical + symlinks), the 6
CLI commands, and the full test matrix (unit/golden/e2e/secret-safety/idempotency).

Out (roadmap v1.1+): built-in templates, richer plugin configuration, tool TUI
configuration, agent lifecycle; encrypted in-repo secrets; imperative add/remove;
tools beyond Claude Code + OpenCode; preserving JSONC comments inside rewritten
regions (documented caveat, not a goal).

## Risks

- **JSONC comment loss** in rewritten `opencode.jsonc` regions — documented caveat.
- **Low-entropy secrets + hash** — sha256 of a resolved value is safe for
  high-entropy API keys; documented that references should hold real secrets.
- **Idempotency value-formatting mismatches** (e.g. `"opus"` vs `opus`) — both
  sides normalized through JSON marshal before comparison.
