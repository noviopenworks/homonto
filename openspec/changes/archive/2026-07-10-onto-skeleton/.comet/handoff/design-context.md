# Comet Design Handoff

- Change: onto-skeleton
- Phase: design
- Mode: full
- Context hash: 0f626ea1e10e55eb1b873daf706e4815ca9bd54166b81685d5d80e6442d2c65a

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/onto-skeleton/proposal.md

- Source: openspec/changes/onto-skeleton/proposal.md
- Lines: 1-58
- SHA256: ff3a28fa76409b17454b576953d6b25b30693aa0b08b7225c7ff552413debaf2

```md
## Why

Changes #1 (foundation: binary + `onto-state.yaml` model + `onto status`) and #2
(`onto init` scaffolds the `docs/` layout) are archived. The dual-binary design
says `onto` "creates and validates skeletons" — the binary owns the structural
shape of a change while skills/agents fill content. This change is the first
sub-increment of the onto workflow engine (originally scoped as #3
"onto-phase-gates"), which is large enough to split further:

- **#3a onto-skeleton** (this change): `onto new <change>` creates a change
  workspace skeleton, and a skeleton validator checks that the files required for
  a change's recorded phase exist. Adds the `onto-state.yaml` writer.
- #3b — phase-transition gating (valid-gate-only transitions + gate preconditions
  + dirty-worktree blocking). (depends on #3a)
- #3c — dependency resolution + archive/close rules. (depends on #3a, #3b)

## What Changes

- Add a writer to `internal/ontostate`: `Marshal(State) ([]byte, error)` and
  `Save(path string, s State) error` (atomic write), so the binary can create an
  `onto-state.yaml`. Round-trips with the existing `Parse`/`Load`.
- Add `onto new <change-name>`: creates `docs/changes/<change-name>/` with an
  `onto-state.yaml` (`change: <name>`, `workflow: full` default, `phase: open`,
  `created:` today's date) plus empty-but-present `proposal.md` and `tasks.md`
  skeleton files. It runs the existing framework-install gate first, validates
  the change name (kebab-case, no path traversal), and REFUSES (non-zero, no
  writes) if the change directory already exists — never clobbers.
- Add skeleton validation: `internal/ontostate` (or a sibling) exposes a
  `RequiredArtifacts(phase) []string` + a `ValidateSkeleton(changeDir) error`
  that confirms the files required for the change's recorded phase exist (open →
  `onto-state.yaml`, `proposal.md`, `tasks.md`). Surface it via `onto status`
  gaining a per-change "skeleton ok / missing <file>" note (read-only, additive).
- This change does NOT add phase transitions (#3b) or dependency/archive/close
  enforcement (#3c). Skeleton content beyond empty placeholders is the skills'
  job.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `onto-binary`: gains `onto new <change>` (skeleton creation), the
  `onto-state.yaml` writer, and phase-aware skeleton validation (surfaced through
  `onto status`).

## Impact

- `internal/ontostate`: add `Marshal`/`Save` and `RequiredArtifacts`/
  `ValidateSkeleton` (+ tests).
- `internal/ontocli`: new `newCmd()` (`onto new`), registered on the root; extend
  `status` to report skeleton validity per change (read-only).
- No new dependency (yaml.v3 already present). No change to `homonto`,
  `internal/cli`, adapters, engine, config, catalog. `onto` stays isolated from
  the projection pipeline.
- Advances the onto workflow engine (#3a of the onto binary work).

```

## openspec/changes/onto-skeleton/design.md

- Source: openspec/changes/onto-skeleton/design.md
- Lines: 1-83
- SHA256: 7e1329a488ffdd3e61307c11687b45ada65639a9fe47f4395ff7ab8740d81b96

