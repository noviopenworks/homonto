---
comet_change: onto-skeleton
role: technical-design
canonical_spec: openspec
archived-with: 2026-07-10-onto-skeleton
status: final
---

# Onto Skeleton — Technical Design

Refinement of `design.md` for `onto-skeleton` (onto binary #3a — the first
sub-increment of the onto workflow engine). Adds the `onto-state.yaml` writer,
`onto new <change>` (skeleton creation), and phase-aware skeleton validation
surfaced through `onto status`.

## Context

onto #1 (foundation) and #2 (`onto init`) are archived. The `onto-state.yaml`
model is read-only today; creating a change needs a writer. Per the dual-binary
design the binary "creates and validates skeletons" while skills fill content.
`onto` remains isolated from homonto's projection pipeline.

## Goals / Non-Goals

**Goals:** `ontostate.Marshal`/`Save` (round-trips `Parse`); `onto new` (gated,
name-validated, no-clobber) creating the open-phase skeleton; `RequiredArtifacts`
+ `ValidateSkeleton`; `onto status` skeleton note (read-only).

**Non-Goals:** phase transitions (#3b), deps/archive/close (#3c), non-empty
skeleton content, isolation changes.

## Decisions

**D1 — `ontostate` writer, self-contained atomic write.** `Marshal(State)
([]byte,error)` = `yaml.Marshal(s)`. `Save(path string, s State) error`:
`os.MkdirAll(filepath.Dir(path),0o755)`, write `path+".tmp-"+strconv/rand` via
`os.WriteFile(...,0o644)`, `os.Rename(tmp,path)`; on any error remove the temp.
No `internal/fsutil` import (keep `onto` self-contained; it is ~8 lines).
Round-trip invariant: `Parse(Marshal(s))` deep-equals `s`.

**D2 — `onto new <change>`: gate → validate name → no-clobber → create.**
`newCmd()` takes a positional `<change-name>` arg and `--dir` (default ".",
matching init/status). Order:
1. `gate(root)` (reuse the function from `init.go`); on error return it (guidance
   to the user via `cmd/onto/main.go`), write nothing.
2. Validate the name with a LOCAL validator `validChangeName(name) error` in
   `internal/ontocli`: non-empty, `name == filepath.Base(name)`, no `..`, matches
   `^[a-z0-9]+(-[a-z0-9]+)*$`. (NOT homonto's `config.validateResourceName` —
   isolation.)
3. If `docs/changes/<name>/` exists → return a non-zero "change %q already exists"
   error, write nothing.
4. `os.MkdirAll(docs/changes/<name>)`; build `State{Change:name, Workflow:"full",
   Phase:"open", Created: time.Now().Format("2006-01-02")}` and
   `ontostate.Save(<dir>/onto-state.yaml, state)`; `os.WriteFile` empty
   `proposal.md` and `tasks.md` (only if absent). Print "created change %q at
   <dir>" listing the files, exit 0.

`time.Now()` is fine in the Go binary — the Date.now prohibition is a comet-JS
constraint, not Go. Tests assert `created` matches `^\d{4}-\d{2}-\d{2}$`, not a
fixed value.

**D3 — `RequiredArtifacts` + `ValidateSkeleton` in `ontostate`, surfaced by
status.** `RequiredArtifacts(phase string) []string`: a map, `open` →
`["onto-state.yaml","proposal.md","tasks.md"]`; unknown/other phases return the
same base set for now (per-phase supersets are #3b's concern). `ValidateSkeleton
(changeDir string) error`: `Load(changeDir/onto-state.yaml)` → `DerivePhase` →
for each `RequiredArtifacts(phase)` `os.Stat`; return an error naming the first
missing file (or nil). `onto status` calls it per discovered change and appends
`" — skeleton ok"` or `" — skeleton: missing <file>"` to that change's line;
still read-only, one bad change never aborts the run.

## Component Boundaries

| Unit | Responsibility | Depends on |
|---|---|---|
| `internal/ontostate` | Marshal/Save, RequiredArtifacts, ValidateSkeleton | yaml.v3, os |
| `internal/ontocli` new.go | `onto new` (gate+validate+create) | ontostate, cobra |
| `internal/ontocli` status.go | skeleton note per change | ontostate |

`onto` imports none of homonto's `internal/{cli,engine,config,adapter,catalog}`.

## Risks / Trade-offs

- **`created` non-determinism** → assert format not value.
- **Coarse RequiredArtifacts** → open-phase-first; #3b refines. Documented.
- **No-clobber race** → irrelevant for a single-shot CLI; prevents the common
  overwrite.
- **Reusing `gate` across new/init** → intended; both are mutating onto commands
  with the same precondition.

## Testing Strategy

1. ontostate: Marshal/Parse round-trip; Save→Load round-trip + parent-dir
   creation; RequiredArtifacts(open); ValidateSkeleton ok + missing-file-names.
2. `onto new`: creates skeleton (state phase open + proposal + tasks), exit 0;
   refuses existing change (pre-placed file untouched); rejects `../evil` and
   other invalid names; gate-failure → guidance, no writes; `created` format.
3. status: skeleton-ok + missing-artifact notes; read-only tree snapshot holds.
4. Isolation grep; both binaries build; `go test [-race] ./...`, vet, gofmt,
   tidy.

## Open Questions

None blocking. Per-phase required-artifact supersets → #3b.
