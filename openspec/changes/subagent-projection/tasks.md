## 1. Tool-layout fixtures and path mapping (confirm real layout first)

- [x] 1.1 Add real-layout test fixtures for Claude `agents/` (plural) and OpenCode `agent/` (singular), user and project scope, mirroring the skills/commands fixtures
- [x] 1.2 Add `subagentpath.Dir(tool, scope, home, projectRoot)` (claude `.claude/agents` user / `<repo>/.claude/agents` project; opencode `.config/opencode/agent` user / `<repo>/.opencode/agent` project) — extend `commandpath` or add a sibling package
- [x] 1.3 Unit tests for all tool/scope combinations, asserting the singular/plural split

## 2. Catalog subagent content and embed

- [x] 2.1 Author `catalog/subagents/code-reviewer.md` (framework-agnostic loose subagent, valid frontmatter + body)
- [x] 2.2 Author `catalog/subagents/codebase-explorer.md` (read-only research subagent, valid frontmatter + body)
- [x] 2.3 Author one comet-framework subagent under `catalog/subagents/<name>.md`
- [x] 2.4 Extend the root `catalog` package `//go:embed` directive to include `all:subagents`
- [x] 2.5 Verify the embed compiles and all three subagents are present in the embedded FS

## 3. Catalog subagent loading, expansion, materialization

- [x] 3.1 Parse an optional `[subagents]` table into `Framework.Subagents` (name → `subagents/<n>.md`); validate each path exists in the embedded FS
- [x] 3.2 Index subagents and add a subagent-path lookup (`SubagentPath(name)`)
- [x] 3.3 Add `ExpandSubagents` (transitive, deduped), mirroring `ExpandCommands`
- [x] 3.4 Add single-file **verbatim** materialization to `.homonto/catalog/subagents/<n>.md`, version-gated (assert byte-for-byte equal to source)
- [x] 3.5 Add the comet framework's `[subagents]` entry to `catalog/frameworks/comet/framework.toml`
- [x] 3.6 Unit tests: subagent table parse, expansion/dedup, single-file materialize, missing-file re-materialize, no-model-injection (content equals source)

## 4. Config subagent expansion

- [x] 4.1 Add `Config.ExpandedSubagentEntriesForTool(tool)` (explicit `[subagents.X]` + framework-expanded subagents, scope/targets inheritance)
- [x] 4.2 Collision detection (explicit vs framework subagent name) and cycle propagation
- [x] 4.3 Verify `EnabledModelTools`/`validateModels` already counts subagent-targeted tools; add a test asserting a subagent enabling a tool without model routes fails clearly
- [x] 4.4 Config tests for subagent expansion, inheritance, collision, target filtering

## 5. Engine materialization orchestration

- [ ] 5.1 Extend catalog materialization to collect declared builtin subagent names and materialize them (single-file) before adapters, under the same version gate
- [ ] 5.2 Ensure `CatalogVersion` is recorded only after skills + commands + subagents materialization succeeds
- [ ] 5.3 Add `WithSubagentCatalogRoot` wiring for both adapters
- [ ] 5.4 Engine tests: first-apply subagent materialization, version-gated skip, missing-file refresh

## 6. Adapter subagent projection

- [ ] 6.1 Claude adapter: `subagentsDir(scope)`, `inactiveSubagentsDir`, `subagentSource(entry)`, `subagentLinks`, plan/apply/adopt/prune for `subagent.<n>` links via variadic managed roots
- [ ] 6.2 OpenCode adapter: same, using `subagentpath` (singular `agent/`)
- [ ] 6.3 Extend `managedRoots()` to include the subagent catalog root (non-empty guard)
- [ ] 6.4 `ObserveHashes`: handle `subagent.<n>` as a symlink hash, mirroring `command.<n>`
- [ ] 6.5 Adapter tests (both tools): builtin subagent link create, idempotent re-apply, conflict-not-clobbered, de-declared prune, scope-switch relocate, adopt pre-existing link, state `subagent.<n>` recorded

## 7. Doctor

- [ ] 7.1 Extend `doctor` to verify subagent links and materialized subagent files for both tools
- [ ] 7.2 Doctor test for a linked builtin subagent (both tools)

## 8. Dogfood

- [ ] 8.1 Declare `code-reviewer` and `codebase-explorer` in `homonto.toml` (builtin, scope project); keep `[frameworks.comet]` for the framework subagent
- [ ] 8.2 Run `homonto apply --yes`; verify materialize + link of all three subagents into targeted tools
- [ ] 8.3 Run `homonto status` (No drift) and `homonto doctor` (subagent links ok for both tools)

## 9. Regression and docs

- [ ] 9.1 Full regression: `go test ./... -count=1`, `go test -race ./...`, `go vet ./...`, `go build ./...`, `gofmt -l .`
- [ ] 9.2 Stale-doc grep: no doc claims subagent projection is unimplemented once shipped; update README "Known limitations" and `docs/guides/using-homonto.md`
- [ ] 9.3 Update `docs/roadmap.md` v1.1 status (subagent projection landed with real content) and the "Immediate Next Work" section (item 2 done; onto binary remains)
- [ ] 9.4 Commit all changes
