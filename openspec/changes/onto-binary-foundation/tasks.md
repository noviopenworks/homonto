## 1. onto-state.yaml model (`internal/ontostate`)

- [x] 1.1 Add `gopkg.in/yaml.v3` to go.mod (`go get gopkg.in/yaml.v3`); run `go mod tidy`
- [x] 1.2 Define the `State` struct (change id/name, `phase`, minimal gate fields) with yaml tags in `internal/ontostate/state.go`
- [x] 1.3 Implement `Parse([]byte) (State, error)` / `Load(path)` — unmarshal + wrap YAML/os errors with the file name; never panic
- [x] 1.4 Implement `Validate()` (phase is one of open|design|build|verify|close — the onto workflow phase set, matching the onto-* skills and legacy state.yaml) and `DerivePhase() (string, error)` (validated recorded phase)
- [x] 1.5 Unit tests: valid parse+derive, malformed-YAML error names the file, unknown-phase error, missing-file error

## 2. onto binary + CLI root (`cmd/onto`, `internal/ontocli`)

- [x] 2.1 Create `internal/ontocli/root.go`: `Version` var (ldflags-stampable) + `NewRootCmd()` (Use "onto", SilenceUsage/Errors) + `version` subcommand, mirroring `internal/cli/root.go`
- [x] 2.2 Create `cmd/onto/main.go` (`package main`) calling `ontocli.NewRootCmd().Execute()`, mirroring root `main.go`
- [x] 2.3 Verify `go build ./cmd/onto` produces the binary and `go build ./...` still builds `homonto`
- [x] 2.4 Test: `onto version` prints `onto <Version>`; a stamped `-ldflags -X ...Version=` value is reflected (build-tag or ldflags test, or a unit test on the version command output)

## 3. `onto status` (read-only, config-independent)

- [x] 3.1 Implement `statusCmd()` in `internal/ontocli`: walk `docs/changes/*/onto-state.yaml`, load each via `internal/ontostate`, print a per-change phase line; report unreadable/malformed changes as invalid without aborting the run
- [x] 3.2 Register `statusCmd()` on the onto root; ensure it never constructs the homonto config/engine and never writes
- [x] 3.3 Tests: status over a temp `docs/changes/` with a valid change (phase reported) and an invalid one (flagged), asserting exit 0 and that no file was created/modified/removed (read-only)
- [x] 3.4 Test: status works with no `homonto.toml` present (degraded/config-independent)

## 4. Regression and docs

- [ ] 4.1 Full regression: `go test ./... -count=1`, `go test -race ./...`, `go vet ./...`, `go build ./...`, `gofmt -l .`, `go mod tidy -diff` (or `go mod verify`)
- [ ] 4.2 Update `docs/road-to-release.md` / `docs/roadmap.md` to note the onto binary foundation (binary + state model + `onto status`) has landed; onto init/gates/doctor/packaging remain (changes #2–#5)
- [ ] 4.3 Commit all changes
