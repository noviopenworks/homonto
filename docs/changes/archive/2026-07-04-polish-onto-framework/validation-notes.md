# Validation Notes: polish-onto-framework

## Mechanical checks (task 4.3, 2026-07-04)

- Self-containment: `grep -rn "openspec|comet|docs/superpowers"
  content/skills/` → no matches (incl. all 13 references/ files)
- Derivation tables: dispatcher vs docs/changes/README.md → byte-identical
- All 13 reference files present
- Regression: `go test ./...` → all packages pass (no Go changes)

## Dry-run A: full lifecycle (fresh-context agent, 2026-07-04)

Checklist 8/8 areas walked; PASS on templates, notes.md protocol
(compaction recovery verified: notes.md = why, derivation = where),
design gate/Status keys, verify template + adversarial followability,
deps/metrics naming consistency across all documents, and re-derivation
at every phase boundary (including the gates-win-upward path and
two-active-change discovery). FAILs raised as defects D1–D16.

## Dry-run B: lint, adversarial, deps, drift (fresh-context agent, 2026-07-04)

Lint caught all six deliberate delta malformations (missing SHALL, no
scenario, malformed scenario, invalid heading, bad RENAMED format,
nonexistent MODIFIED target). Adversarial triage walk PASS (refuted claim
→ scenario fail → gate; no-capability and light-skip paths specified).
FAILs: RENAMED merge ordering, dep-archived check, ship.md immutability —
raised as defects 1–7.

## Defects found: 20 distinct (7 + 16, overlapping ≈3) — ALL FIXED same build

1. RENAMED→MODIFIED merge order now explicit (RENAMED→MODIFIED→REMOVED→
   ADDED) in close SKILL + specs README; lint/delta-template carve-out
   for MODIFIED targeting a RENAMED TO-name.
2. plan.md template gained the `- [ ] done` checkoff line all three
   skills mandate (was literally impossible to comply).
3. Lint staging header fixed; guides check moved to §4 pre-archive (was
   ordered before guides could be resolved).
4. ship.md sanctioned as the single post-archive addition (README archive
   contract + close step 4 + handoff reference); notes.md + ship.md added
   to the workspace contents table.
5. Dep-archived check defined (`archive/*-<dep>/` suffix match);
   nonexistent dep = finding to correct or drop.
6. Field-level rebuild granularity defined (ill-typed/missing field →
   that row only; decisions never reset by field repair); lint type
   checks added; this change's own pre-v2 state.yaml backfilled
   (deps: [], metrics) as the live instance.
7. Lint scenario quantifier (EVERY scenario well-formed + ≥1 per req);
   absent-living-file finding; "first non-empty line after the heading".
8. Skip bookkeeping: protocol-mandated adversarial skips live in the
   Adversarial section, no acceptor.
9. Subagent protocol: isolation target + commit sha in dispatch/return;
   clean-tree rule before re-dispatch; CRITICAL fixes via re-dispatched
   implementer.
10. Mid-build design revisit: `phase: design` + `Status: Under revision`
    flow specified.
11. Wording: metrics initialization, base_ref definition, DEFERRED
    restricted to close, notes.md scope (conversation-shaped phases),
    verify scale range `base_ref..HEAD`, proposal Grounding section.

## Adversarial verify round 1 (2026-07-04) — FAIL → user chose fix-all

Conformance skeptic: most scenarios held (deps flow, RENAMED ordering,
subagent protocol, ship consistency, byte-identical tables); refuted:
notes.md "every skill" over-claim, skip-recording contradiction across 4
documents, lint §3 grep unsatisfiable; workspace template drift.
Robustness skeptic: 14 findings, 5 CRITICAL (lint grep; REMOVED-template
SHALL contradiction; dep suffix-match false positives with live
counterexamples 'core'/'workflow'; workflow field never cross-checked;
preset upgrade not durable) + gate-skip on rebuild, Under-revision
derivation gap, vacuous all-tasks-checked, DEFERRED-no-consumer,
parallel-dispatch race, ship re-offer rigidity, graphify staleness.

All 16 triaged findings fixed: date-anchored dep match + active-overrides
+ cycle finding; workflow cross-check (dispatcher rule 4); durable upgrade
annotation in the Preset marker; gate-protected rebuild (notes.md
Confirmed consulted); derivation rows (Under revision; ≥1 task) in both
copies; lint SHALL scoped to ADDED/MODIFIED, heading-anchored §3 grep,
full-artifact conformance check, DEFERRED consumer in §4 + close entry;
serial-by-default subagent dispatch; skip recording harmonized to the
Adversarial section; notes.md read added to verify/close/presets + spec
reworded to phase skills / preset SHOULD; ship re-offer unprompted rule +
no-PR-skills fallback; graphify staleness-counts-as-absence; workspace
plan.md/notes.md rewritten template-conformant; validation-notes.md
sanctioned in the contents table.
