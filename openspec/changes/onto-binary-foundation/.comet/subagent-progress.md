# onto-binary-foundation — Comet SDD coordinator checkpoint
Plan: docs/superpowers/plans/2026-07-10-onto-binary-foundation.md
Branch: feature/20260710/onto-binary-foundation | base 06e1420
review_mode: standard | tdd_mode: tdd | build_mode: subagent-driven-development

## Tasks (4)
- Task 1: complete (9df1bb2 + fix 2457d34; risk-reviewed APPROVED, Important Load-names-path fix verified)
- Task 2: complete (bd1445a, no risk; onto version works, homonto untouched)
- Task 3: complete (8832e89 + fix aa0efc8; risk-reviewed APPROVED, dead archive-skip removed + branch coverage, read-only verified)
- Task 4: regression + docs — STAGE: implementing

## Minor findings (for final review)
- OF3 [Task3 status.go]: --dir unvalidated (no ../symlink checks). Accepted for a local read-only diagnostic; harden only if fed untrusted input.
- OF4 [Task3 status.go]: success line uses State.Change, invalid line uses dir name — inconsistent if they diverge (brief-specified). Low impact.
- OF5 [Task3 status_test.go treeSnapshot]: snapshots files only, not directories; zero-writes proof is 'zero file writes' not 'zero fs mutations'. Low.
- OF1 [Task1 state.go Parse recover()]: belt-and-suspenders recover masks future non-yaml panics as parse errors. Accept (explicit no-panic req).
- OF2 [Task1]: coordinator checkpoint file appeared in Task1 diff (harmless bookkeeping).
