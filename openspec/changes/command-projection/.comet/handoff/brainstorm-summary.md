# Brainstorm Summary

- Change: command-projection
- Date: 2026-07-10

## Confirmed Technical Approach

Thin parallel of the archived catalog-foundation-skills change, for single-file
commands. Resolves the three flagged deep-design decisions:

1. **Single-file materialization → sibling `MaterializeCommands(dstRoot, names)`**
   in `internal/catalog`. Skills walk a sub-FS dir; commands read one embedded
   file (`commands/<n>.md`) and write `.homonto/catalog/commands/<n>.md` (0644).
   Distinct enough that a focused sibling beats a kind-branching materializer.

2. **Command path → new `internal/commandpath` package** mirroring `skillpath`
   (claude `.claude/commands`, opencode `.config/opencode/command` user /
   `.opencode/command` project). Zero churn to skillpath's callers. Future: if
   change C adds subagents, consider unifying into a `resourcepath.Dir(kind,…)`.

3. **Framework command expansion → factor the shared DFS.** Extract the private
   three-color cycle-detecting traversal into `expandResources(names, selector)`;
   `Expand` (skills) and new `ExpandCommands` become thin wrappers passing the
   `Framework.Skills` / `Framework.Commands` selector. Avoids duplicating
   cycle detection. `ExpandedCommand{Name, Framework}` mirrors `ExpandedSkill`.

Other refinements:
- **Catalog loader**: parse optional `[commands]` into `Framework.Commands`
  (name → `commands/<n>.md`); validate each path exists; global `commands` index +
  `CommandPath(name)`.
- **Config**: `ExpandedCommandEntriesForTool(tool)` mirrors the skill method.
  Collision is command-vs-command only (commands and skills are separate
  namespaces); explicit `[commands.X]` vs framework command → error; cycles
  surface from `ExpandCommands`.
- **Adapter**: add a parallel `commandCatalogRoot` field (`.homonto/catalog/commands`)
  via `WithCommandCatalogRoot`, a `commandsDir(scope)` (from `commandpath`), and
  `commandSource(entry)` (builtin → commandCatalogRoot/<n>.md, else
  homonto/commands/<n>.md). State key `command.<n>` parallels `skill.<n>`.
  `managedRoots()` returns the non-empty set {content, catalogRoot,
  commandCatalogRoot} — reuses the empty-root guard (M1). Plan/apply/prune/adopt
  reuse `internal/link` (already variadic multi-root) unchanged.
  (Zero-risk to shipped skill code; alternative "unify catalogRoot to
  `.homonto/catalog` base" noted as a future cleanup.)
- **Engine**: `materializeCatalog` also collects declared builtin command names,
  calls `MaterializeCommands` before adapters, under the SAME version gate;
  `CatalogVersion` recorded only after skills + commands both succeed. Adapters
  wired with `WithCommandCatalogRoot(.homonto/catalog/commands)`.
- **Doctor**: verify recorded `command.<n>` links + materialized command files.
- **Fixture**: one flat placeholder `catalog/commands/<placeholder>.md`, declared
  in homonto.toml for dogfood.

## Key Trade-offs and Risks

- Parallel roots/packages (commandpath, commandCatalogRoot) vs unifying now —
  chose parallel for zero risk to the just-shipped skill code; unification is a
  clean follow-up when subagents land.
- Four managed roots now — mitigated by the M1 empty-root guard already in
  `managedRoots()`.
- Single-file vs dir materialization divergence — kept minimal via the sibling
  function; link/state/version logic fully reused.

## Testing Strategy

- internal/catalog: `[commands]` parse + missing-path failure; `ExpandCommands`
  transitive/dedup/cycle (shared DFS); single-file materialize into `t.TempDir`
  incl. missing-file re-materialize.
- internal/commandpath: all tool/scope combos.
- internal/config: `ExpandedCommandEntriesForTool` expansion, inheritance,
  command-vs-command collision, target filtering.
- adapters (both): builtin command link create/idempotent/prune/conflict, state
  `command.<n>` recorded.
- engine: first-apply command materialization, version-gated skip, missing-file
  refresh, combined skills+commands version recording.
- Regression + dogfood: apply/status/doctor on the placeholder builtin command.

## Spec Patches

None anticipated — the delta specs cover the scenarios. (If a boundary gap
surfaces while writing the Design Doc, propose a supplementary scenario only.)
