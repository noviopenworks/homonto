# onto-binary Specification

## Purpose
Defines the standalone `onto` binary (built from `cmd/onto/`, independent of the
`homonto` binary) and its workflow commands: `version`, `status` (read-only and
config-independent), `init` (scaffold the `docs/{changes,specs,adr,guides}/`
layout behind a Homonto-managed framework-install gate), `new` (create a change
skeleton), `advance` (phase transitions with per-phase artifact and
tasks-complete gating plus dirty-worktree handling), `close` (archive a completed
change with dependency and worktree gates), and `doctor` (read-only workspace
health) — plus the `onto-state.yaml` change-state model, its parser/writer, the
per-phase required-artifacts and skeleton validation, dependency resolution, and
the cross-platform release packaging that ships both binaries for all six
GOOS/GOARCH targets.
## Requirements
### Requirement: Onto binary builds independently of homonto

The repository SHALL build a second binary `onto` from a dedicated
`package main` at `cmd/onto/`, via `go build ./cmd/onto` and installable with
`go install github.com/noviopenworks/homonto/cmd/onto`. The existing root
`homonto` binary (built from `main.go`) SHALL be unchanged, and `go build ./...`
SHALL build both.

#### Scenario: onto binary compiles from its own package main

- **GIVEN** the repository at a clean checkout
- **WHEN** `go build ./cmd/onto` runs
- **THEN** it produces an `onto` executable, and `go build ./...` still builds the `homonto` binary unchanged

### Requirement: Onto CLI root and version

The `onto` binary SHALL expose a Cobra root command `onto` constructed in the
same style as `homonto`'s `internal/cli.NewRootCmd`, with a `version` subcommand
that prints the build version. The version SHALL be a package-level variable
stampable at release time via `-ldflags "-X …Version=<tag>"`, mirroring how
`homonto`'s version is stamped.

#### Scenario: onto version prints the stamped version

- **WHEN** `onto version` runs
- **THEN** it prints `onto <version>` and exits 0
- **AND** a release build with `-ldflags "-X …Version=v0.1.0-rc.1"` prints that tag

### Requirement: onto-state.yaml change-state model

The `onto` binary SHALL read, validate, and write a per-change state file named
`onto-state.yaml` (at `docs/changes/<name>/onto-state.yaml`) through a dedicated
state package, as the single authority for onto workflow state. The model SHALL
parse the file into a typed structure carrying an explicit `schema_version`, a
typed **core** of gated fields, and a carried **observational** group that is
never gated. It SHALL validate the presence and shape of gated fields only
(enum/format), never their substantive quality (B1: the binary rejects a
malformed value, not an unconvincing one). It SHALL derive the current workflow
phase from the core.

The gated core SHALL include at least: change, workflow (`full|fix|tweak`), phase
(`open|design|build|verify|close`), created, base_ref, deps, isolation
(`branch|worktree|""`), build_mode (`direct|subagent|""`), tdd_mode
(`tdd|direct|""`), verify scale (`light|full|""`), verify result
(`pending|pass|fail`), close merged (bool), guides
(`pending|updated|"waived: <reason>"|""`), archived (bool), and the directive
string. Observational fields (metrics, task counts, verify rounds, escalation
flag) SHALL be carried through reads and writes but SHALL never gate a
transition. Writes SHALL always emit the current `schema_version`.

The binary SHALL migrate legacy state on read: a legacy binary `onto-state.yaml`
(no `schema_version`) and a legacy skill `state.yaml` (no `schema_version`) SHALL
each up-migrate to the current schema. Migration SHALL be ordered and idempotent
(loading a current-version file is a no-op). If a change directory holds BOTH a
legacy `onto-state.yaml` and a legacy `state.yaml` whose gated core fields
disagree (phase, workflow, or archived), the state SHALL be reported as malformed
rather than silently resolved. Parsing an invalid or malformed state SHALL return
a clear error identifying the file, not a panic.

The recognized workflow phases are `open`, `design`, `build`, `verify`, `close`,
with `close` as the terminal phase and `archived` as a terminal boolean.

#### Scenario: parse and derive phase from a valid versioned onto-state.yaml

