## 1. Catalog commands content and embed

- [x] 1.1 Add `catalog/commands/<placeholder>.md` (one flat placeholder command with frontmatter)
- [x] 1.2 Extend the root `catalog` package `//go:embed` directive to include `all:commands`
- [x] 1.3 Verify the embed compiles and the placeholder command is present in the embedded FS

## 2. Catalog command loading, expansion, materialization

- [x] 2.1 Parse an optional `[commands]` table into `Framework.Commands` (name → `commands/<n>.md`); validate each path exists in the embedded FS
- [x] 2.2 Index commands and add a command-path lookup (`CommandPath(name)`)
- [ ] 2.3 Expand framework commands (transitive, deduped) — extend `Expand` or add `ExpandCommands`
- [ ] 2.4 Add single-file command materialization to `.homonto/catalog/commands/<n>.md`, version-gated
- [ ] 2.5 Unit tests: command table parse, command expansion/dedup, single-file materialize, missing-file re-materialize

## 3. Command path mapping

- [ ] 3.1 Add `commandpath.Dir(tool, scope, home, projectRoot)` (claude `.claude/commands`, opencode `.config/opencode/command` user / `.opencode/command` project)
- [ ] 3.2 Unit tests for all tool/scope combinations

## 4. Config command expansion

- [ ] 4.1 Add `Config.ExpandedCommandEntriesForTool(tool)` (explicit `[commands.X]` + framework-expanded commands, scope/targets inheritance)
- [ ] 4.2 Collision detection (explicit vs framework command name) and cycle propagation
- [ ] 4.3 Config tests for command expansion, inheritance, collision, target filtering

## 5. Engine materialization orchestration

- [ ] 5.1 Extend `materializeCatalog` to collect declared builtin command names and materialize them (single-file) before adapters, under the same version gate
- [ ] 5.2 Ensure `CatalogVersion` is recorded only after skills + commands materialization succeeds
- [ ] 5.3 Engine tests: first-apply command materialization, version-gated skip, missing-file refresh

## 6. Adapter command projection

- [ ] 6.1 Claude adapter: `commandsDir(scope)`, `commandSource(entry)`, plan/apply/prune/adopt for `command.<n>` links via variadic managed roots
- [ ] 6.2 OpenCode adapter: same, using `commandpath` (singular `command/`)
- [ ] 6.3 Extend `managedRoots()` to include the commands roots (non-empty guard)
- [ ] 6.4 Adapter tests (both tools): builtin command link create, idempotent re-apply, conflict-not-clobbered, de-declared prune, state `command.<n>` recorded

## 7. Doctor

- [ ] 7.1 Extend `doctor` to verify command links and materialized command files
- [ ] 7.2 Doctor test for a linked builtin command

## 8. Dogfood

- [ ] 8.1 Declare the placeholder command in `homonto.toml` (builtin, scope project)
- [ ] 8.2 Run `homonto apply --yes`; verify materialize + link into both tools
- [ ] 8.3 Run `homonto status` (No drift) and `homonto doctor` (command link ok)

## 9. Regression and docs

- [ ] 9.1 Full regression: `go test ./... -count=1`, `go vet ./...`, `go build ./...`
- [ ] 9.2 Stale-doc grep: no doc claims command projection is unimplemented for skills-and-commands once shipped
- [ ] 9.3 Update `docs/roadmap.md` v1.1 status (command projection machinery landed; content deferred)
- [ ] 9.4 Commit all changes
