---
comet_change: onto-skills-shell-out
role: technical-design
canonical_spec: openspec
archived-with: 2026-07-13-onto-skills-shell-out
status: final
---

# onto-skills-shell-out — Technical Design

N1 change B: make the `onto*` markdown skills invoke the binary for every state
mutation (zero direct state writes) and delete the "markdown-only / no external
CLI" claim. Bundles the minimal binary extensions needed to close change A's
CLI-surface gaps. Decisions taken at recommended defaults (/goal autonomous).

## Context (verified 2026-07-13)

- Change A made `internal/ontostate` + `internal/ontocli` the state authority
  (versioned schema, `onto set {isolation,build-mode,tdd-mode,verify-scale,
  verify-result,close-merged,directive}`, `onto state --json`, status/doctor
  classify).
- Gaps: `onto new` hardcodes `Workflow:"full"` and sets no `base_ref`/`deps`
  (`new.go:79-83`); no `guides` command (guides not in change A's schema); no
  observational setters.
- 8 skills write state (`catalog/skills/onto*/SKILL.md`); `onto-no-slop` is
  prose-only (0 state refs). Metrics are written by ~8 skills but never gate
  (per the 2026-07-13 review).

## Design

### Layer 1 — binary extensions (additive)

1. **`onto new --workflow full|fix|tweak`** (`new.go`): validated flag, default
   `full`. Sets the field; adds no transition rules (N2).
2. **`onto set base-ref <change> <ref>`** and **`onto set deps <change> <names…>`**
   (`set.go`): base-ref is presence-only (any non-empty ref); deps passed via a
   repeatable `--dep` flag (decided over comma-split for names with edge chars).
   Both go through `runTransition` (load→apply→Validate→Save; write-nothing-on-fail).
3. **`guides` gated field** (`state.go`): add `Guides string` to the core with
   shape `pending | updated | waived: <reason>` — validate: value is `pending`,
   `updated`, or has prefix `waived:`. Empty allowed (legacy-tolerant), so **no
   schema_version bump**. `onto set guides <change> <value>` via `enum-like`
   validation (custom, because of the `waived:` prefix).
4. **Observational: drop, not extend.** No setters. The skill rewrite removes the
   metric-writing instructions; the binary keeps carried `Observed` (now empty).
   Metrics never gated → acceptable. No model change required.

### Layer 2 — rewrite the 8 skills to shell out

Field→command map (the audit result — every gated field has a command after
Layer 1):

| State field | Command |
|---|---|
| change, workflow, phase(open), created | `onto new <name> --workflow <w>` |
| base_ref | `onto set base-ref <name> <ref>` |
| deps | `onto set deps <name> --dep …` |
| phase (advance) | `onto advance <name>` |
| decisions.isolation/execution/tdd/directive | `onto set isolation|build-mode|tdd-mode|directive` |
| verify.mode/result | `onto set verify-scale|verify-result` |
| close.merged | `onto set close-merged` |
| guides | `onto set guides` |
| archived | `onto close <name>` |
| metrics.* (observational) | **removed** (no longer written) |
| read current state | `onto state <name> --json` / `onto status` |

Each skill's "edit `state.yaml`" step is replaced with the mapped invocation.
Then delete the "markdown-only / no external CLI" copy from `onto/SKILL.md` (and
any sibling) and state the hard binary dependency. `onto-no-slop` unchanged.

### Verification

- Binary: TDD per command (happy + shape-reject; `guides` shape incl. `waived:`).
- Skills: a **grep-based CI gate** (`scripts/onto-skills-shell-out-check.sh` or
  folded into `gate.sh`) asserting no `catalog/skills/onto*` file contains a
  direct state-file write instruction (`state.yaml`/`onto-state.yaml` edit) and
  none contains the markdown-only/no-CLI copy. This is the enforceable form of
  the exit gate; a full lifecycle dry-run is N7 (the onto E2E suite).

## Non-goals

Workflow-aware transition rules, semantic gate content, dep resolver (N2);
homonto-engine work (gate B); observational setters; schema redesign beyond the
additive `guides`.

## Risks

- A residual hand-write after Layer 1 → the grep gate catches it; the field→command
  map above is complete for every gated field.
- Dropping observational loses metric history → accepted; never gated; no reader
  found that uses it for a decision.
