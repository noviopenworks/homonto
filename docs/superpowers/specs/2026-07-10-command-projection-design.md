---
comet_change: command-projection
role: technical-design
canonical_spec: openspec
archived-with: 2026-07-10-command-projection
status: final
---

# Command Projection — Technical Design

Deep refinement of the open-phase `design.md` (D1–D7). OpenSpec delta specs
(`command-projection` new; `config-model`, `framework-expansion` modified) remain
the canonical WHAT. This is a thin parallel of the archived
`catalog-foundation-skills` implementation, applied to **single-file commands**.
It reuses `internal/link` (variadic multi-root) and `internal/state`
(`CatalogVersion`) unchanged.

## 1. Catalog: commands area, loading, expansion, materialization

### Embed + content
`catalog/commands/<name>.md` flat files. The root `catalog` package's directive
becomes `//go:embed all:frameworks all:skills all:commands version.txt`. One
placeholder command ships (§7).

### Loader (`internal/catalog/catalog.go`)
`Framework` gains `Commands map[string]string` (name → `commands/<n>.md`). `Load`
parses an optional `[commands]` table, validates each command path exists in the
embedded FS (same as skills), and builds a global `commands map[string]string`
index. Add `CommandPath(name) (string, bool)`. `frameworkTOML` gains a
`Commands map[string]string \`toml:"commands"\`` field.

### Expansion (`internal/catalog/expand.go`)
Factor the existing three-color cycle-detecting DFS into a private helper:

```go
type Expanded struct{ Name, Framework string }
func (c *Catalog) expandResources(frameworkNames []string,
    sel func(Framework) map[string]string) ([]Expanded, error)
```

`Expand` (skills) delegates with `sel = func(f Framework) map[string]string {
return f.Skills }` and adapts to `[]ExpandedSkill`; new `ExpandCommands` delegates
with `f.Commands` and returns `[]ExpandedCommand{Name, Framework}`. Cycle
detection and dedup live once in `expandResources`.

### Materialization (`internal/catalog/materialize.go`)
Sibling to `Materialize`:

```go
// MaterializeCommands writes each named builtin command from the embedded FS to
// dstRoot/<name>.md (a single file), replacing any existing file.
func (c *Catalog) MaterializeCommands(dstRoot string, names []string) error
```

Per command: resolve `c.commands[name]`, `fs.ReadFile(c.fsys, path)`, ensure
`dstRoot` exists (0755), write `dstRoot/<name>.md` (0644). Unknown name → error.
No `RemoveAll` needed (single file overwrite); an upgrade simply rewrites the file.

## 2. `internal/commandpath`

New package mirroring `skillpath`:

```go
func Dir(tool, scope, home, projectRoot string) string
// claude   user → <home>/.claude/commands       project → <projectRoot>/.claude/commands
// opencode user → <home>/.config/opencode/command project → <projectRoot>/.opencode/command
```

Note OpenCode uses the **singular** `command/`. Unknown tool → "". A future
change adding subagents may unify `skillpath`/`commandpath` into a
`resourcepath.Dir(kind, …)`; out of scope here.

## 3. `internal/config`

`config` already parses `[commands.X]` and validates source/scope/targets. Add:

```go
func (c *Config) CommandEntriesForTool(tool string) []NamedResource   // explicit only
func (c *Config) ExpandedCommandEntriesForTool(tool string) ([]NamedResource, error)
```

`ExpandedCommandEntriesForTool` mirrors `ExpandedSkillEntriesForTool`: explicit
`[commands.X]` for `tool` plus, for each `[frameworks.<fw>]
source="builtin:<fw>"` targeting `tool`, `catalog.ExpandCommands([fw])` → each
command as `NamedResource{Name, Resource{Source:"builtin:"+name, Scope:fwScope,
Targets:fwTargets}}`, inheriting scope/targets. **Collision is command-vs-command
only** — a command name equal to an explicit `[commands.X]` or expanded by two
frameworks with conflicting decls errors; a skill and a command may share a name
(separate namespaces). Cycles surface from `ExpandCommands`. Reuses the same
`loadedCatalog()` singleton.

## 4. `internal/engine`

