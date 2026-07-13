---
change: stateless-adapter-apply
design-doc: docs/superpowers/specs/2026-07-13-stateless-adapter-apply-design.md
base-ref: cc47628cc4e32b548c29d003c12cb432bb340496
archived-with: 2026-07-13-stateless-adapter-apply
---
# Plan
1. Interface: Adapter.Apply(cfg,cs,res,st); claude/opencode expand(cfg) at top of
   Plan+Apply; codex ignores cfg; engine passes e.Cfg. Update ~61 test sites.
   (Behavior-preserving; existing suite is the regression gate.)
2. Verify: go test ./... -race, vet, build, openspec validate --all.
