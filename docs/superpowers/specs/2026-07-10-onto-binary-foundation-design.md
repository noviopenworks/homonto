---
comet_change: onto-binary-foundation
role: technical-design
canonical_spec: openspec
---

# Onto Binary Foundation — Technical Design

Deep refinement of the open-phase `design.md` for `onto-binary-foundation`
(change #1 of the five-change `onto` binary decomposition). Delivers: the second
binary, its CLI root, the `onto-state.yaml` model, and read-only `onto status`.

## Context

Only `homonto` exists in source (`main.go` → `internal/cli.NewRootCmd`, Cobra;
`Version` stamped via `-ldflags -X …/internal/cli.Version=`). The repo's only
serialization dep is `go-toml/v2`; there is no YAML dependency. `onto` is a
product binary operating the onto workflow (phases open → design → build →
verify → close, per the `onto-*` skills and legacy `state.yaml`); it is
independent of `homonto`'s projection pipeline and of comet (the repo's own dev
workflow).

## Goals / Non-Goals

**Goals:** `cmd/onto` builds an `onto` binary; `internal/ontocli` root +
`version`; `internal/ontostate` parse/validate/derive-phase for
`onto-state.yaml`; read-only, config-independent `onto status`.

**Non-Goals:** `onto init`/scaffolding (#2), phase-gate enforcement (#3),
`onto doctor` (#4), release packaging (#5); any change to `homonto`,
`internal/cli`, adapters, engine, config, catalog; legacy `state.yaml`
migration; artifact-derived phase (only recorded-phase validation here).

## Decisions

### D1 — `cmd/onto/main.go` + `internal/ontocli`

```
cmd/onto/main.go            package main → ontocli.NewRootCmd().Execute()
internal/ontocli/root.go    Version var + NewRootCmd() (Use:"onto") + version + status
internal/ontostate/state.go State, Parse/Load, Validate, DerivePhase
```

`cmd/onto/main.go` mirrors root `main.go` verbatim in structure (Execute, print
error to stderr, `os.Exit(1)`). `internal/ontocli` mirrors `internal/cli`'s
`NewRootCmd`/`Version` pattern but is a separate package so the two binaries do
not couple flags or version. No shared code is extracted in this increment.

### D2 — `gopkg.in/yaml.v3` for `onto-state.yaml`

Add the single dependency `gopkg.in/yaml.v3`. `internal/ontostate` is the only
package importing it. `go.mod`/`go.sum` gain it; CI `govulncheck` already scans
`./...`. Hand-rolling YAML is rejected (the schema grows gate/dependency records
in #3). `go mod tidy` must be clean after adding it.

### D3 — `onto-state.yaml` foundation schema

Minimal, additive, informed by the legacy `state.yaml` but under the new name:

```go
type State struct {
    Change   string `yaml:"change"`            // change id/name
    Workflow string `yaml:"workflow,omitempty"` // full|tweak|hotfix (informational here)
    Phase    string `yaml:"phase"`             // open|design|build|verify|close
    Created  string `yaml:"created,omitempty"`
    BaseRef  string `yaml:"base_ref,omitempty"`
    Deps     []string `yaml:"deps,omitempty"`
    Archived bool   `yaml:"archived,omitempty"`
}
```

Only `change` and `phase` are load-bearing this increment; the rest are parsed
(so a real legacy-shaped file round-trips) but unused until #3/#4. Unknown YAML
keys are ignored (yaml.v3 default) — forward-compatible with fields #3 adds.

- `Parse(b []byte) (State, error)`: `yaml.Unmarshal`; wrap errors as
  `onto-state: <detail>`.
- `Load(path string) (State, error)`: read file (wrap os error naming the path)
  → `Parse`.
- `Validate() error`: `Change` non-empty; `Phase` ∈ the fixed set; else a clear
  error. No panics on any input.
- `DerivePhase() (string, error)`: `Validate` then return `Phase`. (Artifact-
  derived phase is a #4/doctor concern; recorded phase is authoritative here.)

### D4 — `onto status`: read-only, config-independent

`statusCmd()` resolves the workspace root (CWD or a `--dir` flag defaulting to
`.`), globs `docs/changes/*/onto-state.yaml` (skipping `docs/changes/archive/`
for active changes; archived ones may be listed as archived), and for each:
loads via `internal/ontostate`, prints `"<change>: <phase>"`, or
`"<change>: invalid (<reason>)"` when load/validate fails — never aborting the
whole run on one bad file. It constructs NO homonto config/engine, reads no
`homonto.toml`, and performs zero writes. Exit 0 on a clean read even if some
changes are invalid (invalid state is reported, not a process failure — this is
the diagnostic/recovery command). Output is plain lines; `--json` is deferred to
`doctor` (#4).

## Component Boundaries

| Unit | Responsibility | Depends on |
|---|---|---|
| `cmd/onto` | process entry | `internal/ontocli` |
| `internal/ontocli` | onto CLI root, version, status | `internal/ontostate`, cobra |
| `internal/ontostate` | onto-state.yaml parse/validate/derive | `gopkg.in/yaml.v3` |

## Risks / Trade-offs

- **New YAML dep** → single audited module, confined to `internal/ontostate`,
  `govulncheck`-covered.
- **Schema churn in #2–#4** → foundation fields are additive; later changes add,
  never rewrite. Unknown-key tolerance keeps old/new files interoperable.
- **CLI duplication** (`internal/cli` vs `internal/ontocli`) → accepted;
  intentional decoupling of the two binaries.
- **`onto status` exit code semantics** → invalid state files are reported but do
  not fail the process (diagnostic intent); a genuinely unreadable workspace
  root (e.g. no `docs/`) prints an empty/"no changes" result at exit 0.

## Testing Strategy

1. `internal/ontostate`: valid parse+derive; malformed YAML error names the file;
   unknown-phase and empty-change validate errors; missing-file load error;
   no-panic on garbage input.
2. `internal/ontocli`: `onto version` prints `onto <Version>`; command wiring.
3. `onto status`: temp `docs/changes/` with a valid change (phase line) and an
   invalid one (flagged), asserting exit 0, correct lines, and — via a
   before/after file-tree snapshot — that NO file was created/modified/removed;
   status runs with no `homonto.toml` present.
4. Build: `go build ./cmd/onto` and `go build ./...`; regression
   `go test [-race] ./...`, `go vet`, `gofmt -l .`, `go mod tidy` clean.

## Open Questions

None blocking. `--json` output and artifact-derived phase are deferred to #4 by
design.
