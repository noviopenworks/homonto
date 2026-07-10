---
change: onto-skeleton
design-doc: docs/superpowers/specs/2026-07-10-onto-skeleton-design.md
base-ref: 08834df8055bf35b1699a98942df4b44e9fbb980
archived-with: 2026-07-10-onto-skeleton
---

# Onto Skeleton Implementation Plan

> Implement task-by-task with TDD.

## Global Constraints (Design Doc + delta spec)

- onto binary #3a. Adds `onto-state.yaml` writer, `onto new <change>`, and
  phase-aware skeleton validation surfaced by `onto status`.
- `onto` stays isolated: `internal/ontocli`/`internal/ontostate`/`cmd/onto`
  import NONE of homonto's internal/{cli,engine,config,adapter,catalog}. Writer
  does NOT import internal/fsutil (self-contained atomic write).
- onto phases: open|design|build|verify|close. `onto new` seeds phase `open`.
- Mutating commands (`onto new`) run the framework-install `gate()` first and
  write NOTHING on gate failure; never clobber an existing change.
- No new dependency (yaml.v3 already present). Gates: `go build ./...`, `go test
  [-race] ./...`, `go vet`, `gofmt -l .`, `go mod tidy` clean; both binaries build.

## Task 1: onto-state.yaml writer + skeleton validation (`internal/ontostate`)

**Files:** modify `internal/ontostate/state.go`, `internal/ontostate/state_test.go`.

- [x] 1.1 (RED first) `Marshal(s State) ([]byte, error)` = `yaml.Marshal(s)`; `Save(path string, s State) error` = MkdirAll parent (0o755), write `path+".tmp"` variant via os.WriteFile (0o644), os.Rename to path, remove temp on error. Tests: `Parse(Marshal(s))` deep-equals `s`; `Save` then `Load` round-trips; Save into a non-existent subdir creates it.
- [x] 1.2 (RED first) `RequiredArtifacts(phase string) []string` (open → `["onto-state.yaml","proposal.md","tasks.md"]`; other phases same base set for now); `ValidateSkeleton(changeDir string) error` (Load `<dir>/onto-state.yaml`, DerivePhase, os.Stat each required, error names first missing). Tests: ok; missing tasks.md → error containing "tasks.md".
- [x] 1.3 GREEN; gofmt/vet clean for internal/ontostate.
- [x] 1.4 Commit: `feat(ontostate): onto-state.yaml writer + skeleton validation`

## Task 2: `onto new <change>` command (`internal/ontocli`)

**Files:** create `internal/ontocli/new.go`, `internal/ontocli/new_test.go`; modify `internal/ontocli/root.go` (register).

- [x] 2.1 (RED first) Local `validChangeName(name string) error` in internal/ontocli: reject empty, `name != filepath.Base(name)`, contains `..`/`/`/`\`, or not matching `^[a-z0-9]+(-[a-z0-9]+)*$`. Tests: valid (`feature-x`) + invalid (``, `../evil`, `Foo`, `a/b`, `-x`).
- [x] 2.2 (RED first) `newCmd()` positional `<change-name>` + `--dir` (default "."): gate(root) → validChangeName → refuse if `docs/changes/<name>/` exists (non-zero, no writes) → MkdirAll dir, `ontostate.Save(<dir>/onto-state.yaml, State{Change:name,Workflow:"full",Phase:"open",Created:time.Now().Format("2006-01-02")})`, os.WriteFile empty proposal.md + tasks.md (only if absent); print created report, exit 0. Register `newCmd()` in `NewRootCmd()`.
- [x] 2.3 (RED first) Tests via `NewRootCmd().SetArgs([]string{"new",name,"--dir",tmp})` in a prepared workspace (homonto.toml `[frameworks.onto]` + `.homonto/catalog/skills/onto/`): creates skeleton (onto-state.yaml Parses to phase open + change name; proposal.md + tasks.md exist), exit 0; second `new same-name` → non-zero, and a pre-placed `docs/changes/<name>/proposal.md` with known bytes is UNCHANGED; invalid name → non-zero, `docs/changes/<name>` absent; gate-failure (empty dir) → non-zero, no docs/ writes; `created` matches `^\d{4}-\d{2}-\d{2}$`.
- [x] 2.4 GREEN; `grep -E "internal/(config|engine|adapter|catalog)" internal/ontocli/*.go` empty; gofmt/vet clean.
- [x] 2.5 Commit: `feat(onto): 'onto new' creates a gated, no-clobber change skeleton`

## Task 3: status skeleton reporting

**Files:** modify `internal/ontocli/status.go`, `internal/ontocli/status_test.go`.

- [x] 3.1 (RED first) Append per-change `" — skeleton ok"` / `" — skeleton: missing <file>"` to each status line via `ontostate.ValidateSkeleton`; still read-only, non-aborting on one bad change. Tests: complete open skeleton → ok note; change missing tasks.md → missing note; the existing read-only tree-snapshot test still passes.
- [x] 3.2 GREEN; gofmt/vet clean.
- [x] 3.3 Commit: `feat(onto): 'onto status' reports skeleton validity`

## Task 4: regression and docs

- [x] 4.1 Full regression: `go build ./...` (both binaries), `go test ./... -count=1`, `go test -race ./...`, `go vet ./...`, `gofmt -l .` (empty), `go mod tidy` then `git diff --exit-code go.mod go.sum` (clean); E2E: in a prepared temp workspace `onto new demo` creates the skeleton and `onto status` reports it ok.
- [x] 4.2 Update `docs/roadmap.md` "Immediate Next Work": onto #3a (`onto new` + skeleton validate) landed; remaining onto = phase transitions (#3b), deps+archive/close (#3c), doctor (#4), dual-binary packaging (#5). No over-claim.
- [x] 4.3 Commit all changes.

## Self-Review

- Writer self-contained (no fsutil); onto isolation preserved.
- `onto new` gate-first, no-clobber (pre-placed-file-untouched test), name-validated.
- Both binaries build; new + status registered.
