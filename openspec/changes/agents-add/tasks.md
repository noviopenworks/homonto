## 1. Agent lockfile (`internal/agentlock`)

- [x] 1.1 (TDD RED first) New pkg `internal/agentlock`: `Install{Path,Hash}`, `Agent{Source,Version,Mode,Targets,Installed map[string]Install}`, `Lock{Agents map[string]Agent}`; `Load(homontoDir)` (empty on absence), `(*Lock).Save(homontoDir)` (atomic, deterministic JSON), `HashContent([]byte) string` (sha256 hex). Tests: Load-absent→empty; Save then Load round-trips; Save is deterministic (two saves byte-identical); HashContent stable.
- [x] 1.2 GREEN; gofmt/vet clean. Commit: `feat(agentlock): .homonto/agents-lock.json model + Load/Save`

## 2. `homonto agents add` (`internal/cli`)

- [x] 2.1 (TDD RED first) `agentsAddCmd` (`add <name>`, ExactArgs(1)) per Design Doc D2/D3: load config + find agent (undeclared→error); non-local source→"not yet supported"; resolve `homonto/agents/<x>.md` (missing→error naming path); two-pass per agent — conflict-scan all targets (unmanaged existing file→refuse, install nothing), then install (copy=WriteAtomic content / link=link.Link) into `subagentpath.Dir(tool,"user",home,"")/<name>.md`, recording `Installed[tool]={path,hash}`; idempotent (copy hash-match / link target-match → no-op); Save lockfile; print per-target status. Register `add` under `agentsCmd()`.
- [x] 2.2 (TDD RED first) Tests via `NewRootCmd().SetArgs(["agents","add",name,"--config",p])` in a temp workspace (config dir + homonto/agents/<x>.md): copy-mode add → file in each target's agent dir + lockfile records path+hash; re-add unchanged → no-op (files untouched, no rewrite); conflict (pre-existing unmanaged dst) → refused, nothing installed for that agent, lockfile unchanged; builtin source → "not yet supported" error; undeclared name → error; missing local source file → error naming path; link-mode add → symlink created + recorded.
- [x] 2.3 GREEN; gofmt/vet clean. Commit: `feat(cli): 'homonto agents add' installs a local agent (conflict-safe, idempotent)`

## 3. Regression and docs

- [x] 3.1 Full regression: `go build ./...`, `go test ./... -count=1`, `go test -race ./internal/...`, `go vet ./...`, `gofmt -l .` (empty), `go mod tidy` clean. E2E (real `homonto` binary): a workspace with `[agents.rev] source="local:rev" mode="copy"` + `homonto/agents/rev.md` → `homonto agents add rev` installs into the tool agent dirs + writes `.homonto/agents-lock.json`; re-run → no-op; a builtin agent → "not yet supported".
- [x] 3.2 Update `docs/roadmap.md` v2 status (agents add + lockfile landed; update/pin/doctor/migrate + builtin/remote still deferred) + README (mention `homonto agents add`). No over-claim.
- [x] 3.3 Commit all changes.
