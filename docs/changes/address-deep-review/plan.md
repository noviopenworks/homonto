# Plan: address-deep-review

Design: `design.md` (Status: Confirmed 2026-07-04). One commit per task.

## Task 1 — MCP schema + import (risk: high)

- [ ] done
- Files: internal/adapter/claude/claude.go (+_test), internal/importer/
  importer.go (+_test), internal/adapter/claude/testdata/ (new fixtures)
- Do: design §"Claude MCP schema" + §"Import" — failing tests first
  (schema shape vs real fixture; import args-drop reproduction), then
  fix; conformance fixture from real `claude mcp add` output shape
- Verify: new tests fail pre-fix, pass post-fix; `go test ./...` green

## Task 2 — secret safety

- [ ] done
- Files: internal/adapter/claude/claude.go, internal/adapter/opencode/
  opencode.go, internal/adapter/*/util.go → shared fs helper,
  internal/secret/resolver.go, tests
- Do: design §"Secret safety" — failing tests first (missing-state
  leak; 0600→0644 loosening; double-resolve count), then fix
- Verify: leak test red→green; mode preserved; resolver called once

## Task 3 — pruning (risk: high)

- [ ] done
- Files: internal/adapter/adapter.go, both adapters, internal/plan/
  plan.go, internal/state/state.go, internal/engine/engine.go, tests
- Do: design §"Pruning" + per-adapter state save — failing test first
  (removed MCP orphaned; removed skill link dangling), then implement
  delete end-to-end
- Verify: orphan tests red→green; partial-apply state recorded

## Task 4 — robustness

- [ ] done
- Files: internal/jsonutil/jsonutil.go, internal/config/config.go,
  both adapters, tests
- Do: design §"Injection/traversal robustness" + §"Determinism" —
  failing tests first (dotted-key corruption; traversal; ordering)
- Verify: red→green; two consecutive plans render identically

## Task 5 — hygiene

- [ ] done
- Files: LICENSE (new), .github/workflows/ci.yml (new),
  internal/cli/root.go, README.md
- Do: design §"Hygiene"
- Verify: LICENSE present; CI yaml valid; Version is var; README claims
  match code (grep checks)

## Task 6 — onto v2.1 + specs (risk: high)

- [ ] done
- Files: content/skills/onto/SKILL.md, onto-tweak/SKILL.md,
  onto-close/SKILL.md, docs/guides/onto-workflow.md, docs/adr/0007-*.md,
  docs/changes/address-deep-review/specs/*.md (5 deltas),
  docs/changes/address-deep-review/adr/preflight-warns-not-halts.md
- Do: design §"onto v2.1" + §Key decision 3; write the five delta specs
- Verify: lint rules pass on deltas; table/guide sync greps clean