- **GIVEN** a valid `onto-state.yaml` carrying `schema_version`, a gated core, and observational fields
- **WHEN** the state model loads it
- **THEN** it returns the typed state and the derived phase without error, preserving observational fields

#### Scenario: guides accepts pending, updated, and waived forms

- **GIVEN** a change whose `guides` is being set
- **WHEN** it is set to `pending`, `updated`, or a value beginning `waived:`
- **THEN** the value is accepted; any other non-empty value is rejected with a clear error and no write

#### Scenario: legacy state migrates on read

- **GIVEN** a legacy `onto-state.yaml` (no `schema_version`) or a legacy `state.yaml` (no `schema_version`)
- **WHEN** the state model loads it
- **THEN** it up-migrates to the current schema without dropping any gated field, and a subsequent write emits the current `schema_version`

#### Scenario: disagreeing dual legacy files are malformed

- **GIVEN** a change directory holding both a legacy `onto-state.yaml` and a legacy `state.yaml` whose phase, workflow, or archived disagree
- **WHEN** the state model loads the change
- **THEN** it reports the state as malformed and names the conflict, and does not silently pick a winner

#### Scenario: malformed state reports a clear error

- **GIVEN** a state file that is not valid YAML or fails presence/shape validation
- **WHEN** the state model loads it
- **THEN** it returns an error naming the file and the problem, and does not panic

### Requirement: onto status is read-only and config-independent

`onto status` SHALL be a read-only diagnostic command that inspects an existing
`docs/` workspace WITHOUT requiring a `homonto.toml` file or a declared
`[frameworks.onto]` entry. It SHALL enumerate change **directories** under
`docs/changes/` (excluding `archive/`) FIRST, then classify each as `valid`
(state present, parses, validates — report its derived phase), `malformed` (state
present but unparseable/invalid), or `missing-state` (a change directory with no
state file). A change directory whose state file was deleted SHALL therefore
appear as a `missing-state` row and SHALL NOT silently disappear. `onto status`
SHALL NOT create, modify, or delete any file.

#### Scenario: status classifies each change directory

- **GIVEN** `docs/changes/` with one valid change, one whose `onto-state.yaml` is malformed, and one directory with no state file
- **WHEN** `onto status` runs
- **THEN** it reports the first as `valid` with its phase, the second as `malformed`, and the third as `missing-state`, and exits without writing any file

#### Scenario: a deleted state file is not silently dropped

- **GIVEN** a change directory that once had `onto-state.yaml` but the file was deleted
- **WHEN** `onto status` runs
- **THEN** the directory is reported as `missing-state`, not omitted

#### Scenario: status leaves the worktree untouched

- **WHEN** `onto status` runs against any workspace
- **THEN** no file under `docs/` or elsewhere is created, modified, or removed

### Requirement: onto init scaffolds the workflow layout

