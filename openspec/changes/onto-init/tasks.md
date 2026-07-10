## 1. Framework-install gate

- [x] 1.1 Add a gate helper in `internal/ontocli/init.go` (or a small sibling): given a workspace root, return a typed result / error for: (a) `homonto.toml` missing, (b) present but no `[frameworks.onto]` table, (c) `[frameworks.onto]` present but `.homonto/catalog/skills/onto/` missing, (d) OK. Read homonto.toml with `go-toml/v2` (a minimal struct with `Frameworks map[string]any` — presence of key `onto` is enough); do NOT call `internal/config.Load` and do NOT construct the engine
- [x] 1.2 Unit tests (TDD, RED first) for all four gate outcomes over temp workspaces, asserting the guidance message content and that NO `docs/` files are created in the three failing cases

## 2. `onto init` command + scaffold

- [x] 2.1 Implement `initCmd()` with a `--dir` flag (default "."): run the gate; on failure print the specific guidance and return a non-zero error WITHOUT touching `docs/`; on success scaffold `docs/{changes,specs,adr,guides}` via `os.MkdirAll`, tracking created-vs-preexisting, and print the report; never overwrite existing paths
- [x] 2.2 Register `initCmd()` on the onto root in `internal/ontocli/root.go`'s `NewRootCmd()`
- [x] 2.3 Tests (TDD, RED first): in a prepared workspace (homonto.toml with `[frameworks.onto]` + a fake `.homonto/catalog/skills/onto/` dir) init creates the four dirs and reports created, exit 0; a second run is idempotent (pre-existing dirs + any user file under docs/ untouched, reported skipped); gate-failure cases create no docs/ files and exit non-zero
- [x] 2.4 Confirm `onto init` does not import/run `internal/engine` or `internal/adapter`; keep the ontocli isolation from #1 (no homonto projection engine)

## 3. Regression and docs

- [ ] 3.1 Full regression: `go build ./...`, `go test ./... -count=1`, `go test -race ./...`, `go vet ./...`, `gofmt -l .` (empty), `go mod tidy` clean; confirm both binaries build and `onto init --help` shows the command
- [ ] 3.2 Update `docs/roadmap.md` "Immediate Next Work": mark onto #2 (`onto init`) landed; remaining onto work = phase-gates (#3), doctor (#4), dual-binary packaging (#5). Do not over-claim
- [ ] 3.3 Commit all changes
