---
change: agents-add
design-doc: docs/superpowers/specs/2026-07-11-agents-add-design.md
base-ref: dd80dcf6c2ae820d246459601ae0f78bc75a5d23
---

# Plan: agents add + lockfile (v2 #2)

First agent-lifecycle mutation: `homonto agents add` (local sources, copy/link,
conflict-safe, idempotent) + `.homonto/agents-lock.json`. See Design Doc for
exact model + two-pass install logic. TDD.

## Task 1: Agent lockfile (`internal/agentlock`)

- [ ] 1.1 (TDD RED first) New pkg: `Install{Path,Hash}`, `Agent{Source,Version,Mode,Targets,Installed map[string]Install}`, `Lock{Agents map[string]Agent}`; `Load(homontoDir)` (empty on absence), `(*Lock).Save(homontoDir)` (atomic via fsutil.WriteAtomic, deterministic JSON), `HashContent([]byte) string` (sha256 hex). Tests: Load-absent→empty; Save→Load round-trip; two Saves byte-identical; HashContent stable.
- [ ] 1.2 GREEN; gofmt/vet clean. Commit: `feat(agentlock): .homonto/agents-lock.json model + Load/Save`

## Task 2: `homonto agents add` (`internal/cli`)

- [ ] 2.1 (TDD RED first) `agentsAddCmd` per Design Doc D2/D3: find agent (undeclared→err); non-local→"not yet supported"; resolve `homonto/agents/<x>.md` (missing→err naming path); TWO-PASS: conflict-scan all targets (unmanaged existing dst→refuse, install nothing), then install copy=fsutil.WriteAtomic / link=link.Link into `subagentpath.Dir(tool,"user",home,"")/<name>.md`, record Installed{path,hash}; idempotent (copy hash-match / link target-match→no-op); Save lock; print per-target status. Register `add` under `agentsCmd()`.
- [ ] 2.2 (TDD RED first) Tests: copy add → files in each target dir + lockfile path+hash; re-add unchanged → no-op (files untouched); conflict (pre-existing unmanaged dst) → refused, nothing installed, lock unchanged; builtin→"not yet supported"; undeclared→err; missing source file→err naming path; link-mode → symlink + recorded.
- [ ] 2.3 GREEN; gofmt/vet clean. Commit: `feat(cli): 'homonto agents add' installs a local agent (conflict-safe, idempotent)`

## Task 3: Regression and docs

- [ ] 3.1 Full regression (build/test/-race/vet/gofmt/mod tidy). E2E (real `homonto`): `[agents.rev] source="local:rev" mode="copy"` + `homonto/agents/rev.md` → `agents add rev` installs + writes lockfile; re-run no-op; builtin→"not yet supported".
- [ ] 3.2 Update `docs/roadmap.md` v2 status + README (mention `homonto agents add`). No over-claim.
- [ ] 3.3 Commit all changes.