```md
## Context

onto #1 (foundation) and #2 (`onto init`) are archived. This is #3a — the first
sub-increment of the onto workflow engine: `onto new` creates a change skeleton
and skeleton validation checks required-files-per-phase. It needs an
`onto-state.yaml` writer (the model is read-only today).

## Goals / Non-Goals

**Goals:** `internal/ontostate` `Marshal`/`Save` (round-trips `Parse`); `onto new
<change>` (gated, name-validated, no-clobber) creates the open-phase skeleton;
`RequiredArtifacts(phase)` + `ValidateSkeleton(changeDir)`; `onto status` reports
skeleton validity (read-only).

**Non-Goals:** phase transitions (#3b), dependency/archive/close enforcement
(#3c), skeleton content beyond empty placeholders, changing `homonto`/isolation.

## Decisions

**D1 — Writer in `internal/ontostate`, self-contained atomic write.**
`Marshal(State) ([]byte, error)` = `yaml.Marshal`; `Save(path, State)` writes to
`<path>.<pid/rand>.tmp` then `os.Rename` (atomic on same fs), creating parent
dirs with `os.MkdirAll`. To keep `onto` self-contained it does NOT import
homonto's `internal/fsutil`; the temp+rename is a few lines. Round-trip test:
`Parse(Marshal(s)) == s`.

**D2 — `onto new <change>` = gate → validate name → no-clobber → create.**
`newCmd()` (`--dir` default ".", like init/status):
1. Run `gate(root)` (reused from `init.go`); on failure return the guidance error, write nothing.
2. Validate `<change-name>`: non-empty, `filepath.Base(name) == name`, no `..`/`/`/`\`, kebab-case (`^[a-z0-9]+(-[a-z0-9]+)*$`). A local validator in `internal/ontocli` (NOT homonto's `config.validateResourceName` — isolation).
3. If `docs/changes/<name>/` exists → non-zero error "change already exists", write nothing.
4. Create `docs/changes/<name>/`; `ontostate.Save` an `onto-state.yaml` (`change`, `workflow: full`, `phase: open`, `created:` a date passed in — see note); write empty `proposal.md` and `tasks.md` (`os.WriteFile` only if absent). Report created, exit 0.

`created` date: the binary reads the real date at runtime (`time.Now()` is fine
in the binary; only the comet *scripts* forbid Date.now — that constraint is
JS-runtime specific, not Go). Format `YYYY-MM-DD`.

**D3 — `RequiredArtifacts` + `ValidateSkeleton` in `internal/ontostate`.**
`RequiredArtifacts(phase string) []string` returns the required files for a phase
(`open` → `onto-state.yaml`, `proposal.md`, `tasks.md`; other phases return at
least these for now — later phases' extra artifacts like `design.md`/`plan.md`
are refined in #3b). `ValidateSkeleton(changeDir string) error` loads the
change's `onto-state.yaml`, derives the phase, and checks each required artifact
exists; returns an error naming the first missing file. `onto status` calls it
per change and appends "skeleton ok" or "skeleton: missing <file>" to each line —
read-only, and one change's skeleton error never aborts the others.

## Component Boundaries

| Unit | Responsibility | Depends on |
|---|---|---|
| `internal/ontostate` | Marshal/Save, RequiredArtifacts, ValidateSkeleton | yaml.v3, os |
| `internal/ontocli` (new.go) | `onto new` (gate+validate+create) | ontostate, cobra |
| `internal/ontocli` (status.go) | skeleton note per change | ontostate |

`onto` still imports none of homonto's `internal/{cli,engine,config,adapter,catalog}`.

## Risks / Trade-offs

- **`created` timestamp non-determinism in tests** → tests assert the fields that
  matter (change, phase, presence of files) and, for `created`, only that it is a
  well-formed `YYYY-MM-DD`, not a specific value.
- **RequiredArtifacts is coarse for later phases** → deliberately open-phase-first;
  #3b/#3c refine per-phase requirements. Documented.
- **No-clobber via pre-existence check** → a race window exists but is irrelevant
  for a single-shot CLI; the check prevents the common overwrite mistake.

## Testing Strategy

1. ontostate: Marshal/Parse round-trip; Save atomic (file present + parses back);
   RequiredArtifacts(open); ValidateSkeleton ok + missing-file error naming file.
2. `onto new`: prepared workspace → creates skeleton (state phase open, proposal
   + tasks present), exit 0; refuses existing change (no writes); rejects invalid
   names incl. `../evil`; gate-failure → guidance, no writes.
3. status: skeleton-ok and missing-artifact reporting, still read-only (tree
   snapshot).
4. Isolation grep; both binaries build; regression `go test [-race] ./...`, vet,
   gofmt, tidy.

## Open Questions

None blocking. Per-phase required-artifact sets for design/build/verify/close are
refined in #3b alongside phase transitions.

```

