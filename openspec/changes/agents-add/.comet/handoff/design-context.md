# Comet Design Handoff

- Change: agents-add
- Phase: design
- Mode: compact
- Context hash: 4d9af23c188c4a6ae3662eadbbaa4a50af613913c715a1ce9ec4d7f7d61101cd

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/agents-add/proposal.md

- Source: openspec/changes/agents-add/proposal.md
- Lines: 1-58
- SHA256: 796b204e64ec7498b9bbdff2c297abc5f5ad1f8238e09eab28889006231b3b78

```md
## Why

The v2 foundation added the `[agents.<name>]` model and read-only `homonto
agents list`. The next step in the agent lifecycle is *installing* a declared
agent: materializing it into the target tools and recording what was installed so
later increments (update/pin/doctor/migrate) have ground truth. This change adds
`homonto agents add` plus the agent lockfile — the first agent-lifecycle
mutation, and the ground truth the rest of v2 builds on. Scope is kept
self-contained: `local:` sources only (builtin/remote deferred), `copy` and
`link` modes, conflict-safe and idempotent.

## What Changes

- Add an agent lockfile at `.homonto/agents-lock.json` (a new `internal/agentlock`
  package: typed model + `Load`/`Save`, empty on absence). Per installed agent it
  records `source`, `version`, `mode`, `targets`, and per-target
  `{path, hash}` (sha256 of the installed content). The lockfile is separate from
  `state.json` (agent lifecycle needs its own installed-version ground truth).
- Add `homonto agents add <name>`: installs a declared agent.
  - Loads the config, finds `[agents.<name>]` (error if undeclared).
  - Supports `source = "local:<x>"` → resolves `homonto/agents/<x>.md` relative to
    the config dir (error if missing). `builtin:`/remote sources return a clear
    "not yet supported" error (deferred).
  - For each target in the agent's `TargetsOrAll()`: the destination is
    `<agent dir for tool>/<name>.md` (via `subagentpath.Dir`, user scope). `copy`
    mode writes the file content; `link` mode symlinks the source.
  - **Conflict-safe**: if a destination already exists and is NOT a homonto-managed
    install of this agent (absent from the lockfile), it REFUSES and installs
    nothing for that agent (all-or-nothing).
  - **Idempotent**: a target already installed with matching content/hash is a
    no-op; re-running reports "already up to date".
  - Updates the lockfile and prints what was installed/updated per target.
- This is additive; `agents list`, the `[subagents]` projection, and `plan`/
  `apply` are unchanged.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `agent-lifecycle`: gains `homonto agents add` (install a `local:` agent per its
  mode into target tools, conflict-safe and idempotent) and the
  `.homonto/agents-lock.json` lockfile recording installed agents.

## Impact

- New `internal/agentlock` package (lockfile model + Load/Save + hashing helper).
- `internal/cli/agents.go`: new `add` subcommand (`agentsAddCmd`).
- Reuses `internal/subagentpath.Dir`, `internal/fsutil.WriteAtomic`,
  `internal/link.Link`.
- Tests in `internal/agentlock` and `internal/cli`.
- No new dependency. Read-only `agents list` and all v1 behavior unchanged.
- Deferred to later increments: `builtin:`/remote sources; `update`/`pin`/
  `doctor`/`migrate`; three-way-merge/backup; a per-agent scope field;
  `[agents]`-vs-`[subagents]` reconciliation.

```

## openspec/changes/agents-add/design.md

- Source: openspec/changes/agents-add/design.md
- Lines: 1-108
- SHA256: d827b29c7d9b7ed0df5292fbcc6e585911af37a2251249f5f6fc83de7522f03a

[TRUNCATED]

