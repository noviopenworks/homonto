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
