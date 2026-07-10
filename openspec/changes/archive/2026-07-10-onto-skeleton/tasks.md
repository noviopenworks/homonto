## 1. onto-state.yaml writer + skeleton validation (`internal/ontostate`)

- [x] 1.1 (TDD, RED first) Add `Marshal(State) ([]byte, error)` (yaml.Marshal) and `Save(path string, s State) error` (write `<path>.<rand>.tmp` via os.WriteFile then os.Rename; `os.MkdirAll` parent; no fsutil import). Test: `Parse(Marshal(s))` equals `s`; `Save` then `Load` round-trips; parent dir created
- [x] 1.2 (TDD, RED first) Add `RequiredArtifacts(phase string) []string` (open → onto-state.yaml, proposal.md, tasks.md) and `ValidateSkeleton(changeDir string) error` (Load onto-state.yaml, DerivePhase, check each required artifact exists, error names first missing file). Tests: ok case; missing-tasks.md error names the file
- [x] 1.3 Run → GREEN; gofmt/vet clean for internal/ontostate

## 2. `onto new <change>` command (`internal/ontocli`)

- [x] 2.1 (TDD, RED first) Add a local kebab-case name validator in internal/ontocli (`^[a-z0-9]+(-[a-z0-9]+)*$`, reject empty / `..` / `/` / `\` / non-Base). Tests for valid + several invalid names (incl. `../evil`, `Foo`, ``)
- [x] 2.2 (TDD, RED first) Implement `newCmd()` (`--dir` default "."): run `gate(root)` (reuse from init.go) → validate name → if `docs/changes/<name>/` exists return non-zero "already exists" (no writes) → else create dir, `ontostate.Save` onto-state.yaml (change, workflow full, phase open, created `time.Now().Format("2006-01-02")`), write empty `proposal.md` + `tasks.md`; report created, exit 0. Register `newCmd()` on the root
- [x] 2.3 (TDD) Tests via `NewRootCmd().SetArgs([]string{"new","<name>","--dir",tmp})`: prepared workspace creates skeleton (onto-state.yaml phase open + proposal + tasks), exit 0; existing change refused with no writes (assert a pre-placed file under docs/changes/<name>/ untouched); invalid name rejected, nothing created; gate-failure → guidance, nothing created
- [x] 2.4 Run → GREEN; confirm (grep) new.go imports no internal/{config,engine,adapter,catalog}; gofmt/vet clean

## 3. status skeleton reporting

- [x] 3.1 (TDD, RED first) Extend `onto status` to append per-change "skeleton ok" / "skeleton: missing <file>" via `ontostate.ValidateSkeleton`, still read-only and non-aborting on one bad change. Tests: complete open skeleton → ok; missing tasks.md → missing note; read-only tree snapshot still holds
- [x] 3.2 Run → GREEN; gofmt/vet clean

## 4. Regression and docs

- [x] 4.1 Full regression: `go build ./...` (both binaries), `go test ./... -count=1`, `go test -race ./...`, `go vet ./...`, `gofmt -l .` (empty), `go mod tidy` clean; `onto new --help` and `onto status` work; a fresh `onto new demo` in a prepared temp workspace creates the skeleton and `onto status` reports it ok
- [x] 4.2 Update `docs/roadmap.md` "Immediate Next Work": onto #3a (`onto new` skeleton create + validate) landed; remaining onto = phase transitions (#3b), deps+archive/close (#3c), doctor (#4), dual-binary packaging (#5). No over-claim
- [x] 4.3 Commit all changes
