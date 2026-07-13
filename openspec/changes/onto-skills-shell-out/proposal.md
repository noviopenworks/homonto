## Why

Change A (`onto-binary-authoritative-state`, archived) made the onto Go binary the
single authority for onto workflow state. But the markdown skills still write
state by hand — the `onto*` skills edit `docs/changes/<name>/state.yaml`
directly, and `onto/SKILL.md` still claims onto is "markdown-only" with "no
external CLI." Both are now false. Until the skills stop hand-writing state, the
two planes can still diverge (F1/F3 residue on the skill side).

Grounding change A's surface against what the skills actually write revealed that
A's CLI does **not** cover every field: `onto new` hardcodes `workflow: full` and
sets no `base_ref`/`deps` (`internal/ontocli/new.go:79`), there is no command for
the skill's gated `guides` field (which change A never put in the schema), and the
observational `metrics` fields are carried but have no setter. So the skills
cannot shell out for everything yet. Per the 2026-07-13 scope decision, change B
**bundles** the minimal binary extensions with the skill rewrite.

## What Changes

- **Close the binary CLI-surface gaps (additive; no schema redesign):**
  - `onto new --workflow full|fix|tweak` (stop hardcoding `full`), so the
    fix/tweak presets can create their workflow via the binary.
  - `onto set base-ref <change> <ref>` and `onto set deps <change> <names...>`
    for the fields `onto new` captures today by hand.
  - Add `guides` (pending|updated|`waived: <reason>`) to the state schema as a
    gated core field, plus `onto set guides <change> <value>`.
  - **Observational fields decision (design):** either add minimal setters
    (`onto set metric <phase> <date>`, counts) or drop onto's observational
    tracking (metrics/tasks_total/verify_rounds/preset_escalated are never gated).
    Leaning **drop** — they never gate and re-derive cheaply.
- **Rewrite the `onto*` skills to shell out — zero direct state writes.** Every
  state mutation in onto, onto-open, onto-design, onto-build, onto-verify,
  onto-close, onto-fix, onto-tweak becomes an `onto <command>` invocation
  (`new`/`advance`/`close`/`set …`); reads use `onto state <change> --json` /
  `onto status`. `onto-no-slop` is prose-only and is expected to touch no state.
- **BREAKING (doctrine):** delete the "markdown-only / no external CLI" copy from
  the skills; state onto's hard dependency on the compiled binary.

## Capabilities

### New Capabilities

- (none)

### Modified Capabilities

- `onto-binary`: additive command surface (`onto new --workflow`, `onto set
  base-ref|deps|guides`), and the state-model requirement gains the `guides`
  gated field (+ the observational drop/keep outcome).

## Impact

- **Code:** `internal/ontocli/new.go` (`--workflow` flag), `internal/ontocli/set.go`
  (base-ref/deps/guides setters), `internal/ontostate/state.go` (`guides` field +
  validation; observational drop if chosen) + tests.
- **Catalog:** the 8 state-writing `onto*` skills (`catalog/skills/onto*`) rewritten
  to invoke the CLI; markdown-only copy deleted.
- **Spec:** `openspec/specs/onto-binary/spec.md` delta.
- **Out of scope:** semantic gate *content* / workflow-aware *transition rules* /
  dep resolver (N2 — note `--workflow` here only sets the field, it does not add
  workflow-aware gating); homonto-engine work (gate B); any further schema
  redesign beyond adding `guides`.
- **Design decisions deferred to the design phase:** the observational drop-vs-keep
  call; whether every one of the 8 skills fully maps to commands with no residual
  hand-write; how `deps` is passed (repeatable flag vs comma list); whether a
  thin skill capability spec is warranted or the delta stays on `onto-binary`.