`onto init` SHALL scaffold the onto workflow directory layout under the
workspace root: `docs/changes/`, `docs/specs/`, `docs/adr/`, and `docs/guides/`.
It SHALL be idempotent — an existing directory or file is preserved and never
overwritten — and it SHALL report which paths it created versus skipped. `onto
init` SHALL NOT create `homonto.toml` (that is `homonto init`'s job) and SHALL
NOT run the Homonto projection engine.

#### Scenario: init creates the docs layout in a prepared workspace

- **GIVEN** a workspace whose `homonto.toml` declares `[frameworks.onto]` and whose onto framework has been applied by Homonto
- **WHEN** `onto init` runs
- **THEN** `docs/changes/`, `docs/specs/`, `docs/adr/`, and `docs/guides/` exist and the command reports the created paths, exiting 0

#### Scenario: init is idempotent

- **GIVEN** a workspace where `onto init` already created the layout (and a user has added content under `docs/`)
- **WHEN** `onto init` runs again
- **THEN** existing directories and files are left untouched, newly-created paths (if any) are reported as created and pre-existing ones as skipped, and the command exits 0

### Requirement: onto init requires the Homonto-managed framework install

`onto init` is a mutating command and SHALL require that the project has declared
and applied `onto` through Homonto before it creates any `docs/` files:

- If `homonto.toml` is absent at the workspace root, `onto init` SHALL print a
  message directing the user to run `homonto init`, and exit non-zero.
- If `homonto.toml` exists but does not declare `[frameworks.onto]`, `onto init`
  SHALL print a message directing the user to declare `[frameworks.onto]` and run
  `homonto apply`, and exit non-zero.
- If `[frameworks.onto]` is declared but the onto framework has not been applied
  (no materialized evidence such as `.homonto/catalog/skills/onto/`), `onto init`
  SHALL print a message directing the user to run `homonto apply`, and exit
  non-zero.

In every failing case `onto init` SHALL NOT create, modify, or delete any file
under `docs/`.

#### Scenario: init refuses without homonto.toml

- **GIVEN** a workspace with no `homonto.toml`
- **WHEN** `onto init` runs
- **THEN** it prints guidance to run `homonto init`, creates no `docs/` files, and exits non-zero

#### Scenario: init refuses when frameworks.onto is not declared

- **GIVEN** a `homonto.toml` that does not declare `[frameworks.onto]`
- **WHEN** `onto init` runs
- **THEN** it prints guidance to declare `[frameworks.onto]` and run `homonto apply`, creates no `docs/` files, and exits non-zero

#### Scenario: init refuses when the framework is declared but not applied

- **GIVEN** a `homonto.toml` declaring `[frameworks.onto]` but no applied evidence (no `.homonto/catalog/skills/onto/`)
- **WHEN** `onto init` runs
- **THEN** it prints guidance to run `homonto apply`, creates no `docs/` files, and exits non-zero

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

`onto new <change-name> [--workflow full|fix|tweak]` SHALL create
`docs/changes/<change-name>/` containing an `onto-state.yaml` (`change` = the
name, `workflow` = the `--workflow` value defaulting to `full`, `phase` = `open`,
`created` = the current date) and empty-but-present `proposal.md` and `tasks.md`
skeleton files. `--workflow` SHALL accept only `full`, `fix`, or `tweak`; any
other value SHALL be rejected with a non-zero exit and no writes. It SHALL run the
framework-install gate first (same as `onto init`), SHALL validate `<change-name>`
is kebab-case with no path traversal (reject `..`, `/`, empty), and SHALL REFUSE
with a non-zero exit and NO writes if `docs/changes/<change-name>/` already exists.

#### Scenario: new creates the open-phase skeleton with the chosen workflow

- **GIVEN** a prepared workspace (framework-install gate passes) with no `docs/changes/feature-x/`
- **WHEN** `onto new feature-x --workflow fix` runs
- **THEN** `docs/changes/feature-x/onto-state.yaml` exists with `phase: open` and `workflow: fix`, alongside `proposal.md` and `tasks.md`, exiting 0

#### Scenario: new defaults workflow to full

- **WHEN** `onto new feature-y` runs with no `--workflow`
- **THEN** the created `onto-state.yaml` has `workflow: full`

#### Scenario: new rejects an invalid workflow

- **WHEN** `onto new feature-z --workflow epic` runs
- **THEN** it exits non-zero with a validation error and creates nothing

#### Scenario: new refuses to clobber an existing change

- **GIVEN** `docs/changes/feature-x/` already exists (with content)
- **WHEN** `onto new feature-x` runs
- **THEN** it exits non-zero, prints that the change already exists, and modifies no file under `docs/changes/feature-x/`

#### Scenario: new rejects an invalid change name

- **WHEN** `onto new "../evil"` (or a non-kebab-case / empty name) runs
- **THEN** it exits non-zero with a validation error and creates nothing

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

### Requirement: Per-phase required artifacts

`internal/ontostate.RequiredArtifacts(phase)` SHALL return the cumulative set of
files that must exist at each workflow phase: `open` → `onto-state.yaml`,
`proposal.md`, `tasks.md`; `design` → those plus `design.md`; `build` → those
plus `plan.md`; `verify` and `close` → those plus `verification.md`. An unknown
phase SHALL return at least the `open` base set. `ValidateSkeleton` SHALL use this
per-phase set so a change's skeleton requirements tighten as it advances.

#### Scenario: build phase requires plan.md

- **GIVEN** a change at phase `build` with `onto-state.yaml`, `proposal.md`, `tasks.md`, `design.md` but no `plan.md`
- **WHEN** `ValidateSkeleton` runs
- **THEN** it returns an error naming `plan.md` as missing

### Requirement: onto advance gates phase transitions

`onto advance <change>` SHALL move a change to the next phase in the fixed order
`open → design → build → verify → close`, and ONLY through that order (no skips,
no reversals). It SHALL run the framework-install gate first. Before advancing it
SHALL verify the transition's precondition, which is that the CURRENT phase's
deliverables are complete (a phase's artifacts are produced while a change is in
that phase, so they gate leaving it, not entering it):

- every artifact in `RequiredArtifacts(currentPhase)` exists (e.g. leaving
  `design` requires `design.md`; leaving `build` requires `plan.md`; leaving
  `verify` requires `verification.md`; leaving `open` requires only the open
  artifacts proposal.md + tasks.md), AND
- when leaving `build`, every `tasks.md` checkbox is checked (at least one
  checkbox present, no unchecked `- [ ]`).

On success it SHALL write the new phase to `onto-state.yaml` and report the
transition. On a failed precondition it SHALL exit non-zero, name what is
missing, and leave the recorded phase unchanged. Advancing a change already at
`close` (or with an unknown phase) SHALL be an error with no write.

#### Scenario: advance open to design needs only the open artifacts

- **GIVEN** a change at phase `open` with `proposal.md` and `tasks.md` (as `onto new` creates), and no `design.md` yet
- **WHEN** `onto advance <change>` runs
- **THEN** the recorded phase becomes `design` and the command reports `open → design`, exiting 0

#### Scenario: advance refuses when the current phase's deliverable is missing

- **GIVEN** a change at phase `design` that has not yet produced `design.md`
- **WHEN** `onto advance <change>` runs
- **THEN** it exits non-zero naming `design.md` as missing and the recorded phase stays `design`

#### Scenario: advance out of build requires all tasks checked

- **GIVEN** a change at phase `build` with `plan.md` present but an unchecked `- [ ]` item in `tasks.md`
- **WHEN** `onto advance <change>` runs
- **THEN** it exits non-zero indicating tasks are incomplete and the recorded phase stays `build`

#### Scenario: advance past close is an error

- **GIVEN** a change at phase `close`
- **WHEN** `onto advance <change>` runs
- **THEN** it exits non-zero indicating the change is already at the terminal phase and writes nothing

### Requirement: dirty worktree blocks the close transition

`onto advance` SHALL check the workspace's git worktree cleanliness (via `git
status --porcelain`). A dirty worktree SHALL produce a WARNING for a normal
transition (open→design, design→build, build→verify) but SHALL still allow it.
For the release-critical `verify → close` transition a dirty worktree SHALL BLOCK
the advance: the command exits non-zero, reports the dirty worktree, and does not
change the phase.

#### Scenario: dirty worktree warns but allows a normal advance

- **GIVEN** a change at phase `open` (with the open artifacts proposal.md + tasks.md) in a workspace with uncommitted changes
- **WHEN** `onto advance <change>` runs
- **THEN** it advances to `design` (exit 0) after printing a dirty-worktree warning

#### Scenario: dirty worktree blocks verify to close

- **GIVEN** a change at phase `verify` whose `verification.md` (and earlier deliverables) exist, in a workspace with uncommitted changes
- **WHEN** `onto advance <change>` runs
- **THEN** it exits non-zero reporting the dirty worktree and the recorded phase stays `verify`

### Requirement: Dependency resolution

`internal/ontostate.DepsResolved(root string, deps []string) []string` SHALL
return the subset of `deps` that are NOT resolved. A dependency `<dep>` is
resolved iff an archived change directory matching
`docs/changes/archive/*-<dep>` exists under `root`. An empty or nil `deps` SHALL
yield no unresolved dependencies (nil and empty slice are equivalent — both mean
"no dependencies").

#### Scenario: resolved and unresolved dependencies are distinguished

- **GIVEN** a workspace where `docs/changes/archive/2026-07-10-a/` exists but there is no archived `b`
- **WHEN** `DepsResolved(root, ["a","b"])` is called
- **THEN** it returns `["b"]` (a is resolved, b is not)

#### Scenario: no dependencies is always resolved

- **WHEN** `DepsResolved(root, nil)` or `DepsResolved(root, [])` is called
- **THEN** it returns an empty list

### Requirement: onto close archives a completed change

`onto close <change>` SHALL archive a completed change. It SHALL run the
framework-install gate, validate the change name, and require ALL of the
following before archiving (each failing case exits non-zero and archives
NOTHING):

- the change's recorded phase is `close` (a change not yet at `close` is
  rejected with guidance to run `onto advance`);
- every dependency listed in the change's `onto-state.yaml` is resolved
  (`DepsResolved` returns empty); otherwise it names the unresolved dependencies;
- the git worktree is clean (a dirty OR undeterminable worktree blocks the
  archive — this is a release-critical operation).

On success it SHALL set `archived: true` in the change's `onto-state.yaml`, then
move `docs/changes/<change>/` to `docs/changes/archive/<YYYY-MM-DD>-<change>/`
(creating the archive directory if needed), and report the archived path. If the
archive target directory already exists it SHALL refuse (non-zero) and move
nothing.

If the archive move itself fails after `archived: true` was written, `onto
close` SHALL roll the `archived` flag back to `false` (re-saving the in-place
`onto-state.yaml`) and exit non-zero, so a failed archive leaves the change
fully un-archived — never marked archived while still at its original path.

#### Scenario: close archives a close-phase change with resolved deps and a clean worktree

- **GIVEN** a change at phase `close` with no unresolved deps in a clean git worktree
- **WHEN** `onto close <change>` runs
- **THEN** `docs/changes/<change>/` is moved to `docs/changes/archive/<date>-<change>/`, its `onto-state.yaml` has `archived: true`, and the command reports the archived path, exiting 0

#### Scenario: a failed archive move leaves the change un-archived

- **GIVEN** a change at phase `close` that passes every archive precondition
- **WHEN** `onto close <change>` runs but the move into the archive directory fails
- **THEN** the command exits non-zero, the change directory remains at its original path, and its `onto-state.yaml` has `archived: false` (the flag was rolled back)

#### Scenario: close refuses a change not at the close phase

- **GIVEN** a change at phase `build`
- **WHEN** `onto close <change>` runs
- **THEN** it exits non-zero, reports the change is not at `close`, and archives nothing

### Requirement: onto doctor reports workflow and project health

`onto doctor [--dir <root>]` SHALL be a strictly read-only, config-independent
diagnostic that reports the health of an onto workspace. It SHALL perform zero
writes, never construct a homonto config/engine, and never read `homonto.toml`.
It SHALL run regardless of whether the onto framework is installed. `--dir` SHALL
default to `.`.

`onto doctor` SHALL check, and surface each problem it finds as an individual
finding line:

- **docs layout**: `docs/changes`, `docs/specs`, `docs/adr`, and `docs/guides`
  each exist as directories under the root; a missing one is a finding.
- **active change classification**: it SHALL enumerate change **directories**
  under `docs/changes/` (the single `*` excludes archived changes) FIRST, then
  classify each as `valid` (state loads, validates, derives a phase), `malformed`
  (state present but unparseable/invalid), or `missing-state` (a change directory
  with no state file). A `malformed` or `missing-state` directory is a finding —
  a change whose state file was deleted SHALL be reported, not silently skipped.
- **phase matches artifacts**: for each valid active change, every artifact
  required for its derived phase is present; a missing required artifact is a
  finding.
- **dependency and gate consistency**: for each valid active change, every
  dependency it lists is resolved (an archived `docs/changes/archive/*-<dep>`
  exists); an unresolved dependency is a finding. An active change whose state
  records `archived: true` is a finding.
- **archive layout**: each `docs/changes/archive/*` entry is a directory holding
  a valid `onto-state.yaml` marked `archived: true`; a missing or invalid state
  file, or one not marked archived, is a finding.

On a healthy workspace `onto doctor` SHALL print a single `healthy` line and
exit 0. When one or more findings exist it SHALL print each finding and a count
summary and exit non-zero.

#### Scenario: healthy workspace reports healthy and exits 0

- **GIVEN** a root with the full `docs/{changes,specs,adr,guides}` layout, a valid active change whose artifacts match its phase and whose deps are resolved, and a well-formed archive entry
- **WHEN** `onto doctor` runs
- **THEN** it prints `healthy` and exits 0

#### Scenario: a missing-state change directory is a finding

- **GIVEN** a change directory under `docs/changes/` that has no state file (e.g. `onto-state.yaml` was deleted)
- **WHEN** `onto doctor` runs
- **THEN** it classifies the directory as `missing-state`, reports it as a finding, and exits non-zero — the directory is not silently skipped

#### Scenario: invalid state is a finding

- **GIVEN** an active change whose state is malformed or fails validation
- **WHEN** `onto doctor` runs
- **THEN** it reports the change as invalid, naming the problem, and exits non-zero

#### Scenario: onto doctor is read-only and needs no framework install

- **GIVEN** a root with no `homonto.toml` and no installed onto framework
- **WHEN** `onto doctor` runs
- **THEN** it still runs (reporting docs-layout findings) and writes nothing

### Requirement: Release packaging ships both binaries

The release pipeline SHALL cross-compile, version-stamp, checksum, and publish
**both** the `homonto` and `onto` binaries for every supported target. A shared,
locally-runnable build script `scripts/build-release.sh <version>` SHALL be the
single source of the packaging logic, invoked by the release workflow so the
same code path runs on and off CI.

For each of the six targets (`linux/amd64`, `linux/arm64`, `darwin/amd64`,
`darwin/arm64`, `windows/amd64`, `windows/arm64`) the script SHALL produce a
**separate archive per binary**:

- `homonto_<version>_<os>_<arch>` containing the `homonto` binary plus `LICENSE`
  and `README.md`;
- `onto_<version>_<os>_<arch>` containing the `onto` binary plus `LICENSE` and
  `README.md`.

Windows archives SHALL be `.zip` and carry the `.exe` suffix on the binary;
other targets SHALL be `.tar.gz`. Each binary SHALL be built with
`CGO_ENABLED=0`, `-trimpath`, and `-ldflags "-s -w -X <pkg>.Version=<version>"`
where `<pkg>` is `github.com/noviopenworks/homonto/internal/cli` for `homonto`
and `github.com/noviopenworks/homonto/internal/ontocli` for `onto`. A single
`SHA256SUMS` file SHALL cover every produced archive (12 in total).

#### Scenario: release build produces both binaries' archives for every target

- **GIVEN** the repository at a clean checkout and a version string
- **WHEN** `scripts/build-release.sh <version>` runs
- **THEN** `dist/` contains a `homonto_<version>_<os>_<arch>` archive and an `onto_<version>_<os>_<arch>` archive for each of the six targets (12 archives), and a `SHA256SUMS` listing all of them

#### Scenario: each binary carries its own stamped version

- **WHEN** the release build stamps the binaries
- **THEN** the `homonto` binary reports `<version>` via `homonto version` and the `onto` binary reports `<version>` via `onto version`, each stamped through its own package's `Version` ldflag

#### Scenario: windows archives are zips with .exe binaries

- **WHEN** the release build targets `windows/amd64` or `windows/arm64`
- **THEN** the produced archives are `.zip` files and the binary inside is named `homonto.exe` / `onto.exe`

#### Scenario: CI smoke covers the onto version stamp

- **GIVEN** the CI workflow
- **WHEN** it runs the version-stamp smoke checks
- **THEN** it stamps and runs `onto version` (in addition to `homonto version`) and fails if the stamped version is not reported

### Requirement: onto exposes state transitions and a structured read

The `onto` binary SHALL expose, through its CLI, a command for every gated state
mutation of an active change and a structured read of a change's full state, so a
caller can drive the entire workflow lifecycle without editing a state file by
hand. This SHALL include setters for isolation, build mode, tdd mode, verify scale,
verify result, close merged, directive, base ref, deps, and guides. Each
transition command SHALL validate the presence and shape of the field it sets
(rejecting a malformed value with a clear error) and SHALL write through the
versioned state model. The structured read SHALL emit the full validated state
(including derived phase) as JSON.

#### Scenario: a transition command sets a gated field with validation

- **GIVEN** an active change at a phase where a gated field (e.g. isolation) may be set
- **WHEN** the corresponding `onto` transition command runs with a valid value
- **THEN** the field is written through the state model and a subsequent read reflects it
- **AND** running it with a value outside the field's allowed shape is rejected with a clear error and no write

#### Scenario: base-ref and deps setters record creation fields

- **GIVEN** an active change
- **WHEN** `onto set base-ref <change> <ref>` and `onto set deps <change> --dep a --dep b` run
- **THEN** the state records the base ref and the dependency list, reflected in a subsequent read

#### Scenario: structured read emits the full state as JSON

- **GIVEN** a change with a valid `onto-state.yaml`
- **WHEN** the `onto` structured read command runs for that change
- **THEN** it emits the full validated state and derived phase as JSON, writing no file

### Requirement: onto ships a full-lifecycle conformance suite

onto SHALL carry a full-lifecycle conformance test suite that drives the real CLI
through `new → set → advance → close` and asserts the binary gates REJECT bad
work, not only that the happy path succeeds. The suite SHALL cover at least:
advancing without a required phase artifact is refused; an invalid `--workflow`
is refused with no change created; a gated-field setter with an out-of-shape value
writes nothing; and `status`/`doctor` classify a malformed or missing state rather
than silently dropping it. This substitutes for the dogfooding feedback loop onto
forgoes (the project builds with Comet and ships onto — see `docs/personas.md`).

#### Scenario: the conformance suite proves gates reject bad work

- **GIVEN** the onto conformance suite
- **WHEN** it runs against the onto CLI
- **THEN** it passes only if each gate rejects its bad-work case (missing artifact, invalid workflow, out-of-shape enum, malformed/missing state) and the happy-path lifecycle still succeeds

### Requirement: onto close requires close-phase evidence

`onto close` SHALL, in addition to its existing phase, dependency, and
clean-worktree gates, require the close-phase evidence tokens that the workflow
produces, and SHALL refuse to archive (with a clear error naming the missing
evidence) when they are absent. For a `full` workflow it SHALL require
`verify.result == pass`, `close.merged == true`, and `guides` resolved (`updated`
or a `waived:<reason>`, never `pending` or empty). For a `fix` or `tweak` preset it
SHALL require the reduced set those presets produce — `verify.result == pass` and
`close.merged == true` — and SHALL NOT require `guides`. This makes archiving gate
on real evidence (B1: the token is present and well-formed), not merely on file
existence and checked boxes.

#### Scenario: full close is refused without a passing verification

- **GIVEN** a `full` change at the close phase whose `verify.result` is `pending` (or `fail`)
- **WHEN** `onto close` runs
- **THEN** it refuses with an error naming the missing passing verification, and archives nothing

#### Scenario: full close is refused without resolved guides

- **GIVEN** a `full` change at close with `verify.result == pass` and `close.merged == true` but `guides` still `pending`
- **WHEN** `onto close` runs
- **THEN** it refuses naming the unresolved guides, and archives nothing

#### Scenario: a tweak close does not require guides

- **GIVEN** a `tweak` change at close with `verify.result == pass` and `close.merged == true` and `guides` unset
- **WHEN** `onto close` runs
- **THEN** the reduced preset gate is satisfied and (with deps + clean worktree) the change archives

### Requirement: onto advance gates on phase evidence

`onto advance` SHALL require the evidence a phase must have before leaving or
entering it, beyond artifact existence and checked tasks. Leaving `verify` SHALL
require `verify.result == pass` (never `pending` or `fail`). Entering `build` SHALL
require `isolation` chosen (`branch` or `worktree`), so planning work is never
committed unisolated. A missing token SHALL block the transition with a clear
error, writing nothing.

#### Scenario: leaving verify requires a passing result

- **GIVEN** a change at `verify` whose `verify.result` is `pending`
- **WHEN** `onto advance` runs
- **THEN** it refuses naming the missing passing verification and leaves the phase unchanged

#### Scenario: entering build requires isolation

- **GIVEN** a change ready to advance from `design` to `build` with no `isolation` set
- **WHEN** `onto advance` runs
- **THEN** it refuses naming the missing isolation and leaves the phase at `design`
