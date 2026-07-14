# Guides

User-facing documentation, one topic per file. Guides explain how to *use* the
system; the OpenSpec capability specs (`openspec/specs/`) define what it *must
do*.

- [`getting-started.md`](getting-started.md) — hands-on walkthrough of both
  binaries with real command output, plus a supported / not-supported matrix.
  **Start here.**
- [`using-homonto.md`](using-homonto.md) — the `homonto` CLI: config shape,
  projection behavior, status/adoption, and known limitations.
- [`status-and-adoption.md`](status-and-adoption.md) — state adoption, drift,
  pending changes, and pruning behavior.
- [`comet-workflow.md`](comet-workflow.md) — this repository's development
  workflow (Comet + OpenSpec + Superpowers).
- [`onto-workflow.md`](onto-workflow.md) — the `onto` binary and its spec-driven
  workflow, shipped as a bundled product framework.
- [`onto-flow-and-gates.md`](onto-flow-and-gates.md) — precise reference for how a
  change enters the onto workflow, advances between phases, and exits, with every
  entry/exit gate the binary enforces.
- [`enforcement.md`](enforcement.md) — making onto's gates non-skippable at the
  tool boundary with hooks (`onto doctor --quiet` + Claude `settings.json` hooks /
  an OpenCode plugin).
- [`subagents.md`](subagents.md) — the `[subagents.*]` resource: sources
  (builtin/local/remote), link vs copy mode, scope/targets, model routes, remote
  pinning, and lifecycle.

Living capability specs are in [`../../openspec/specs/`](../../openspec/specs/);
change history is in
[`../../openspec/changes/archive/`](../../openspec/changes/archive/). The release
gate lives in [`../release-checklist.md`](../release-checklist.md).
