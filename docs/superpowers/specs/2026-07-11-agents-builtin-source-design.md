---
comet_change: agents-builtin-source
role: technical-design
canonical_spec: openspec
archived-with: 2026-07-11-agents-builtin-source
status: final
---


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

- **Reusing the subagent catalog for builtin agents**: a builtin agent == a curated
  catalog subagent file. Coherent (they're the same curated content); documented.
  The `[agents]`-vs-`[subagents]` reconciliation (a later concern) may formalize
  this.
- **Catalog upgrade = builtin source drift**: after a homonto upgrade that changes
  a builtin agent's content, doctor reports "source changed" and `update` merges —
  exactly the desired lifecycle. The `version` field stays informational (no
  catalog-version pinning yet).
- **builtin + link error**: a minor constraint; copy is the norm for
  lifecycle-managed agents anyway.

## Migration Plan

Additive; builtin agents newly resolvable. No migration. `local:` unchanged.

## Open Questions

None — approved scope (builtin near-term, remote deferred).