`Build` computes `commandCatalogRoot := filepath.Join(stateDir, "catalog",
"commands")` and threads it into both adapters via `WithCommandCatalogRoot`.
`materializeCatalog` extends: collect the union of declared builtin **command**
names (across tools, via `ExpandedCommandEntriesForTool`) alongside skill names;
if either set is non-empty, load the catalog; the version gate now also requires
every command file to exist; on miss, `Materialize(...skills)` **and**
`MaterializeCommands(commandCatalogRoot, cmdNames)`, then record `CatalogVersion`
only after **both** succeed (preserves the partial-materialization safety). Add
`CommandDir()` accessor for doctor.

## 5. Adapters (`claude`, `opencode`)

Add `commandCatalogRoot string` + `WithCommandCatalogRoot`. New:

```go
func (a *Adapter) commandsDir(scope string) string        // commandpath.Dir(tool, scope, …)
func (a *Adapter) commandSource(e config.NamedResource) string
// builtin:<n> → filepath.Join(a.commandCatalogRoot, n+".md"); else homonto/commands/<n>.md
```

`Plan` also computes command entries (`ExpandedCommandEntriesForTool`, error
propagated) and produces `command.<n>` link ops exactly like `skill.<n>`
(create/update/adopt/prune), keyed under `commandsDir(scope)`. `managedRoots()`
returns the non-empty subset of `{a.content, a.catalogRoot,
a.commandCatalogRoot}` — the M1 empty-root guard still applies, and every
`link.*` call passes `a.managedRoots()...`. `Apply` links commands and records
`command.<n>` state. Pruning of a de-declared `command.<n>` recovers its dst from
state and removes only a managed-root symlink. `ObserveHashes` treats
`command.<n>` like `skill.<n>` (symlink readlink → `dst -> src` hash). Both
adapters change identically (opencode via its `commandpath` dir).

Note: `a.content` (`homonto/`) already covers `homonto/commands/`, so only the
catalog command root is a genuinely new managed root.

## 6. Doctor (`internal/engine/status.go`)

Extend the skill-link check to also iterate recorded `command.<n>` keys: verify
the tool-side symlink exists and points at the expected `commandSource`, and for
builtin commands that the materialized `.homonto/catalog/commands/<n>.md` exists.
Report missing/broken like skill links.

## 7. Placeholder fixture command

`catalog/commands/<placeholder>.md` — one flat command with a short frontmatter
(`name`, `description`) and body, clearly marked as a placeholder pending real
content. Declared in `homonto.toml` as `[commands.<placeholder>] source =
"builtin:<placeholder>" scope = "project"` for dogfood. Name TBD in build (e.g.
`homonto-catalog-info` or a neutral `example-command`).

## 8. Data flow (apply)

```
config.Load → ExpandedCommandEntriesForTool(tool): explicit [commands.X] +
  framework [commands] expansion (scope/targets inherited)
engine.Apply → materializeCatalog: version-gated Materialize(skills) +
  MaterializeCommands(commands) → record CatalogVersion (after both)
adapter.Apply → link command.<n> dst → .homonto/catalog/commands/<n>.md,
  record state; conflict-safe, prune-safe via variadic managedRoots
status/doctor → drift = link changed OR materialized file missing
```

## 9. Error handling & edge cases

- Missing command path in `framework.toml` → `catalog.Load` fails fast.
- Command-vs-command collision / cycle → error before any write.
- Interrupted materialization → version not recorded → re-materialize next apply.
- Conflict (real file / foreign link at command dst) → reported, never clobbered.
- Empty command set → no catalog command work (skills still materialize).
- Skill and command sharing a name → allowed (separate state keys `skill.<n>` /
  `command.<n>`, separate tool dirs).

## 10. Testing strategy

- `internal/catalog`: `[commands]` parse + missing-path failure; `ExpandCommands`
  transitive/dedup/cycle (shared `expandResources`); `MaterializeCommands` into
  `t.TempDir()` incl. missing-file re-materialize and overwrite.
- `internal/commandpath`: all tool/scope combinations.
- `internal/config`: `ExpandedCommandEntriesForTool` expansion, scope/target
  inheritance, command-vs-command collision, target filtering, skill/command
  name-share allowed.
- adapters (both): builtin command link create, idempotent re-apply, prune of
  de-declared command, conflict-not-clobbered, `command.<n>` recorded.
- `internal/engine`: first-apply command materialization, version-gated skip,
  missing-file refresh, combined skills+commands version recording.
- Regression (`go test ./... -count=1`, vet, build) + dogfood
  `apply/status/doctor` on the placeholder builtin command.

## 11. Spec Patches
None.
