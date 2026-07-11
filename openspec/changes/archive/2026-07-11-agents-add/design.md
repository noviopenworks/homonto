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
exact `Installed[tool].Path`. Anything else present at `dst` is a foreign file →
conflict. (For `link` mode, an existing symlink we created and still points at the
source is a no-op; reuse `link.Link`'s own managed-root check if convenient, else
compare `os.Readlink`.)

## Risks / Trade-offs

- **Lockfile vs state.json**: intentionally separate — agent lifecycle needs
  installed-version ground truth distinct from the plan/apply drift model. A later
  `agents doctor` reads this lockfile.
- **User scope only**: agents have no scope field yet; installing at user scope is
  the sensible default (matches how a global agent is shared). A scope field is a
  later increment.
- **Copy vs link idempotency**: copy compares content hash; link compares the
  symlink target. Both give a clean no-op on re-run.
- **subagent dir collision**: agents install into the same tool agent dir as
  `[subagents]`. Names could collide; the conflict check refuses to clobber a
  file we don't own (including a subagent-projected one), so it is safe. The
  `[agents]`-vs-`[subagents]` reconciliation is a documented later decision.

## Migration Plan

Additive; the lockfile is created on first `agents add`. No migration.

## Open Questions

None for this increment. builtin/remote resolution and update/migrate are
deferred, scoped follow-ups.