## openspec/changes/onto-skeleton/tasks.md

- Source: openspec/changes/onto-skeleton/tasks.md
- Lines: 1-23
- SHA256: be9f5c3dc4f7558b9cde305afb5fdb6f27904ebe48e99b49fda1cc32b72b6cf1

```md
## 1. onto-state.yaml writer + skeleton validation (`internal/ontostate`)

- [ ] 1.1 (TDD, RED first) Add `Marshal(State) ([]byte, error)` (yaml.Marshal) and `Save(path string, s State) error` (write `<path>.<rand>.tmp` via os.WriteFile then os.Rename; `os.MkdirAll` parent; no fsutil import). Test: `Parse(Marshal(s))` equals `s`; `Save` then `Load` round-trips; parent dir created
- [ ] 1.2 (TDD, RED first) Add `RequiredArtifacts(phase string) []string` (open → onto-state.yaml, proposal.md, tasks.md) and `ValidateSkeleton(changeDir string) error` (Load onto-state.yaml, DerivePhase, check each required artifact exists, error names first missing file). Tests: ok case; missing-tasks.md error names the file
- [ ] 1.3 Run → GREEN; gofmt/vet clean for internal/ontostate

## 2. `onto new <change>` command (`internal/ontocli`)

- [ ] 2.1 (TDD, RED first) Add a local kebab-case name validator in internal/ontocli (`^[a-z0-9]+(-[a-z0-9]+)*$`, reject empty / `..` / `/` / `\` / non-Base). Tests for valid + several invalid names (incl. `../evil`, `Foo`, ``)
- [ ] 2.2 (TDD, RED first) Implement `newCmd()` (`--dir` default "."): run `gate(root)` (reuse from init.go) → validate name → if `docs/changes/<name>/` exists return non-zero "already exists" (no writes) → else create dir, `ontostate.Save` onto-state.yaml (change, workflow full, phase open, created `time.Now().Format("2006-01-02")`), write empty `proposal.md` + `tasks.md`; report created, exit 0. Register `newCmd()` on the root
- [ ] 2.3 (TDD) Tests via `NewRootCmd().SetArgs([]string{"new","<name>","--dir",tmp})`: prepared workspace creates skeleton (onto-state.yaml phase open + proposal + tasks), exit 0; existing change refused with no writes (assert a pre-placed file under docs/changes/<name>/ untouched); invalid name rejected, nothing created; gate-failure → guidance, nothing created
- [ ] 2.4 Run → GREEN; confirm (grep) new.go imports no internal/{config,engine,adapter,catalog}; gofmt/vet clean

## 3. status skeleton reporting

- [ ] 3.1 (TDD, RED first) Extend `onto status` to append per-change "skeleton ok" / "skeleton: missing <file>" via `ontostate.ValidateSkeleton`, still read-only and non-aborting on one bad change. Tests: complete open skeleton → ok; missing tasks.md → missing note; read-only tree snapshot still holds
- [ ] 3.2 Run → GREEN; gofmt/vet clean

## 4. Regression and docs

- [ ] 4.1 Full regression: `go build ./...` (both binaries), `go test ./... -count=1`, `go test -race ./...`, `go vet ./...`, `gofmt -l .` (empty), `go mod tidy` clean; `onto new --help` and `onto status` work; a fresh `onto new demo` in a prepared temp workspace creates the skeleton and `onto status` reports it ok
- [ ] 4.2 Update `docs/roadmap.md` "Immediate Next Work": onto #3a (`onto new` skeleton create + validate) landed; remaining onto = phase transitions (#3b), deps+archive/close (#3c), doctor (#4), dual-binary packaging (#5). No over-claim
- [ ] 4.3 Commit all changes

```

