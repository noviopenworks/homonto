## Why

`openspec validate --all` reports every canonical spec valid, yet three living
specs describe a command surface and capability that no longer ship. The
imperative `homonto agents list/add/update/doctor/prune` group, its
content-addressed base-blob store (`agentblob`), and its lockfile-driven prune
(`agentlock`) were removed when `[agents]` was collapsed into the declarative
`[subagents]` + `apply` model (item 9, `38b32ec`). The CLI now registers only
`plan/apply/status/doctor/init/import` (`internal/cli/root.go:28`); `agentlock`
and `agentblob` no longer exist; `[agents.<name>]` still parses but is nilled out
by the loader (`internal/config/config.go:526`).

Because the validator checks form, not correspondence to reality (F5), validation
currently proves syntax, not truth — the single most corrosive fact the
2026-07-13 review surfaced. This is a hard blocker for the `v0.1.0-rc.1` tag
(ROADMAP N3): a public release must not ship specs that mandate a removed command
group.

## What Changes

- **BREAKING (spec truth):** `cli-commands` no longer specifies the `agents`
  command group. The command surface requirement and the agent-subcommand
  scenarios are corrected to the six commands the CLI actually registers, plus
  `version`.
- `agent-lifecycle` is retired or reduced to describe only shipped behavior. The
  imperative-command / `agentblob` / `agentlock` requirements describe removed
  code and are removed; whether a thin capability remains to document the
  `[agents.<name>]` parse-then-fold deprecation shim is decided in `design.md`.
- `config-model` requirement "Lifecycle-managed agents SHALL be declarable as
  `[agents.<name>]`" is corrected: `[agents.<name>]` still parses but is folded
  into `[subagents]` and carries no separate lifecycle (matching
  `config.go:526`).
- CI gains a **coarse spec↔code correspondence check**: a spec that names a
  `homonto <command>` the CLI does not register fails, so a form-valid-but-false
  spec cannot pass again.
- README (`:118`) and `docs/guides/using-homonto.md` (`:14`) are **verified
  already correct** (they already say the table is folded with no imperative
  group) — no edit expected, only confirmation.

## Capabilities

### New Capabilities

- (none)

### Modified Capabilities

- `cli-commands`: the command-surface requirement and agent-subcommand scenarios
  change to match the registered command set.
- `agent-lifecycle`: removed-behavior requirements are deleted; the capability is
  retired or reduced to the deprecation shim (resolved in design).
- `config-model`: the `[agents.<name>]` declarability requirement changes from
  "lifecycle-managed, declarable" to "parses but is folded into `[subagents]`,
  no separate lifecycle."

## Impact

- **Specs:** `openspec/specs/cli-commands/spec.md`,
  `openspec/specs/agent-lifecycle/spec.md`,
  `openspec/specs/config-model/spec.md`.
- **CI:** one new correspondence check (script + workflow wiring; a grep-level
  gate is sufficient).
- **Docs:** README and `using-homonto` verified only; `docs/superpowers/*`
  historical design docs are explicitly **out of scope** (they correctly record
  the removal; their active-only contradiction is F19, not this change).
- **No code behavior change.** The observation that `[agents]` parses then nils
  silently (a possible F35-adjacent silent no-op) is noted but **out of scope** —
  N3 aligns spec truth, it does not change config behavior.
- **Release:** unblocks the `v0.1.0-rc.1` truth gate.
