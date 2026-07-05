# Delta Spec: onto-workflow (address-deep-review)

## MODIFIED Requirements

### Requirement: Required tooling preflight

The dispatcher SHALL check that `rtk` is installed (all shell operations
then go through rtk) and that graphify is available (open/design phases
ground codebase understanding in graphify/codegraph queries when it is),
and SHALL warn and proceed — never halt — when either is missing: a
missing rtk produces a warning that token costs will be higher; missing
graphify with no existing index produces a warning and records
`grounding: direct file reading (graphify unavailable)` in the change's
notes.md Grounding section. Indexing SHALL remain the user's decision, and
a stale index gets the same ask-or-proceed treatment, never a halt.

#### Scenario: Missing rtk

- **GIVEN** a machine without rtk on PATH
- **WHEN** `/onto` is invoked
- **THEN** the workflow warns that token costs will be higher, recommends
  installing rtk, and proceeds

#### Scenario: Missing graphify

- **GIVEN** a machine with neither the graphify skill nor an existing
  index (`graphify-out/`, `.codegraph/`)
- **WHEN** `/onto` is invoked
- **THEN** the workflow warns, records the direct-file-reading grounding
  fallback in notes.md, and proceeds

### Requirement: Preset paths with upgrade rules

The workflow SHALL provide preset paths — `/onto-fix` (bug fixes; failing
test reproducing the bug required first) and `/onto-tweak`
(copy/config/docs/prompt changes, plus small features within tweak limits:
≤5 files with test files excluded, no new capability, no existing-spec
requirement change) — that skip the design phase but keep verify and
close. The workflow SHALL force an upgrade confirmation
to the full path when scope grows: fix — 3+ files, architecture/schema
changes, new public API; tweak — 5+ files, cross-module coordination, 5+ new
tests, config key additions/removals, new capability, or spec-affecting
changes.

#### Scenario: Upgrade trigger

- **GIVEN** an active `/onto-fix` change
- **WHEN** the fix grows to touch four files
- **THEN** the skill pauses, explains the trigger, and asks the user to
  confirm upgrading to the full workflow before continuing

#### Scenario: Small feature stays a tweak

- **GIVEN** a small feature request touching 2 non-test files, adding no
  new capability, and changing no existing spec requirement
- **WHEN** the work is routed
- **THEN** it runs as `/onto-tweak` without the full workflow's design
  phase

### Requirement: Close phase with documentation obligation

The close phase SHALL lint the change before merging (delta-spec format:
only ADDED/MODIFIED/REMOVED/RENAMED sections, SHALL/MUST in every
ADDED/MODIFIED requirement's first line, GIVEN/WHEN/THEN scenarios;
state.yaml validity; `Result:` line present; ADR draft fields; post-merge
no delta-only section headings; dangling-reference audit) with findings
blocking archive exactly like the guides obligation; then merge spec deltas into `docs/specs/`
(including RENAMED semantics), assign final numbers to ADR drafts and move
them to `docs/adr/` with status Accepted, rewrite the workspace's
references to `adr/<slug>.md` in `design.md` and `notes.md` to the final
`docs/adr/NNNN-<slug>.md` paths so the archive ships no dangling ADR
references, write or update `docs/guides/`
(or record an explicit `guides: "waived: <reason>"`), finalize
`metrics` (phase dates, tasks_total, verify_rounds, upgraded), archive the
workspace to `docs/changes/archive/YYYY-MM-DD-<name>/`, set
`archived: true` — after final user confirmation — and offer a ship
handoff: a ready PR body (proposal why/what + verification summary +
evidence pointers) written to the archive's `ship.md` when accepted, with
PR creation remaining outside the workflow.

#### Scenario: Guides not updated

- **GIVEN** a change reaching close with `guides: pending`
- **WHEN** the agent attempts to archive
- **THEN** close is blocked until guides are updated or a waiver reason is
  recorded

#### Scenario: Malformed delta caught at lint

- **GIVEN** a delta spec whose requirement lacks SHALL in its first line
- **WHEN** close runs the lint checklist
- **THEN** the finding blocks the merge until the delta is fixed

#### Scenario: RENAMED requirement merged

- **GIVEN** a delta with `## RENAMED Requirements` mapping FROM/TO names
- **WHEN** close merges the delta
- **THEN** the living spec's requirement heading is renamed with its body
  preserved unless a MODIFIED block also targets the new name

#### Scenario: ADR references rewritten before archive

- **GIVEN** a workspace design.md referencing `adr/some-decision.md` and a
  close phase that numbered that draft to `docs/adr/0009-some-decision.md`
- **WHEN** close completes and the workspace is archived
- **THEN** the archived design.md references `docs/adr/0009-some-decision.md`
  and no reference to the moved draft path remains
