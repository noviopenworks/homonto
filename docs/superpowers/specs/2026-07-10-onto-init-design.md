---
comet_change: onto-init
role: technical-design
canonical_spec: openspec
---

# Onto Init — Technical Design

Refinement of the open-phase `design.md` for `onto-init` (change #2 of 5 for the
`onto` binary). Adds `onto init` — the first mutating command — which scaffolds
`docs/{changes,specs,adr,guides}` behind a Homonto framework-install gate.

## Context

`onto-binary-foundation` (#1, archived) shipped the binary, `internal/ontostate`,
and read-only `onto status`. Per the dual-binary design, `onto` is Homonto-managed:
mutating commands require `homonto.toml` + `[frameworks.onto]` + installed
framework metadata. `onto init` is the first such command.

## Goals / Non-Goals

**Goals:** idempotent scaffold of the four `docs/` dirs; framework-install gate
that refuses (non-zero, zero `docs/` writes) with specific guidance when the
project isn't Homonto-prepared.

**Non-Goals:** #3 skeleton create-validate / phase gates, #4 doctor, #5 packaging;
creating `homonto.toml`; running the projection engine; touching `onto status`
or the state model.

## Decisions

**D1 — `internal/ontocli/init.go`: gate → scaffold, registered on the root.**
`initCmd()` has a `--dir` flag (default ".", like `status`). It runs the gate
first; only on success does it scaffold. Nothing is written on a gate failure.

**D2 — Gate = lightweight homonto.toml read + filesystem applied-check; NOT
`config.Load`.** To preserve #1's isolation of `onto` from homonto's projection
pipeline and avoid failing on unrelated config-validation errors, the gate:
- reads `<root>/homonto.toml` with `go-toml/v2` into a minimal struct
  `struct{ Frameworks map[string]toml.Unmarshaler /*or any*/ }` — actually
  `struct{ Frameworks map[string]any `toml:"frameworks"` }`; `[frameworks.onto]`
  present ⇔ `_, ok := Frameworks["onto"]`.
- treats `.homonto/catalog/skills/onto/` (a directory next to homonto.toml) as
  applied-evidence (the path homonto materializes the onto framework's skills to).

It does NOT import/call `internal/config.Load`, `internal/engine`, or
`internal/adapter`. Gate order, each exiting non-zero with specific guidance and
zero `docs/` writes:
1. no `homonto.toml` → "run `homonto init`".
2. `homonto.toml` but no `[frameworks.onto]` → "declare `[frameworks.onto]` and run `homonto apply`".
3. `[frameworks.onto]` but no `.homonto/catalog/skills/onto/` → "run `homonto apply`".

Rejected: `config.Load` (couples to catalog + full validation; a stray unrelated
config error would wrongly block `onto init`).

**D3 — Idempotent directory-only scaffold, skip-existing, created/skipped
report.** For each of `docs/{changes,specs,adr,guides}`: stat first (record
pre-existing), then `os.MkdirAll` (idempotent, 0o755). Report created vs skipped.
Never overwrite any existing file/dir. No template files inside — skeleton
content is #3's concern. Mirrors `internal/scaffold.Init`'s skip-existing style.

## Component Boundaries

| Unit | Responsibility | Depends on |
|---|---|---|
| `internal/ontocli` (init.go) | gate + scaffold + `initCmd` | `go-toml/v2`, `os`, cobra |

No new dependency (`go-toml/v2` is already a homonto dep). `onto` still imports
none of homonto's `internal/{cli,engine,config,adapter,catalog}`.

## Risks / Trade-offs

- **Applied-evidence heuristic** (`.homonto/catalog/skills/onto/`) → single-point
  structural proxy; updates in one place if the materialize layout changes.
- **Gate doesn't fully validate config** → deliberate (D2); only `[frameworks.onto]`
  presence matters here.
- **Directory-only scaffold** → templates deferred to #3; keeps this minimal.

## Testing Strategy

1. Gate: four outcomes (no toml / no onto / declared-not-applied / OK) over temp
   workspaces; assert guidance text and zero `docs/` writes in the three failures.
2. `onto init`: prepared workspace (homonto.toml with `[frameworks.onto]` + fake
   `.homonto/catalog/skills/onto/`) → creates four dirs, reports created, exit 0;
   second run idempotent (pre-existing + a user file under docs/ untouched,
   reported skipped); gate-failure cases exit non-zero, no docs/ writes.
3. Isolation: grep confirms init.go imports no homonto engine/adapter/config.
4. Regression: both binaries build; `go test [-race] ./...`, vet, gofmt, tidy.

## Open Questions

None blocking. Skeleton templates inside the scaffolded dirs → #3.
