## Context

The dual-binary release needs an `onto` binary beside `homonto`. Today only
`homonto` exists (`main.go` → `internal/cli.NewRootCmd`). This change lands the
foundation of the `onto-binary` capability: the second binary, its CLI root, the
`onto-state.yaml` model, and the read-only `onto status`. Mutating commands
(`init`, phase gates, `doctor`) and release packaging are separate later changes
(#2–#5 in the proposal). `onto` is a product feature (it operates the onto
workflow for homonto's users); it is independent of how this repo is developed
(comet) and of `homonto`'s config-projection pipeline.

## Goals / Non-Goals

**Goals**

- `go build ./cmd/onto` produces an `onto` binary; `go build ./...` builds both;
  root `homonto` untouched.
- `onto` Cobra root + `version` (ldflags-stampable), mirroring `internal/cli`.
- `internal/ontostate`: parse/validate `onto-state.yaml`, derive phase.
- `onto status`: read-only, config-independent workspace inspection.

**Non-Goals**

- `onto init` / docs scaffolding (change #2), phase-gate enforcement /
  skeleton create-validate (change #3), `onto doctor` (change #4), dual-binary
  release packaging (change #5).
- Any change to `homonto`'s behavior, `internal/cli`, adapters, engine, config,
  or the catalog.
- Migration from the legacy `state.yaml` name.

## Decisions

**D1 — Binary layout: `cmd/onto/main.go` + `internal/ontocli`.** Add a second
`package main` under `cmd/onto/` (Go's conventional multi-binary layout) whose
`main` calls `ontocli.NewRootCmd().Execute()`, exactly as root `main.go` calls
`cli.NewRootCmd()`. A new `internal/ontocli` package holds the `onto` root,
`Version` var, and subcommands, kept separate from `internal/cli` (the `homonto`
CLI) so the two binaries share nothing that could couple their flags/version.
Alternative (one package main with a mode flag) rejected — the design mandates
two distinct binaries.

**D2 — YAML parsing for `onto-state.yaml` (key dependency decision).** The repo
currently has NO YAML dependency (config is TOML via `go-toml/v2`); the state
file is mandated as `onto-state.yaml`. Decision: add `gopkg.in/yaml.v3` as the
single new dependency for the `onto` state model. Rationale: it is the de-facto
standard, well-audited, and covered by `govulncheck` in CI; hand-rolling a YAML
parser for a structured state file (phase + gate/dependency records to come in
#3) is error-prone. This is called out explicitly because the project prizes a
tiny dependency set — the trade-off is accepted for a real YAML file. (The Design
Doc will confirm the exact struct tags and the minimal field set.)

**D3 — Minimal foundation state schema; derive = validated read.** For this
increment `onto-state.yaml` carries at least a change identifier and a `phase`
field from the onto workflow phase set open → design → build → verify → close
(terminal `close`, matching the `onto-*` skills and legacy `state.yaml`; NOT the
comet dev terminal `archive`). "Derive phase" validates
the recorded phase is a known value and returns it. Full artifact-based phase
derivation and gate consistency are change #3/#4 concerns; the model is
structured so those extend it without a rewrite.

**D4 — `onto status` is strictly read-only and degraded-safe.** It discovers
changes by walking `docs/changes/*/onto-state.yaml`, loads each via
`internal/ontostate`, and prints a per-change phase line; unreadable/malformed
state is reported as invalid, not fatal to the whole run. It never constructs the
homonto config/engine and never writes — satisfying the design's "read-only
degraded exception" that works without `homonto.toml` or `[frameworks.onto]`.

## Risks / Trade-offs

- **New YAML dependency** → Mitigation: single well-known module, `govulncheck`
  in CI, confined to the `onto` state model.
- **onto-state.yaml schema churn across #2–#4** → Mitigation: keep the
  foundation schema minimal and additive; later changes add fields (gates,
  deps), not rewrite existing ones.
- **CLI duplication between `internal/cli` and `internal/ontocli`** → Accepted:
  the two binaries are intentionally decoupled; shared helpers can be extracted
  later only if a real need appears.

## Migration Plan

Purely additive: new `cmd/onto/`, `internal/ontocli`, `internal/ontostate`, and
one new go.mod dependency. No existing binary or package changes behavior.
Rollback is removing the new packages and the dependency.

## Open Questions

- Exact `onto-state.yaml` field set and struct tags (confirmed in the Design
  phase against `docs/superpowers/specs/2026-07-09-dual-binary-release-design.md`
  and the legacy `state.yaml` shape for reference only).
- Whether `onto status` output should be plain lines or also offer `--json`
  (default: plain for the foundation; `--json` can come with `doctor` in #4).