## openspec/changes/onto-skeleton/specs/onto-binary/spec.md

- Source: openspec/changes/onto-skeleton/specs/onto-binary/spec.md
- Lines: 1-69
- SHA256: 95a7781d55438e0f8c09b1e2da7299d3a3957db83f9b56433ed5a34ec84ff889

```md
## ADDED Requirements

### Requirement: onto-state.yaml writer

`internal/ontostate` SHALL provide a serializer that round-trips with its parser:
`Marshal(State) ([]byte, error)` producing YAML that `Parse` reads back to an
equal `State`, and `Save(path string, s State) error` writing that YAML
atomically (temp + rename). `Save` SHALL NOT clobber via a partial write on
error.

#### Scenario: state round-trips through Marshal and Parse

- **GIVEN** a valid `State` (change + phase build)
- **WHEN** it is `Marshal`ed and the bytes are `Parse`d back
- **THEN** the parsed `State` equals the original (change, phase, and any set fields)

### Requirement: onto new creates a change skeleton

`onto new <change-name>` SHALL create `docs/changes/<change-name>/` containing an
`onto-state.yaml` (`change` = the name, `workflow` defaulting to `full`, `phase`
= `open`, `created` = the current date) and empty-but-present `proposal.md` and
`tasks.md` skeleton files. It SHALL run the framework-install gate first (same as
`onto init`), SHALL validate `<change-name>` is kebab-case with no path traversal
(reject `..`, `/`, empty), and SHALL REFUSE with a non-zero exit and NO writes if
`docs/changes/<change-name>/` already exists (never clobber an existing change).

#### Scenario: new creates the open-phase skeleton

- **GIVEN** a prepared workspace (framework-install gate passes) with no `docs/changes/feature-x/`
- **WHEN** `onto new feature-x` runs
- **THEN** `docs/changes/feature-x/onto-state.yaml` (phase open), `proposal.md`, and `tasks.md` exist, and the command reports the created change, exiting 0

#### Scenario: new refuses to clobber an existing change

- **GIVEN** `docs/changes/feature-x/` already exists (with content)
- **WHEN** `onto new feature-x` runs
- **THEN** it exits non-zero, prints that the change already exists, and modifies no file under `docs/changes/feature-x/`

#### Scenario: new rejects an invalid change name

- **WHEN** `onto new "../evil"` (or a non-kebab-case / empty name) runs
- **THEN** it exits non-zero with a validation error and creates nothing

#### Scenario: new requires the framework install

- **GIVEN** a workspace without `homonto.toml` or `[frameworks.onto]` or the applied onto framework
- **WHEN** `onto new feature-x` runs
- **THEN** it prints the same framework-install guidance as `onto init`, creates nothing, and exits non-zero

### Requirement: phase-aware skeleton validation

`internal/ontostate` SHALL expose the artifacts required for each workflow phase
(`RequiredArtifacts(phase) []string`) and a `ValidateSkeleton(changeDir) error`
that confirms the files required for the change's recorded phase are present. For
the `open` phase the required artifacts SHALL be `onto-state.yaml`, `proposal.md`,
and `tasks.md`. `onto status` SHALL report each change's skeleton validity
(e.g. "skeleton ok" or "skeleton: missing <file>") without writing any file.

#### Scenario: status reports a complete open-phase skeleton as ok

- **GIVEN** a change at phase open with `onto-state.yaml`, `proposal.md`, `tasks.md`
- **WHEN** `onto status` runs
- **THEN** it reports the change's phase and that its skeleton is ok, writing nothing

#### Scenario: status reports a missing required artifact

- **GIVEN** a change at phase open missing `tasks.md`
- **WHEN** `onto status` runs
- **THEN** it reports the change's skeleton as missing `tasks.md`, still writing nothing and not aborting other changes

```
