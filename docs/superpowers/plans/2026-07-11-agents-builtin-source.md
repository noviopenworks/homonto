---
change: agents-builtin-source
design-doc: docs/superpowers/specs/2026-07-11-agents-builtin-source-design.md
base-ref: 83874b75af7e9dfc2051ec03fd61a335a7db8ea0
---

# Plan: builtin agent source resolution (v2 #6a)

Resolve `builtin:<name>` agents from the embedded catalog. See Design Doc
D1/D2/D3/D4. TDD; local: behavior + all prior agent tests must stay green.

## Task 1: `catalog.SubagentContent`
- [ ] 1.1 (TDD RED first) Add `func (c *Catalog) SubagentContent(name) ([]byte,bool,error)` (D1). Tests (fixture FS w/ subagents/x.md): known→content+true; unknown→(nil,false,nil).
- [ ] 1.2 GREEN; gofmt/vet clean. Commit: `feat(catalog): SubagentContent reads a builtin agent's content by name`

## Task 2: `resolveAgentSource` + wire into add/update/doctor
- [ ] 2.1 (TDD RED first) `resolveAgentSource(ag,cfgDir)→([]byte,err)` (D2: local/builtin/remote-err).
- [ ] 2.2 (TDD RED first) Wire into agentsAddCmd, runAgentUpdate, agentsDoctorCmd (D3); `builtin:`+`link`→error (D4); doctor drift via resolver. Tests: add builtin:code-reviewer→installs catalog content+lockfile; unknown builtin→err; builtin+link→err; local unchanged (all prior tests green); doctor builtin healthy.
- [ ] 2.3 GREEN; gofmt/vet clean. Commit: `feat(cli): resolve builtin: agent sources from the embedded catalog`

## Task 3: Regression and docs
- [ ] 3.1 Full regression (build/test/-race/vet/gofmt/mod tidy). E2E: `[agents.cr] source="builtin:code-reviewer"`→add installs catalog content, doctor healthy; builtin+link→error; local still works.
- [ ] 3.2 Update `docs/roadmap.md` v2 + README (builtin: sources). No over-claim (remote deferred).
- [ ] 3.3 Commit all changes.
