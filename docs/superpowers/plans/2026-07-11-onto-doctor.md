---
change: onto-doctor
design-doc: docs/superpowers/specs/2026-07-11-onto-doctor-design.md
base-ref: 57140e3991dd9a790c24ca3d038979bfe86b4d95
---

# Plan: onto doctor (#4)

Read-only workflow/project health command for the `onto` binary. Assembles
existing `internal/ontostate` primitives + `ontocli.docsLayout` into one report;
no new `ontostate` API. See the Design Doc for the concrete control flow, finding
formats, edge cases, and full test matrix.

## Task 1: `onto doctor` command (`internal/ontocli`)

- [ ] 1.1 (TDD, RED first) Add `internal/ontocli/doctor.go`: `doctorCmd()`
  (`cobra.NoArgs`, `--dir` default ".") + `runDoctor(cmd, root)`. Read-only,
  ungated (NO `gate()` call). Accumulate `findings []string`:
  - **docs layout**: `os.Stat` each of `docsLayout`; missing/non-dir → finding
    `"docs layout: missing directory <d>"`.
  - **active changes**: `filepath.Glob(root/docs/changes/*/onto-state.yaml)`
    (single `*` excludes archive). Per change (keyed by dir basename):
    `ontostate.Load` err → `"<name>: invalid onto-state.yaml: <err>"`, continue;
    `DerivePhase` err → `"<name>: cannot derive phase: <err>"`, continue;
    `ValidateSkeleton(changeDir)` err → `"<name>: phase <phase> missing
    artifact: <err>"`; `DepsResolved(root, st.Deps)` non-empty →
    `"<name>: unresolved dependencies: <list>"`; `st.Archived` →
    `"<name>: active change marked archived: true …"`.
  - **archive layout**: `filepath.Glob(root/docs/changes/archive/*)`; per dir
    entry: `Load` `onto-state.yaml` err → `"archive/<name>: invalid or missing
    onto-state.yaml: <err>"`, continue; `!st.Archived` → `"archive/<name>: not
    marked archived: true"`. Non-dir entries ignored.
  - Verdict: 0 findings → `cmd.Println("healthy")`, return nil; else print each
    finding to `cmd.OutOrStdout()` then return `fmt.Errorf("onto doctor: %d
    problem(s) found", len(findings))`.
  RED first: write `doctor_test.go` cases (below) and confirm they fail before
  implementing.
- [ ] 1.2 (TDD) Tests `internal/ontocli/doctor_test.go` via
  `NewRootCmd().SetArgs([]string{"doctor","--dir",tmp})` + `cmd.SetOut(&buf)` +
  `Execute()`. Seed helpers: `seedDocsLayout`, `seedActive(name,phase,artifacts,
  deps)`, `seedArchived(name,archived)`. Cases (assert err nil/non-nil AND
  stdout content; negative cases assert no files created):
  1. healthy → nil err, stdout `healthy`;
  2. missing `docs/adr` → non-nil, names it;
  3. invalid active state (malformed YAML) → non-nil, names change + invalid;
  4. phase `build` without `plan.md` → non-nil, names missing artifact;
  5. unresolved dep (`deps:[missing]`) → non-nil, contains `missing`;
  6. active change `archived:true` → non-nil, contains `archived`;
  7. archive entry with `archived:false` → non-nil, names the entry;
  8. ungated read-only (no `homonto.toml`, missing docs) → still runs (non-nil
     for layout findings), asserts NO new files created.
- [ ] 1.3 GREEN. Register `doctorCmd()` on the root in `root.go`.
  `grep -nE "internal/(config|engine|adapter|catalog)" internal/ontocli/*.go`
  empty (isolation held). `gofmt -l internal/ontocli` empty; `go vet
  ./internal/ontocli/...` clean.
- [ ] 1.4 Commit: `feat(onto): 'onto doctor' reports workflow/project health (read-only, non-zero on findings)`

## Task 2: Regression and docs

- [ ] 2.1 Full regression: `go build ./...` (both binaries), `go test ./...
  -count=1`, `go test -race ./...`, `go vet ./...`, `gofmt -l .` (empty),
  `go mod tidy` clean; E2E in a temp git workspace: build `onto`, run `onto
  doctor` on a healthy scaffolded workspace (exit 0, `healthy`), then break it
  (remove a docs dir / seed an invalid change) → `onto doctor` exits non-zero
  naming the problem.
- [ ] 2.2 Update `docs/roadmap.md` "Immediate Next Work": onto #4 (`onto
  doctor`) landed — the onto binary now exposes advance/close/doctor/init/new/
  status/version; only dual-binary release packaging (#5) remains before
  `v0.1.0-rc.1`. No over-claim.
- [ ] 2.3 Commit all changes.
