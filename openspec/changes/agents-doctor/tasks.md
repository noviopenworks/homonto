## 1. `homonto agents doctor` (`internal/cli`)

- [ ] 1.1 (TDD RED first) `agentsDoctorCmd` (`doctor`, NoArgs) per Design Doc D1/D2/D3: load config + `agentlock.Load`; accumulate findings (sorted iteration) — declared-not-installed; local: source drift (source missing / hash != recorded) via `agentlock.HashContent`; per declared target: not-installed / missing-on-disk (Lstat recorded path) / copy modified-on-disk (ReadFile hash != recorded); installed-target-no-longer-declared; orphan (installed not declared). Verdict: 0 findings → `healthy` + nil; else print each + return `fmt.Errorf("homonto agents doctor: %d problem(s) found", n)`. Register `doctor` under `agentsCmd()`.
- [ ] 1.2 (TDD RED first) Tests via `NewRootCmd().SetArgs(["agents","doctor","--config",p])` in a temp workspace (config + homonto/agents + .homonto/agents-lock.json seeded, or built by running `agents add` first): healthy → nil err, stdout `healthy`; declared-not-installed → non-nil naming agent; orphan → non-nil; source drift (edit source after add) → non-nil; modified-on-disk (edit installed copy) → non-nil; missing-on-disk (delete installed file) → non-nil; read-only (no files created). Prefer building state by invoking `agents add` in-test for realism.
- [ ] 1.3 GREEN; gofmt/vet clean. Commit: `feat(cli): 'homonto agents doctor' reports declared-vs-installed drift`

## 2. Regression and docs

- [ ] 2.1 Full regression (build/test/-race/vet/gofmt/mod tidy). E2E (real `homonto`): add a local agent, `agents doctor` → `healthy` exit 0; edit the source file → `agents doctor` reports drift exit non-zero; delete an installed file → reports missing-on-disk.
- [ ] 2.2 Update `docs/roadmap.md` v2 status (agents doctor landed) + README (mention `homonto agents doctor`). No over-claim.
- [ ] 2.3 Commit all changes.
