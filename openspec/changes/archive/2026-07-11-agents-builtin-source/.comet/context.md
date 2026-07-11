# Comet Design Handoff

- Change: agents-builtin-source
- Phase: design
- Mode: compact
- Context hash: 3b02a24ff3e9e7647b7677e5801eb6280f510c04c9109e92745f807803a8ee85

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/agents-builtin-source/proposal.md

- Source: openspec/changes/agents-builtin-source/proposal.md
- Lines: 1-53
- SHA256: a3e9900b68d0e425ccfe3fb5ce5cac0b83983b67a1d16610eca13511f94ec9dd

```md
## Why

The v2 agent lifecycle (`agents add / list / doctor / update [--all]` with
three-way merge) works only for `local:` sources — `builtin:` returns "not yet
supported". This change resolves `builtin:<name>` agents from the embedded
catalog (the same curated agent files the framework ships, indexed as subagents),
so a user can declare and manage a bundled agent without authoring it under
`homonto/agents/`. Remote sources remain deferred (an explicit first-release
non-goal).

## What Changes

- Add `catalog.Catalog.SubagentContent(name string) ([]byte, bool, error)`: reads
  a builtin agent's content from the embedded catalog by name (the curated agent
  files are the framework's subagents), returning `ok=false` for an unknown name.
- Add a source resolver in `internal/cli` shared by `add`/`update`/`doctor`:
  `resolveAgentSource(ag, cfgDir) ([]byte, error)` →
  - `local:<x>` → `homonto/agents/<x>.md` (as today);
  - `builtin:<x>` → the embedded catalog content (unknown → clear error);
  - anything else → "unsupported source (remote not yet supported)".
- `agents add` and `agents update` resolve the source via that resolver, so both
  now accept `builtin:` agents. All downstream logic (hashing, materialize,
  base-blob store, three-way merge, `.merged` conflict sidecar) is source-agnostic
  and works unchanged — including auto-merging a user's local edits with a
  *catalog upgrade* to a builtin agent.
- `agents doctor` resolves the source via the resolver too, so a `builtin:` agent
  gets the same "source changed since install" drift detection (a catalog upgrade
  that changes the builtin content), and an unknown/unresolvable source is a
  finding.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `agent-lifecycle`: `agents add`/`update`/`doctor` resolve `builtin:<name>`
  sources from the embedded catalog (in addition to `local:`), so bundled agents
  are declarable and lifecycle-managed (install, drift, three-way merge). Remote
  sources remain unsupported (deferred).

## Impact

- `internal/catalog/catalog.go`: new `SubagentContent` method.
- `internal/cli/agents.go`: `resolveAgentSource` helper; `add`/`update`/`doctor`
  use it (replacing the `local:`-only reads and the "not yet supported" branch).
- Tests in `internal/catalog` and `internal/cli` (a fixture catalog with a builtin
  agent).
- No new dependency. `local:` behavior unchanged; remote still rejected.
- Deferred: remote sources (first-release non-goal); catalog-version-aware pinning
  for builtin agents; `[agents]`-vs-`[subagents]` reconciliation.

```

## openspec/changes/agents-builtin-source/design.md

- Source: openspec/changes/agents-builtin-source/design.md
- Lines: 1-98
- SHA256: 79e25e4bb0a4e635903a7c4cf6347df5433fa3c322d3488bdb33a1756bd2ee52

[TRUNCATED]

```md
## Context

v2 #6a — resolve `builtin:<name>` agents from the embedded catalog so add/update/
doctor work with bundled agents, not just `local:`. Remote deferred (v1 non-goal).
A builtin agent IS a curated catalog agent file (the framework's subagents index).

## Goals / Non-Goals

**Goals**: `catalog.SubagentContent`; a shared `resolveAgentSource` (local+builtin,
reject remote) used by add/update/doctor; the whole existing lifecycle (install/
merge/blob/sidecar) works for builtin unchanged.

**Non-Goals**: remote sources; catalog-version-aware version pinning for builtin;
link mode for builtin (link needs a local file path — builtin has no on-disk
source path, so builtin agents are copy-only this increment: a `builtin:` + `link`
declaration is an error); `[agents]`-vs-`[subagents]` reconciliation.

## Decisions

### D1 — `catalog.Catalog.SubagentContent(name) ([]byte, bool, error)`

```go
func (c *Catalog) SubagentContent(name string) ([]byte, bool, error) {
    p, ok := c.subagents[name]
    if !ok { return nil, false, nil }
    b, err := fs.ReadFile(c.fsys, p)
    return b, true, err
}
```
Mirrors `SubagentPath`; reads via the private `fsys` (embedded FS in production).

### D2 — `resolveAgentSource(ag config.Agent, cfgDir string) (content []byte, err error)` (internal/cli/agents.go)

```go
switch {
case strings.HasPrefix(ag.Source, "local:"):
    p := filepath.Join(cfgDir, "homonto", "agents", strings.TrimPrefix(ag.Source,"local:")+".md")
    b, err := os.ReadFile(p)
    if err != nil { return nil, fmt.Errorf("source file %s: %w", p, err) }
    return b, nil
case strings.HasPrefix(ag.Source, "builtin:"):
    name := strings.TrimPrefix(ag.Source, "builtin:")
    cat, err := catalog.New()
    if err != nil { return nil, err }
    b, ok, err := cat.SubagentContent(name)
    if err != nil { return nil, err }
    if !ok { return nil, fmt.Errorf("unknown builtin agent %q", name) }
    return b, nil
default:
    return nil, fmt.Errorf("unsupported agent source %q (remote sources are not yet supported)", ag.Source)
}
```

### D3 — Wire into add/update/doctor

- **add** (`agentsAddCmd`): replace the `!HasPrefix(local:)→"not yet supported"`
  check + local `os.ReadFile(srcPath)` with `content, err := resolveAgentSource(ag,
  cfgDir)`. Everything after (hash, conflict-scan, materialize copy/link, blob Put)
  is unchanged. **Link + builtin guard**: builtin has no local source path to
  symlink, so if `mode==link && builtin:` → error "link mode requires a local:
  source". (Or fall back to copy — but explicit error is clearer.) local+link
  unchanged (symlinks the local file).
- **update** (`runAgentUpdate`): same replacement — `content := resolveAgentSource`.
  The merge path is source-agnostic. Keep the link-mode branch requiring a local
  source path (builtin+link errors).
- **doctor** (`agentsDoctorCmd`): the source-drift check currently only handles
  `local:`. Replace with: `srcContent, rerr := resolveAgentSource(ag, cfgDir)`; if
  `rerr != nil` → finding "source unresolved: <err>"; else if `HashContent(srcContent)
  != <recorded base hash>` → "source changed since install". This gives builtin
  agents drift detection (catalog upgrade) uniformly.

### D4 — Link mode + builtin

`link` mode symlinks a local file. A `builtin:` source has no stable on-disk path
(it lives in the embedded FS), so `builtin:` + `link` is rejected at install with a
clear error. `builtin:` agents are effectively copy-mode. (Materializing the
builtin to a stable path then linking is a possible future refinement; not now.)

## Risks / Trade-offs


```

