## 1. Catalog structure and content

- [x] 1.1 Create `catalog/frameworks/{onto,comet,superpowers,openspec}/framework.toml` with name, version, dependencies, and skills tables
- [x] 1.2 Copy skill content from `homonto/skills/` into `catalog/skills/<name>/` for all bundled skills referenced by frameworks
- [x] 1.3 Add `catalog/version.txt` with the initial catalog version string
- [x] 1.4 Verify all framework.toml files reference skills that exist under `catalog/skills/`

## 2. Catalog Go package

- [x] 2.1 Create `internal/catalog/catalog.go` with embedded FS (`go:embed all:catalog`), framework metadata parser, and framework/skill lookup APIs
- [x] 2.2 Add dependency graph builder with cycle detection and transitive expansion
- [x] 2.3 Add materialization function: extract builtin skill from embedded FS to `.homonto/catalog/skills/<name>/`
- [x] 2.4 Add catalog version read and comparison for re-materialization gating
- [x] 2.5 Write unit tests for catalog parsing, expansion, cycle detection, and materialization

## 3. Config integration

- [x] 3.1 Extend `config.Load` to expand `[frameworks.X]` into effective skill entries with builtin source
- [x] 3.2 Add name collision detection between framework-expanded skills and explicit `[skills.X]` entries
- [x] 3.3 Add `Config.ExpandedSkillEntriesForTool(tool)` that returns effective skills including framework expansion
- [x] 3.4 Write config tests for framework expansion, dependency resolution, collision detection, and cycle rejection

## 4. Engine and materialization orchestration

- [x] 4.1 Add materialization step in engine build/apply: before adapters run, materialize all builtin skills to `.homonto/catalog/skills/`
- [x] 4.2 Track catalog version in state; gate re-materialization on version change
- [x] 4.3 Pass materialized catalog root path to adapters alongside existing content root

## 5. Adapter changes

- [x] 5.1 Extend claude adapter: resolve `builtin:<name>` skills to `.homonto/catalog/skills/<name>/` path
- [x] 5.2 Extend opencode adapter: same builtin source resolution
- [x] 5.3 Update linker managed-root check to accept `.homonto/catalog/skills/` as a valid managed root for pruning
- [x] 5.4 Update doctor to check builtin skill content at materialized path
- [x] 5.5 Write adapter tests for builtin skill projection, pruning, and conflict detection

## 6. Dogfood config update

- [x] 6.1 Update `homonto.toml` to use `[frameworks.comet] source = "builtin:comet"` instead of individual `[skills.X]` entries for Comet/OpenSpec/Superpowers skills
- [x] 6.2 Keep any skills not covered by frameworks as explicit local entries
- [x] 6.3 Run `homonto apply --yes` and verify all skills materialize and link correctly
- [x] 6.4 Run `homonto status` and `homonto doctor` and verify no drift and all links ok

## 7. Regression and docs

- [x] 7.1 Run full regression: `go test ./... -count=1`, `go vet ./...`, `go build ./...`
- [x] 7.2 Run stale-doc grep to ensure no doc claims builtin projection is unimplemented for skills
- [x] 7.3 Update `docs/NEXT_AGENT.md` with catalog-foundation verification evidence
- [x] 7.4 Commit all changes
