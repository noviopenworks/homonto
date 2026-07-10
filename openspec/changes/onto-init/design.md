## Context

`onto-binary-foundation` (#1, archived) shipped the `onto` binary,
`internal/ontostate`, and read-only `onto status`. `onto init` (#2) is the first
mutating command: it scaffolds `docs/{changes,specs,adr,guides}` but only after
the project has declared and applied `[frameworks.onto]` through Homonto (the
dual-binary design's "managed, not an alternate installer" rule).

## Goals / Non-Goals

**Goals:** `onto init` scaffolds the four docs dirs idempotently; a
framework-install gate refuses (non-zero, no writes) when homonto.toml is absent,
lacks `[frameworks.onto]`, or the framework is unapplied; clear guidance messages.

**Non-Goals:** phase-gate enforcement / skeleton create-validate (#3), `onto
doctor` (#4), release packaging (#5); creating `homonto.toml` (that is `homonto
init`); running the projection engine; changing `onto status` or the state model.

## Decisions

**D1 — `internal/ontocli/init.go` with an explicit gate then scaffold.** `onto
init` runs the gate first; only if it passes does it scaffold. Registered on the
onto root alongside `version`/`status`.

**D2 — Gate via a lightweight homonto.toml read + a filesystem applied-check,
NOT the full config engine.** To keep `onto` decoupled from homonto's projection
pipeline (per #1's isolation) and avoid failing on unrelated config-validation
errors, the gate reads `homonto.toml` directly with `go-toml/v2` and checks only
for a `[frameworks.onto]` table (a `[frameworks]` map containing key `onto`). It
does NOT call `internal/config.Load` (which pulls the catalog and validates
models/all resources) and does NOT construct the engine/adapters. Applied
evidence is a filesystem check: `.homonto/catalog/skills/onto/` exists next to
`homonto.toml` (the materialized onto framework). Rationale: minimal coupling,
fast, and robust to unrelated config problems; the check is intentionally
structural, not a full config validation. Alternative (`config.Load`) rejected —
too much coupling and it would reject a config with any unrelated validation
error, blocking a legitimate `onto init`.

Gate order (first failure wins, each exits non-zero with specific guidance, no
writes):
1. `homonto.toml` missing → "run `homonto init`".
2. `homonto.toml` present but no `[frameworks.onto]` → "declare `[frameworks.onto]` and run `homonto apply`".
3. `[frameworks.onto]` present but `.homonto/catalog/skills/onto/` missing → "run `homonto apply`".

**D3 — Idempotent scaffold, skip-existing, report created vs skipped.** Mirror
`internal/scaffold.Init`'s skip-existing behavior: for each of
`docs/{changes,specs,adr,guides}`, `os.MkdirAll` (idempotent) and track whether
it pre-existed to report created vs skipped. Directory-only scaffold for this
increment (no template files inside — skeleton content is #3's concern). Never
overwrite. A `--dir` flag (default `.`) selects the workspace root, consistent
with `onto status`.

## Risks / Trade-offs

- **Applied-evidence heuristic** (`.homonto/catalog/skills/onto/`) → it is the
  materialization path homonto uses; if that layout changes, the check updates in
  one place. Acceptable structural proxy for "framework applied".
- **Lightweight TOML read vs config.Load** → the gate does not fully validate the
  config; it only needs `[frameworks.onto]` presence. This is deliberate (D2).
- **Directory-only scaffold** → no README/templates yet; #3 adds skeleton
  content. Keeps this change minimal.

## Migration Plan

Additive: new `internal/ontocli/init.go` + test; register one subcommand. No
change to existing packages/binaries. Rollback is removing the command.

## Open Questions

None blocking. Skeleton file templates inside the scaffolded dirs are deferred to
#3 (phase-gates / skeleton create-validate).