```md
## Context

v2 #2. After the read-only foundation (`[agents.<name>]` + `agents list`), this
installs a declared agent and records it in a lockfile — the ground truth for
later update/pin/doctor/migrate. Scoped self-contained: `local:` sources, copy &
link modes, conflict-safe, idempotent. Reuses `subagentpath.Dir` (install dir),
`fsutil.WriteAtomic` (copy), `link.Link` (symlink).

## Goals / Non-Goals

**Goals**: `.homonto/agents-lock.json` (new `internal/agentlock` pkg) + `homonto
agents add <name>` for local sources (copy/link), conflict-safe + idempotent +
recorded.

**Non-Goals**: builtin/remote sources (clear "deferred" error); update/pin/doctor/
migrate; three-way-merge/backup; per-agent scope (user scope only); touching
`plan`/`apply`/`state.json`/`[subagents]`.

## Decisions

### D1 — Lockfile (`internal/agentlock`)

```go
package agentlock
type Install struct { Path string `json:"path"`; Hash string `json:"hash"` }
type Agent struct {
    Source  string             `json:"source"`
    Version string             `json:"version,omitempty"`
    Mode    string             `json:"mode"`
    Targets []string           `json:"targets"`
    Installed map[string]Install `json:"installed"` // tool -> install
}
type Lock struct { Agents map[string]Agent `json:"agents"` }
func Load(homontoDir string) (*Lock, error) // reads <dir>/agents-lock.json, empty if absent
func (l *Lock) Save(homontoDir string) error // atomic write
func HashContent(b []byte) string // sha256 hex (reuse secret.Hash or crypto/sha256)
```
`homontoDir` = `.homonto` next to the config (same anchor the engine uses:
`filepath.Join(filepath.Dir(configPath), ".homonto")`). Deterministic JSON
(sorted keys via encoding/json on maps is sorted) so re-saves are stable.

### D2 — `homonto agents add <name>` (`internal/cli/agents.go`)

```
cfgPath := --config; cfgDir := filepath.Dir(cfgPath); homontoDir := cfgDir/.homonto
c := config.Load(cfgPath)
ag, ok := c.Agents[name]; if !ok -> error "agent %q is not declared"
if !strings.HasPrefix(ag.Source, "local:") -> error "agents add: only local: sources are supported yet (got %q)"
srcName := trimprefix(ag.Source, "local:")
srcPath := filepath.Join(cfgDir, "homonto", "agents", srcName+".md")
content, err := os.ReadFile(srcPath); if err -> error naming srcPath
hash := agentlock.HashContent(content)
lock := agentlock.Load(homontoDir)
home := os.UserHomeDir()
for _, tool := range ag.TargetsOrAll():
    dir := subagentpath.Dir(tool, "user", home, "")   // projectRoot "" = user scope
    dst := filepath.Join(dir, name+".md")
    prev, wasManaged := lock.Agents[name].Installed[tool]  // managed iff recorded with this dst
    if fileExists(dst):
        if !wasManaged || prev.Path != dst -> CONFLICT: refuse (collect, install nothing for this agent)
        if mode==copy && prev.Hash==hash -> noop (already up to date)
        if mode==link && isSymlinkTo(dst, srcPath) -> noop
    // install
    mkdirall(dir)
    if mode==copy: fsutil.WriteAtomic(dst, content)
    if mode==link: link.Link(srcPath, dst, homontoDir?)   // managed root
    record Installed[tool] = {Path:dst, Hash:hash}
// all-or-nothing per agent: do the conflict scan FIRST across all targets; if any conflict, refuse before writing
lock.Agents[name] = {Source,Version,Mode:ModeOrDefault,Targets:TargetsOrAll,Installed}
lock.Save(homontoDir)
print per-target: "installed"/"updated"/"up to date"
```

Two-pass per agent: (1) scan all targets for an unmanaged-file conflict → if any,
refuse and write nothing; (2) install + record. This keeps "installs nothing for
that agent" on conflict.

### D3 — Managed vs unmanaged

A destination is "managed by us" iff the lockfile records this agent with that

```

Full source: openspec/changes/agents-add/design.md

## openspec/changes/agents-add/tasks.md

- Source: openspec/changes/agents-add/tasks.md
- Lines: 1-16
- SHA256: f7d76abae1f83a24e0033147e606345b39ade79536b4f9bc21bf33286b94c0e6

