## High-level approach

Two layers, sequenced: finish the binary command+schema surface, then rewrite the
skills against it. Deep design (per-skill command mapping, observational
drop-vs-keep, exact flag shapes) is refined in the design phase.

### Layer 1 — close the CLI-surface gaps (additive)

Building on change A's schema, add only what the skills need to stop hand-writing
state:

- **`onto new --workflow full|fix|tweak`** — `new.go` currently hardcodes
  `Workflow: "full"`. Add a validated flag. This sets the field only; it adds no
  workflow-aware transition *rules* (that is N2).
- **`onto set base-ref <change> <ref>`**, **`onto set deps <change> <names…>`** —
  the two creation fields `onto new` doesn't set. `deps` shape (repeatable flag vs
  comma list) decided in design.
- **`guides` gated field** — add to `ontostate.State` as a gated core field with
  shape `pending|updated|waived: <reason>` (the skill's contract), plus
  `onto set guides <change> <value>`. This is a small schema addition (a
  `schema_version` bump is likely unnecessary — the field is additive and
  legacy-tolerant like the others; confirm in design).
- **Observational fields** — decision point. Lean **drop**: remove
  `metrics/tasks_total/verify_rounds/preset_escalated` from onto's model since
  they never gate a transition and the skills can stop tracking them. Alternative:
  keep and add thin setters. The drop simplifies both planes.

### Layer 2 — rewrite the skills to shell out

For each of the 8 state-writing `onto*` skills, replace every "edit state.yaml"
instruction with the corresponding `onto` command:

| Skill action | Command |
|---|---|
| create change (+ workflow) | `onto new <name> --workflow <w>` |
| capture base_ref / deps | `onto set base-ref`, `onto set deps` |
| advance phase | `onto advance <name>` |
| record isolation/exec/tdd/directive | `onto set isolation|build-mode|tdd-mode|directive` |
| record verify scale/result | `onto set verify-scale|verify-result` |
| mark close.merged / guides | `onto set close-merged|guides` |
| archive | `onto close <name>` |
| read current state | `onto state <name> --json` / `onto status` |

Then **delete the "markdown-only / no external CLI" copy** from `onto/SKILL.md`
and any sibling, and state the hard binary dependency. `onto-no-slop` is
prose-discipline and should need no change (verify).

### Verification strategy

- Binary extensions: TDD, same shape as change A (happy + shape-reject per
  command; `--workflow` validation; `guides` shape).
- Skills: a **grep-based gate** proving no `onto*` skill contains a direct state
  write (no `state.yaml`/`onto-state.yaml` edit instruction) and no
  "markdown-only / no external CLI" copy — the enforceable form of the exit gate.
  A full-lifecycle skill dry-run belongs to N7 (the onto E2E suite), not here.

## Non-goals

- Semantic gate content, workflow-aware transition *rules*, dep resolver (N2).
- Homonto-engine / projection work (gate B).
- Any schema redesign beyond adding `guides` (and the observational drop).

## Risks

- **A skill writes a field with no command even after Layer 1** — mitigated by an
  explicit per-skill field→command audit in design before any rewrite.
- **Dropping observational loses history other tooling reads** — check no skill or
  doctor path depends on `metrics` before dropping; if any does, keep + add a
  setter instead.
