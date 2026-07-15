# Guides

User-facing documentation, one topic per file.

## First steps

- [`getting-started.md`](getting-started.md) — hands-on walkthrough of both
  binaries with real command output, plus a supported / not-supported matrix.
  **Start here.**

## Reference

- [`configuration.md`](configuration.md) — every `homonto.toml` table and
  field, defaults, and the fail-fast validation rules.
- [`cli-reference.md`](cli-reference.md) — every `homonto` command: flags,
  exit codes, and examples.
- [`onto-reference.md`](onto-reference.md) — every `onto` command, the phase
  flow, and every entry/exit gate the binary enforces.

## Concepts

- [`secrets.md`](secrets.md) — `${pass:…}` / `${ENV_VAR}` references and the
  referenced-never-stored guarantees.
- [`projection-and-state.md`](projection-and-state.md) — the apply pipeline:
  surgical merge, symlinked content, state, drift vs. pending, adoption, and
  pruning.
- [`subagents.md`](subagents.md) — the `[subagents.*]` resource: sources
  (builtin/local/remote), link vs. copy mode, scope/targets, model routes, and
  the tool-neutral `homonto:` frontmatter block.
- [`remote-source-trust.md`](remote-source-trust.md) — pinned, fail-closed
  remote installs: threat model, verification pipeline, and lifecycle.

## The onto workflow

- [`onto-workflow.md`](onto-workflow.md) — concepts: the binary/skills split,
  the five phases, presets, and the specialist subagents.
- [`enforcement.md`](enforcement.md) — making onto's gates non-skippable at the
  tool boundary with hooks (`onto doctor --quiet` + Claude `settings.json`
  hooks / an OpenCode plugin).

## When something looks wrong

- [`troubleshooting.md`](troubleshooting.md) — known limitations, gotchas, and
  workarounds for both binaries.

## Developing homonto itself

- [`comet-workflow.md`](comet-workflow.md) — this repository's development
  workflow (Comet + OpenSpec + Superpowers). These are **external** tools the
  maintainers use; homonto does not bundle them (see
  [ADR 0015](../adr/0015-ship-only-onto-frameworks.md)). See also
  [`../personas.md`](../personas.md) for why we build with Comet but ship onto,
  [`../adr/`](../adr/) for durable architecture decisions, and
  [`../release-checklist.md`](../release-checklist.md) for the release gate.