Full source: openspec/changes/agents-builtin-source/design.md

## openspec/changes/agents-builtin-source/tasks.md

- Source: openspec/changes/agents-builtin-source/tasks.md
- Lines: 1-16
- SHA256: 03a536fc7e702d84b6bf7dd006025da3071c6c8dd778d1e069d0d88eff708dfa

```md
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

```

## openspec/changes/agents-builtin-source/specs/agent-lifecycle/spec.md

- Source: openspec/changes/agents-builtin-source/specs/agent-lifecycle/spec.md
- Lines: 1-71
- SHA256: 10e0e30eea928e656ed44d770ab959384c16412a86cf4572c06cb8b644bbe21d

```md
## MODIFIED Requirements

### Requirement: homonto agents add installs a declared agent

`homonto agents add <name>` SHALL install a declared `[agents.<name>]` agent into
its target tools and record the installation in `.homonto/agents-lock.json`. The
agent's source SHALL be resolved as follows:

- `local:<x>` → `homonto/agents/<x>.md` (relative to the config directory);
- `builtin:<x>` → the embedded catalog's curated agent content by name (an
  unknown builtin name is an error);
- any other scheme (e.g. remote) → a clear "not yet supported" error.

For each target in the agent's targets it installs the resolved content into that
tool's agent directory as `<name>.md` (`copy` writes the content, `link` symlinks
a local source). The command SHALL be conflict-safe (refuse to clobber an
unmanaged file, all-or-nothing per agent), idempotent, and record each target's
path and content hash plus persist the base content to the blob store. An
undeclared agent name, or an unresolvable source, SHALL be an error.

#### Scenario: Add a builtin agent

- **GIVEN** a `[agents.rev]` with `source = "builtin:<name>"` where `<name>` is a curated catalog agent
- **WHEN** `homonto agents add rev` runs
- **THEN** the catalog content is installed into each target and recorded in the lockfile

#### Scenario: Add an unknown builtin agent is an error

- **GIVEN** a `[agents.x]` with `source = "builtin:not-a-real-agent"`
- **WHEN** `homonto agents add x` runs
- **THEN** it errors that the builtin agent is unknown

#### Scenario: Add a local copy-mode agent

- **GIVEN** `[agents.rev]` with `source = "local:rev"` and `mode = "copy"`, a `homonto/agents/rev.md`, and both tools targeted
- **WHEN** `homonto agents add rev` runs
- **THEN** `rev.md` is written into each tool's agent directory, the lockfile records the agent with each target's path and content hash, and the command reports the installs

#### Scenario: Add refuses to clobber an unmanaged file

- **GIVEN** a destination `<name>.md` that already exists and is not recorded in the lockfile
- **WHEN** `homonto agents add <name>` runs
- **THEN** it refuses naming the conflict and installs nothing for that agent

#### Scenario: undeclared agent is an error

- **WHEN** `homonto agents add nope` runs against a config with no `[agents.nope]`
- **THEN** it errors that the agent is not declared

### Requirement: homonto agents update re-materializes an installed agent

`homonto agents update <name>` (and `--all`) SHALL reconcile an already-installed
declared agent with its current source, resolving the source the same way as
`agents add` (`local:` from `homonto/agents/`, `builtin:` from the embedded
catalog, other schemes unsupported). The three-way merge, `.merged` conflict
sidecar, base-blob advance, backup fallback, and idempotency SHALL apply to
`builtin:` agents exactly as to `local:` — including auto-merging a user's local
edits with a catalog upgrade to a builtin agent. An undeclared, not-yet-installed,
or unresolvable-source agent SHALL be an error.

#### Scenario: update merges a catalog upgrade into a builtin agent's local edits

- **GIVEN** an installed `builtin:` copy agent with a local edit, and a newer catalog whose content for that agent changed disjointly
- **WHEN** `homonto agents update <name>` runs
- **THEN** the local edit and the catalog change are three-way-merged (or a `<dst>.merged` sidecar is written on conflict)

#### Scenario: update requires a prior install

- **GIVEN** a declared agent with no lockfile record
- **WHEN** `homonto agents update <name>` runs
- **THEN** it errors that the agent is not installed and points to `agents add`

```
