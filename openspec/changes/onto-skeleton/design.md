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
