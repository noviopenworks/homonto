# subagent-projection — Comet SDD coordinator checkpoint

Plan: docs/superpowers/plans/2026-07-10-subagent-projection.md
Branch: feature/20260710/subagent-projection | Base(a53950f) plan-base(a53950f)
review_mode: standard | tdd_mode: tdd | build_mode: subagent-driven-development

## Tasks (9)
- Task 1: complete (efd4218, no risk; Minor: strings.Index->Cut nit for final review)
- Task 2: complete (44a9b71, no risk)
- Task 3: complete (7f96e98, risk-reviewed APPROVED; dir-scan D1/D5 reconciliation confirmed safe)
- Task 4: complete (49f7575, no risk; EnabledModelTools already counts subagents, test-locked)
- Task 5: engine materialization + WithSubagentCatalogRoot — STAGE: implementing
- Task 6: adapter subagent projection (both tools) — pending
- Task 7: doctor verification — pending
- Task 8: dogfood — pending
- Task 9: regression + docs — pending

## Minor findings (for final review)
- SP2 [Task3 catalog.go:68-86]: subagents/ dir-scan (new logic) has no dedicated unit test; untested edges: non-.md/subdir under subagents/, dir-scan vs framework decl collision at different path. Add a targeted test.
- SP3 [Task3 expand_test.go:129]: ExpandSubagents test checks single-framework membership only (transitive/dedup proven via shared expandResources). Weak standalone evidence, not a gap.
- SP1 [Task1 frontmatter_test.go:21]: strings.Index can use strings.Cut (Minor, brief-verbatim).
