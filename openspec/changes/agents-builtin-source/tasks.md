## 1. `catalog.SubagentContent` (`internal/catalog`)

- [ ] 1.1 (TDD RED first) Add `func (c *Catalog) SubagentContent(name string) ([]byte, bool, error)` per D1 (reads `c.fsys` at `c.subagents[name]`; unknown → (nil,false,nil)). Tests (fixture FS with a `subagents/x.md`): known name → content + true; unknown → (nil,false,nil).
- [ ] 1.2 GREEN; gofmt/vet clean. Commit: `feat(catalog): SubagentContent reads a builtin agent's content by name`

## 2. `resolveAgentSource` + wire into add/update/doctor (`internal/cli`)

- [ ] 2.1 (TDD RED first) Add `resolveAgentSource(ag config.Agent, cfgDir string) ([]byte, error)` per D2 (local → homonto/agents/<x>.md; builtin → catalog.SubagentContent; else "not yet supported").
- [ ] 2.2 (TDD RED first) Wire into `agentsAddCmd`, `runAgentUpdate`, `agentsDoctorCmd` (D3): replace the local-only reads + "not yet supported" branch with the resolver; add the `builtin: + link` → error guard (D4). doctor source-drift uses the resolver (unresolved → finding). Tests: add a builtin agent (fixture/bundled) → installs catalog content + lockfile; unknown builtin → error; builtin+link → error; local: unchanged (all prior add/update/doctor tests green); doctor on a builtin agent whose catalog content matches → healthy.
- [ ] 2.3 GREEN; gofmt/vet clean. Commit: `feat(cli): resolve builtin: agent sources from the embedded catalog`

## 3. Regression and docs

- [ ] 3.1 Full regression (build/test/-race/vet/gofmt/mod tidy). E2E (real `homonto`): declare `[agents.<x>] source="builtin:<real-bundled-agent>"` → `agents add` installs the catalog content; `agents doctor` → healthy; a `builtin:x mode="link"` → clear error. local: agents still work end-to-end.
- [ ] 3.2 Update `docs/roadmap.md` v2 status (builtin: sources resolved; remote still deferred as a v1 non-goal) + README (agents can source `builtin:`). No over-claim.
- [ ] 3.3 Commit all changes.
