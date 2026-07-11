---
change: agents-doctor
design-doc: docs/superpowers/specs/2026-07-11-agents-doctor-design.md
base-ref: ed2ecd501c13bf977bfdc68b2f5e40c80e7e1b5c
archived-with: 2026-07-11-agents-doctor
---

# Plan: agents doctor (v2 #3)

Read-only `homonto agents doctor`: declared (config) vs installed (agentlock
lockfile) vs disk drift report. See Design Doc for exact checks. TDD.

## Task 1: `homonto agents doctor` (`internal/cli`)

- [x] 1.1 (TDD RED first) `agentsDoctorCmd` per Design Doc D1/D2/D3: config.Load + agentlock.Load; sorted findings — declared-not-installed; local: source drift (HashContent(homonto/agents/<x>.md) != recorded / missing); per declared target not-installed / missing-on-disk (Lstat) / copy modified-on-disk (ReadFile hash != recorded); installed-target-no-longer-declared; orphan. 0→`healthy`+nil; else print+`fmt.Errorf(...N problem(s)...)`. Register `doctor` under `agentsCmd()`.
- [x] 1.2 (TDD RED first) Tests via `NewRootCmd().SetArgs(["agents","doctor","--config",p])`: healthy (add then doctor)→nil,`healthy`; declared-not-installed→non-nil; orphan→non-nil; source drift (edit source after add)→non-nil; modified-on-disk (edit installed copy)→non-nil; missing-on-disk (delete file)→non-nil; read-only. Build state by running `agents add` in-test.
- [x] 1.3 GREEN; gofmt/vet clean. Commit: `feat(cli): 'homonto agents doctor' reports declared-vs-installed drift`

## Task 2: Regression and docs

- [x] 2.1 Full regression (build/test/-race/vet/gofmt/mod tidy). E2E (real `homonto`): add local agent→doctor `healthy` exit 0; edit source→doctor drift non-zero; delete installed file→missing-on-disk.
- [x] 2.2 Update `docs/roadmap.md` v2 status + README (mention `homonto agents doctor`). No over-claim.
- [x] 2.3 Commit all changes.
