# onto-init — Comet SDD coordinator checkpoint
Plan: docs/superpowers/plans/2026-07-10-onto-init.md
Branch: feature/20260710/onto-init | base 5bfd362
review_mode: standard | tdd_mode: tdd | build_mode: subagent-driven-development
## Tasks (3)
- Task 1: complete (5fd379d, no risk; isolation clean, 11/11 ontocli tests)
- Task 2: onto init command + idempotent scaffold — STAGE: implementing
- Task 3: regression + docs — pending
## Minor findings (for final review)
- OF-i1 [Task1 init.go]: malformed-TOML parse-error path wrapped but not unit-tested (low risk, go-toml passthrough).
