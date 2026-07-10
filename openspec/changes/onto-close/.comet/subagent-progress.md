# onto-close — Comet SDD coordinator checkpoint
Plan: docs/superpowers/plans/2026-07-10-onto-close.md
Branch: feature/20260710/onto-close | base a3137d2
review_mode: standard | tdd_mode: tdd | build_mode: subagent-driven-development
## Tasks (3)
- Task 1: complete (c50ac93; coordinator-verified pure/read-only/prefix-safe, 38 tests; self-contained. Risk=new pure helper, minimal.)
- Task 2: complete (6cc686f + hardening fix 1c68c19; risk-reviewed APPROVED opus. no-move/no-flip invariant HOLDS; refusal tests now assert archived==false. E2E full workflow→close archives correctly.)
- Task 3: regression + docs — STAGE: implementing
## Minor findings (for final review)
- OF-c1 [Task2 close.go archived-before-move]: Save flips archived:true before Rename; crash/TOCTOU window leaves change in-place flagged archived (self-healing on retry; spec-specified). Minor.
- OF-c2 [Task2 close.go:94 Stat]: no-clobber treats non-IsNotExist Stat error as proceed (negligible; Rename would fail anyway).
