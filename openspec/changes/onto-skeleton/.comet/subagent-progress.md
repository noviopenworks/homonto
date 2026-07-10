# onto-skeleton — Comet SDD coordinator checkpoint
Plan: docs/superpowers/plans/2026-07-10-onto-skeleton.md
Branch: feature/20260710/onto-skeleton | base 08834df
review_mode: standard | tdd_mode: tdd | build_mode: subagent-driven-development
## Tasks (4)
- Task 1: complete (ee3c472, risk-reviewed APPROVED; self-contained, 15/15 tests)
- Task 2: onto new command — STAGE: implementing
- Task 3: status skeleton reporting — pending
- Task 4: regression + docs — pending
## Minor findings (for final review)
- OF-s1 [Task1 state.go Deps omitempty]: []string{} deps round-trips to nil (inherent YAML/Go nil-vs-empty impedance; not fixable via tags). Binary produces nil deps → holds in practice. Handle nil/empty deps semantics in #3c (dependency resolution).
- OF-s2 [Task1]: Save error-cleanup branches untested; RequiredArtifacts only tested for "open"; temp-file path+".tmp" no-collision-check (single-shot CLI, low). Minor.
