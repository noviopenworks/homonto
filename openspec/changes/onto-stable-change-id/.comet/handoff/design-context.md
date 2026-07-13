# Comet Design Handoff

- Change: onto-stable-change-id
- Phase: design
- Mode: compact
- Context hash: bf4af8d7eb26b85439c7e32eedf651a2c6c33734f964267439b64ac52864f6ca

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/onto-stable-change-id/proposal.md

- Source: openspec/changes/onto-stable-change-id/proposal.md
- Lines: 1-38
- SHA256: 47595565c84d5b88c707b5b7c8bddb11270831b6728e055ecc3bb58e52ee200d

```md
# onto: a stable change ID, independent of the change name

## Why

Roadmap X1 (stable IDs). X1's original tension — stable IDs vs OpenSpec's
name-matching — is about the external comet/OpenSpec tooling. But **onto is
homonto's own binary-enforced workflow**, keyed today entirely on the change name
/ directory (`docs/changes/<name>/`). A rename loses all identity, and a
dependency (`deps: [<name>]`) or any cross-reference is a fragile string match —
the exact fragility X1 names. onto can carry a stable identity without touching
OpenSpec: a content-independent ID assigned once at creation.

## What Changes

- `onto-state.yaml` gains a stable `id` — a short random hex identifier assigned
  once by `onto new` and never rewritten. It survives a change rename and is the
  durable anchor for future traceability (deps-by-id, cross-refs).
- `onto new` generates the id (crypto/rand); `onto set`/`advance`/`close`
  preserve it verbatim (immutable); `onto state --json` and `onto status` surface
  it.
- On-read of a legacy state with no `id`, the value stays empty (backward
  compatible) — the id is assigned only at creation, never retroactively minted
  (so it cannot change meaning across reads).

## Impact

- **Specs:** `onto-binary` gains a requirement that a change carries a stable,
  name-independent id assigned at creation and never mutated.
- **Behavior:** additive; every existing command works, legacy states load with
  an empty id.
- **Risk:** low — an additive immutable field + generation at `onto new`; Go
  tests pin generation, uniqueness, and immutability.

## Non-goals

- Migrating `deps` or references from names to ids (the traceability-graph
  follow-on); retroactively minting ids for legacy changes; any change to the
  comet/OpenSpec flow (which keeps its name-matching).

```

## openspec/changes/onto-stable-change-id/design.md

- Source: openspec/changes/onto-stable-change-id/design.md
- Lines: 1-31
- SHA256: 108d2c619ce5eec3edf874109cf402e23b016fe85f7119a7123d5bb11e9750e1

```md
# Design — onto stable change id

## Approach

`ontostate.State` gains `ID string` (`yaml:"id,omitempty" json:"id,omitempty"`).
`onto new` (`internal/ontocli/new.go` `runNew`) generates it via a `newID()` that
reads crypto/rand and hex-encodes 4 bytes (an 8-char hex id), setting it on the
State before Save. No other command writes `ID`; `Save` round-trips it, so
`set`/`advance`/`close` (which Load → mutate → Save) preserve it verbatim. `Load`
leaves an absent id empty and never mints one, so an id never changes meaning
across reads (backward-compatible with pre-feature states).

`onto state --json` already marshals the whole State, so the id surfaces via the
json tag; `onto status` prints it in its per-change summary.

## Why crypto/rand is fine here

The "no Date/random" rule constrains the comet workflow *scripts* (Math.random/
Date.now break resume). This is the onto Go binary — a normal program — where
crypto/rand for a one-time id is correct. Tests assert the id is present,
8 hex chars, unique across two changes, and unchanged by transitions — not a
fixed value.

## Risk

Low — additive immutable field + generation at creation. onto* command tests
pin generation/uniqueness/immutability.

## Alternatives
- A content/timestamp-derived id — rejected; not stable across a rename and not
  guaranteed unique. A random id assigned once is both.

```

## openspec/changes/onto-stable-change-id/tasks.md

- Source: openspec/changes/onto-stable-change-id/tasks.md
- Lines: 1-11
- SHA256: d4e8fdfeeaa180f7a9b07267f7d0ee50b0969d558b2119c473b5b4d8df1edbfa

```md
# Tasks — onto-stable-change-id

## 1. Stable id field + generation + immutability
- [ ] State gains `id` (yaml/json); onto new generates a short random hex id;
      set/advance/close preserve it; state --json / status surface it; legacy
      (no id) loads empty, never retro-minted. TDD: new produces a well-formed
      unique id; a second change differs; the id survives advance/set unchanged.

## 2. Verify
- [ ] `go test ./internal/ontostate/... ./internal/ontocli/... -race`, vet,
      build (incl. cmd/onto), `openspec validate --all` green.

```

## openspec/changes/onto-stable-change-id/specs/onto-binary/spec.md

- Source: openspec/changes/onto-stable-change-id/specs/onto-binary/spec.md
- Lines: 1-24
- SHA256: c67809dacf11f35c10960fc33a7269e664e9e6d865abb1ebf7e3910e6dbe7a2a

```md
# onto-binary

## ADDED Requirements

### Requirement: A change carries a stable, name-independent id

`onto new` SHALL assign each change a stable identifier stored as `id` in its
`onto-state.yaml` — a content-independent value generated once at creation that
is never rewritten by any later command (`set`, `advance`, `close` preserve it
verbatim), so a change's identity survives a rename of its name or directory.
`onto state --json` and `onto status` MUST surface the id. A legacy state file
with no `id` MUST load with an empty id (backward compatible) and MUST NOT have
one retroactively minted, so an id never changes meaning across reads.

#### Scenario: onto new assigns a stable unique id

- **WHEN** two changes are created with `onto new`
- **THEN** each has a non-empty `id` in its `onto-state.yaml`, the two ids differ,
  and each id is unchanged by subsequent `advance`/`set`

#### Scenario: a legacy state without an id loads unchanged

- **WHEN** an `onto-state.yaml` written before this feature (no `id`) is read
- **THEN** it loads with an empty id and no id is minted on read

```
