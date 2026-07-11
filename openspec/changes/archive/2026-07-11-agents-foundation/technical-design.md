---
comet_change: agents-foundation
role: technical-design
canonical_spec: openspec
archived-with: 2026-07-11-agents-foundation
status: final
---

# Agents Foundation — Technical Design

Refinement of `openspec/changes/agents-foundation/design.md`. Roadmap v2 #1: the
`[agents.<name>]` lifecycle model + a read-only `homonto agents list`. No
mutation/lockfile/projection (deferred). Reuses existing config validation
helpers.

## Model (`internal/config/config.go`)

```go
// Agent is a v2 lifecycle-managed agent (distinct from the v1 [subagents]
// symlink Resource): it carries version + mode for update/migration later.
type Agent struct {
    Source  string   `toml:"source"`  // builtin:<name> | local:<name>
    Version string   `toml:"version"` // optional; empty = unpinned
    Targets []string `toml:"targets"` // optional; empty = both tools
    Mode    string   `toml:"mode"`    // optional; copy | link (empty = link)
}
func (a Agent) TargetsOrAll() []string {
    if len(a.Targets) == 0 { return []string{"claude", "opencode"} }
    return a.Targets
}
func (a Agent) ModeOrDefault() string { if a.Mode == "" { return "link" }; return a.Mode }
// Config gains: Agents map[string]Agent `toml:"agents"`
```

## Validation (`validateAgents`, called from Parse/Load beside validateResources)

```go
func validateAgents(agents map[string]Agent) error {
    for name, ag := range agents {
        if err := validateKey("agents", name); err != nil { return err }
        label := "agents." + name
        if !validSource(ag.Source) {
            return fmt.Errorf("parse config: %s source %q is invalid; use builtin:<name> or local:<name>", label, ag.Source)
        }
        switch ag.Mode {
        case "", "copy", "link":
        default:
            return fmt.Errorf("parse config: %s mode %q is invalid; valid values are \"copy\" and \"link\"", label, ag.Mode)
        }
        for _, target := range ag.Targets {
            if target != "claude" && target != "opencode" {
                return fmt.Errorf("parse config: %s targets unknown tool %q; valid targets are \"claude\" and \"opencode\"", label, target)
            }
        }
    }
    return nil
}
```
Call it from the same place `validateResources(...)` is called in Parse/Load.

## `homonto agents list` (`internal/cli/agents.go`)

```go
func agentsCmd() *cobra.Command {
    cmd := &cobra.Command{Use: "agents", Short: "Inspect lifecycle-managed agents"}
    cmd.AddCommand(agentsListCmd())
    return cmd
}
func agentsListCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "list",
        Short: "List declared agents (read-only)",
        Args:  cobra.NoArgs,
        RunE: func(cmd *cobra.Command, _ []string) error {
            cfgPath, _ := cmd.Flags().GetString("config")
            c, err := config.Load(cfgPath)
            if err != nil { return err }
            names := make([]string, 0, len(c.Agents))
            for n := range c.Agents { names = append(names, n) }
            sort.Strings(names)
            if len(names) == 0 { cmd.Println("No agents declared."); return nil }
            for _, n := range names {
                ag := c.Agents[n]
                v := ag.Version; if v == "" { v = "unpinned" }
                cmd.Printf("%s: %s  version=%s  targets=%s  mode=%s\n",
                    n, ag.Source, v, strings.Join(ag.TargetsOrAll(), ","), ag.ModeOrDefault())
            }
            return nil
        },
    }
}
```
Register `agentsCmd()` on the root in `root.go` next to the other commands.
Read-only: it only `config.Load`s — never builds the engine, never writes.
`--config` is the root's persistent flag (inherited).

## Tests

- config (`config_test.go`): parse a full agent (all fields); defaults (no
  version→unpinned via empty, no targets→both via TargetsOrAll, no mode→link via
  ModeOrDefault); invalid source `https://…` rejected; invalid mode `symlink`
  rejected; unknown target rejected.
- cli (`agents_test.go`): `NewRootCmd().SetArgs(["agents","list","--config",p])`
  with a temp toml — two agents printed sorted with source/version/targets/mode;
  unpinned agent shows `unpinned`; no `[agents]` → `No agents declared.`; the
  command loads config only (read-only — no engine, no writes). Mirror
  `status_test.go` harness for building the root cmd + capturing output.

## Verification

TDD RED→GREEN; full regression. E2E (real `homonto` binary): a toml with a
pinned builtin agent + an unpinned local `mode="copy"` agent → `homonto agents
list` prints both sorted; an invalid agent source fails `homonto agents list`
(load error).

## Deferred (later v2 increments)

`add`/`update`/`pin`/`doctor`/`migrate`, the lockfile + installed-version state,
compatibility checks per target, three-way-merge/backup, remote sources, and the
`[agents]`-vs-`[subagents]` supersession decision.
