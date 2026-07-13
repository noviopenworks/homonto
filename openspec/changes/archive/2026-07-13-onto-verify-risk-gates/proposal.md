# onto-verify: risk-based scale trigger + non-waivable finding classes

## Why

Roadmap X3 (F11/F12). The `onto-verify` skill scales verification (light/full)
by change size — light applies to a preset within `≤5 non-test files`. So a
**one-file security change gets light scrutiny** (one optional skeptic, skips
recorded) while a large mechanical refactor gets full — verification keys on
size, not risk (F11). And in light mode a skeptic finding may be waived/skipped
(F12) — the escape hatch is too broad: a security or data-loss finding can be
recorded as a skip.

These are judgments, so per the B1 decision they live in the **skill** (the
agent's risk assessment), not the binary (which enforces ceremony, not
judgment).

## What Changes (onto-verify skill copy)

- **F11 — risk forces full.** The scale check's `full` trigger adds: a diff
  touching a **security-sensitive surface** (secret resolution, remote fetch/
  verify, file deletion/pruning, or permission/ownership) forces `full`
  regardless of file count — a one-file security change is never
  under-scrutinized. `light` is narrowed to "a preset within its limits AND no
  security-sensitive surface."
- **F12 — non-waivable finding classes.** The adversarial triage declares that a
  **security defect, data loss, or a failed core-acceptance scenario** is
  CRITICAL and MUST be fixed — never waived, skipped, or gate-accepted as a
  deviation, in light or full mode. Only lower-severity findings are eligible for
  a recorded skip.

## Impact

- **Specs:** `onto-binary` gains a requirement recording that the onto workflow's
  verification scale is risk-aware and its critical finding classes are
  non-waivable (a skill-enforced, agent-judgment guarantee under B1).
- **Behavior:** the onto-verify skill copy is tightened; no binary/Go change.
- **Risk:** none to the toolchain — a shipped-framework prose change. Verified by
  onto-no-slop discipline and the catalog still loading the skill.

## Non-goals

- Putting risk/finding-class judgment in the binary (B1: the binary enforces the
  presence-shape of the verification token, not the judgment behind it).
- comet-side equivalents (the comet skills are the separate workflow tooling).
