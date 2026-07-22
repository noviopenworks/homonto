# Subagent Dispatch Progress — remove-model-tiers

- review_mode: standard (risk tasks get a per-task reviewer; 1 final lightweight review)
- tdd_mode: direct (no enforced TDD)
- branch: feature/20260721/remove-model-tiers
- base-ref: f77dbf105a29aec9cfa8984637afbb14b9a9c82a

## Coupling decision

The refactor is compile-coupled (agentfm + config + adapters change together).
Per the subagent-driven-development skill's own routing, tightly-coupled work is
one logical task. Dispatching the coupled core as a single implementer task (not
a bundle of independent tasks). The Design Doc mandates land-as-one, no shim.

## Current task: coupled core (the whole implementation)

- Plan task text: "implement remove-model-tiers per Design Doc D1–D5 + tests"
- OpenSpec tasks: tasks.md groups 1–8 (checked off together after review)
- Stage: implementing
- Implementer: general subagent (single dispatch, coupled unit)
- Risk signals (self-anticipated): cross-module, data/config migration, public API
  contract change (breaking) → this IS a risk task → per-task reviewer will fire
- Definition of done: go build ./... && go vet ./... && go test ./internal/... &&
  homonto doctor all green; no residual TierNames/role:/[models. in code or catalog
- Report contract: status DONE|DONE_WITH_CONCERNS|BLOCKED|NEEDS_CONTEXT, commits,
  changed files, test summary, risk-signal self-report, concerns