```md
## 1. Agent lockfile (`internal/agentlock`)

- [ ] 1.1 (TDD RED first) New pkg `internal/agentlock`: `Install{Path,Hash}`, `Agent{Source,Version,Mode,Targets,Installed map[string]Install}`, `Lock{Agents map[string]Agent}`; `Load(homontoDir)` (empty on absence), `(*Lock).Save(homontoDir)` (atomic, deterministic JSON), `HashContent([]byte) string` (sha256 hex). Tests: Load-absent→empty; Save then Load round-trips; Save is deterministic (two saves byte-identical); HashContent stable.
- [ ] 1.2 GREEN; gofmt/vet clean. Commit: `feat(agentlock): .homonto/agents-lock.json model + Load/Save`

## 2. `homonto agents add` (`internal/cli`)

- [ ] 2.1 (TDD RED first) `agentsAddCmd` (`add <name>`, ExactArgs(1)) per Design Doc D2/D3: load config + find agent (undeclared→error); non-local source→"not yet supported"; resolve `homonto/agents/<x>.md` (missing→error naming path); two-pass per agent — conflict-scan all targets (unmanaged existing file→refuse, install nothing), then install (copy=WriteAtomic content / link=link.Link) into `subagentpath.Dir(tool,"user",home,"")/<name>.md`, recording `Installed[tool]={path,hash}`; idempotent (copy hash-match / link target-match → no-op); Save lockfile; print per-target status. Register `add` under `agentsCmd()`.
- [ ] 2.2 (TDD RED first) Tests via `NewRootCmd().SetArgs(["agents","add",name,"--config",p])` in a temp workspace (config dir + homonto/agents/<x>.md): copy-mode add → file in each target's agent dir + lockfile records path+hash; re-add unchanged → no-op (files untouched, no rewrite); conflict (pre-existing unmanaged dst) → refused, nothing installed for that agent, lockfile unchanged; builtin source → "not yet supported" error; undeclared name → error; missing local source file → error naming path; link-mode add → symlink created + recorded.
- [ ] 2.3 GREEN; gofmt/vet clean. Commit: `feat(cli): 'homonto agents add' installs a local agent (conflict-safe, idempotent)`

## 3. Regression and docs

- [ ] 3.1 Full regression: `go build ./...`, `go test ./... -count=1`, `go test -race ./internal/...`, `go vet ./...`, `gofmt -l .` (empty), `go mod tidy` clean. E2E (real `homonto` binary): a workspace with `[agents.rev] source="local:rev" mode="copy"` + `homonto/agents/rev.md` → `homonto agents add rev` installs into the tool agent dirs + writes `.homonto/agents-lock.json`; re-run → no-op; a builtin agent → "not yet supported".
- [ ] 3.2 Update `docs/roadmap.md` v2 status (agents add + lockfile landed; update/pin/doctor/migrate + builtin/remote still deferred) + README (mention `homonto agents add`). No over-claim.
- [ ] 3.3 Commit all changes.

```

## openspec/changes/agents-add/specs/agent-lifecycle/spec.md

- Source: openspec/changes/agents-add/specs/agent-lifecycle/spec.md
- Lines: 1-53
- SHA256: cd7cfe44962ea68ec46b230ffdd2a6df17fea90cd8abada96f3bb6354bc0c395

```md
## ADDED Requirements

### Requirement: homonto agents add installs a declared agent

`homonto agents add <name>` SHALL install a declared `[agents.<name>]` agent into
its target tools and record the installation in an agent lockfile at
`.homonto/agents-lock.json`. This increment supports `local:<x>` sources only;
`builtin:` and remote sources SHALL return a clear "not yet supported" error.

For a `local:<x>` agent the command SHALL resolve `homonto/agents/<x>.md`
(relative to the config directory), and for each target in the agent's targets
install it into that tool's agent directory as `<name>.md`: `copy` mode writes the
content, `link` mode symlinks the source. The command SHALL be:

- **conflict-safe**: if a destination already exists and is not a homonto-managed
  install of this agent (not recorded in the lockfile), it SHALL refuse and
  install nothing for that agent;
- **idempotent**: a target already installed with matching content SHALL be a
  no-op;
- **recorded**: on success the lockfile SHALL record the agent's source, version,
  mode, targets, and each target's installed path and content hash.

An undeclared agent name SHALL be an error. A missing local source file SHALL be
an error naming the expected path.

#### Scenario: Add a local copy-mode agent

- **GIVEN** `[agents.rev]` with `source = "local:rev"` and `mode = "copy"`, a `homonto/agents/rev.md`, and both tools targeted
- **WHEN** `homonto agents add rev` runs
- **THEN** `rev.md` is written into each tool's agent directory, the lockfile records the agent with each target's path and content hash, and the command reports the installs

#### Scenario: Add is idempotent

- **GIVEN** an already-installed agent with unchanged content
- **WHEN** `homonto agents add <name>` runs again
- **THEN** each target is a no-op and nothing is rewritten

#### Scenario: Add refuses to clobber an unmanaged file

- **GIVEN** a destination `<name>.md` that already exists and is not recorded in the lockfile
- **WHEN** `homonto agents add <name>` runs
- **THEN** it refuses naming the conflict and installs nothing for that agent

#### Scenario: builtin source is not yet supported

- **GIVEN** `[agents.x]` with `source = "builtin:x"`
- **WHEN** `homonto agents add x` runs
- **THEN** it returns a clear error that builtin sources are not yet supported

#### Scenario: undeclared agent is an error

- **WHEN** `homonto agents add nope` runs against a config with no `[agents.nope]`
- **THEN** it errors that the agent is not declared

```
