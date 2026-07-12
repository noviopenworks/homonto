# Comet Design Handoff

- Change: adapter-contract-codex-pilot
- Phase: design
- Mode: compact
- Context hash: 0ff43049b54b80813ef5cd39ec1ba792afdda52dbd6ed1b06b1e75e7374689b7

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/adapter-contract-codex-pilot/proposal.md

- Source: openspec/changes/adapter-contract-codex-pilot/proposal.md
- Lines: 1-62
- SHA256: 9180dcb07f09d724dd1862584d48576ea82baefbd7cd66087fcb4e463a958082

```md
## Why

Adding a tool adapter today means re-implementing the entire projection control
flow. Claude and OpenCode each carry their own copy of the same machinery:
compute desired managed-key → value, diff against disk and state to produce
create/update/delete/noop/adopt changes, surgically merge only managed keys into
the tool's structured file (preserving unmanaged keys), and record state. The
only real difference between them is the **file format** (Claude JSON vs OpenCode
JSONC) and the **key mapping**. Without a shared contract, a third adapter copies
hundreds of lines of subtle, security-sensitive merge logic. Roadmap item 11
requires publishing an adapter contract that removes this duplication and proving
it with one pilot adapter that ships without re-implementing the control flow.

## What Changes

- **Adapter contract — a shared managed-key projection core.** Extract the
  duplicated structured-file projection (desired → plan diff → surgical apply →
  observe hashes) into a format-agnostic core parameterized by a `Codec`
  (get/set/delete/canonicalize a key path in a document) and a managed-key
  namespace. The existing `jsonutil` becomes the JSON codec.
- **TOML codec.** A new `tomlutil` codec (get/set/delete/canonical over TOML)
  so the core can drive a TOML-configured tool.
- **Codex pilot adapter.** A real third adapter for **Codex (OpenAI Codex CLI)**,
  which stores config in `~/.codex/config.toml` with `[mcp_servers.<name>]`
  tables (command/args/env). It projects MCP servers surgically through the
  shared core + TOML codec, implementing only the per-adapter parts (file path,
  key mapping, codec) — proving a third adapter ships without duplicating the
  Claude/OpenCode control flow. Wired into config targets, engine, plan/apply/
  status/doctor.
- **Compatibility fixture format.** A real-config fixture (a `config.toml` with
  unmanaged user keys) plus tests asserting the pilot's surgical merge preserves
  unmanaged content, is idempotent (byte-stable re-plan), and prunes de-declared
  keys — the reusable shape for validating any future adapter.

Non-goal (deferred to follow-up): deep catalog governance
(versioning/deprecation/provenance automation); multiple new adapters in
parallel; migrating Claude/OpenCode's non-structured-file projection (skills,
plugins, links) onto the core beyond what keeps their behavior byte-identical.

## Capabilities

### New Capabilities
- `adapter-contract`: the format-agnostic managed-key projection core (Codec
  interface + plan/apply/observe engine + namespace ownership) and the
  compatibility-fixture contract a conforming adapter must satisfy.
- `codex-adapter`: the Codex pilot adapter — `~/.codex/config.toml`
  `[mcp_servers.<name>]` surgical projection built on the contract.

### Modified Capabilities
- `tool-adapters`: Codex joins Claude and OpenCode as a supported adapter; the
  adapter surface is now contract-driven.

## Impact

- New `internal/adapter/structproj` (or equivalent) projection core + `Codec`.
- New `internal/tomlutil` TOML codec.
- New `internal/adapter/codex/` adapter + tests + `testdata/` fixture.
- `internal/config` (Codex as a known target/tool), `internal/engine` (wire the
  Codex adapter), plan/status/doctor tool lists.
- New ADR (adapter contract) + guide.
- Claude/OpenCode behavior unchanged (their tests stay green); the core is
  additive, and the JSON codec is the same `jsonutil` they already use.

```

## openspec/changes/adapter-contract-codex-pilot/design.md

- Source: openspec/changes/adapter-contract-codex-pilot/design.md
- Lines: 1-95
- SHA256: ba04ab7c216527679d802a251c2e72feb3c8a3977086d9073bea7e9332911f81

[TRUNCATED]

```md
## Context

The `adapter.Adapter` interface (Name/Plan/Apply/ObserveHashes) is already clean.
The duplication is in the **implementations**: `internal/adapter/claude` and
`internal/adapter/opencode` each carry the same managed-key structured-file
projection — a `desired()` map (managed key → JSON value), a Plan loop diffing
desired vs on-disk (via `jsonutil.Get/Canonical`) vs recorded state to emit
create/update/delete/noop/adopt, an Apply that writes only managed keys via
`jsonutil.SetJSON/DeleteJSON` (preserving unmanaged keys), and an ObserveHashes
that re-hashes recorded keys from disk. Only the file format and the key mapping
differ. A third adapter (Codex, TOML) must not copy this again.

## Goals / Non-Goals

**Goals:**
- Publish a format-agnostic projection core (the adapter contract) that owns the
  plan/apply/observe control flow for managed keys in a structured file.
- Parameterize it by a `Codec` (get/set/delete/canonicalize a key path) so JSON
  (existing `jsonutil`) and TOML (new `tomlutil`) both plug in.
- Ship a Codex pilot adapter on the core: `~/.codex/config.toml`
  `[mcp_servers.<name>]` MCP projection, surgical + idempotent + fail-safe on
  unmanaged keys, with plan/apply/status/doctor.
- Provide a real-config compatibility fixture format proving the pilot preserves
  unmanaged content and is byte-stable.

**Non-Goals:**
- Rewriting Claude/OpenCode onto the core (behavior-risk); the core is proven by
  Codex. A same-behavior migration of their MCP slice is attempted only if it
  keeps every existing test green, else documented as follow-up.
- Catalog governance (versioning/deprecation/provenance) and skills/plugins/link
  projection for Codex (MCP + minimal settings only in the pilot).

## Decisions

### 1. The projection core (`internal/adapter/structproj`)
A `Codec` abstracts the document:
```
type Codec interface {
    Get(doc []byte, path string) (canonicalValue string, present bool)
    Set(doc []byte, path string, jsonValue string) ([]byte, error)
    Delete(doc []byte, path string) ([]byte, error)
    Canonical(jsonValue string) string   // stable form for hashing/compare
    EnsureRoot(doc []byte) ([]byte, error) // empty/whitespace → empty object doc
}
```
`Project(desired map[string]string, disk []byte, st *state.State, ns string, codec Codec) ([]adapter.Change, error)` reproduces the existing Claude/OpenCode diff loop exactly (create/update/noop/adopt for declared keys; delete for state keys under `ns` no longer desired). `Apply(changes, disk, codec, res, st, ns)` writes only managed keys and records state. `Observe(st, disk, ns, codec)` re-hashes recorded keys. This is the extracted control flow — one implementation, format-injected.

The path grammar is the codec's concern: JSON uses the existing `jsonutil`
segment escaping; TOML uses dotted `mcp_servers.<name>.command` addressing.

### 2. TOML codec (`internal/tomlutil`)
Backed by `github.com/pelletier/go-toml/v2` (already a dependency). `Get` reads a
dotted path and returns a canonical JSON encoding of the value (so the core can
compare/hash values format-independently — the state layer stays JSON-based).
`Set`/`Delete` edit the TOML tree and re-marshal, preserving unmanaged tables and
keys. `Canonical` normalizes the JSON value form. Comment preservation is a
non-goal (documented), matching OpenCode's JSONC limitation.

### 3. Codex adapter (`internal/adapter/codex`)
Config file `~/.codex/config.toml`. Managed keys: for each declared MCP server
targeting `codex`, project `mcp_servers.<name>` = `{command, args, env}` (Codex's
documented MCP shape). The adapter supplies only: `Name() == "codex"`, the file
path, the `desired()` mapping (config MCPs → `mcp_servers.<name>` values, secret
tokens preserved), and the TOML codec; Plan/Apply/ObserveHashes delegate to the
core. Secret handling reuses the existing `secret.Resolver`/redaction so `plan`
never prints a secret. Minimal settings projection (e.g. `model`) may ride the
same core if a `[settings.codex]` mapping is declared; MCP is the pilot's proof.

### 4. Config / engine wiring
Add `codex` to the known tool set (config target validation, `TargetsOrAll`
default stays claude+opencode unless a resource opts into codex — the pilot is
opt-in so existing configs are unaffected). The engine constructs the Codex
adapter alongside the others. Plan/status/doctor include it.

### 5. Compatibility fixture contract
`testdata/` holds a real `config.toml` with unmanaged user keys (`model`, a
user's own `[mcp_servers.other]`). The fixture suite asserts: apply projects the
declared server, preserves the unmanaged key and the user's own server, a second
plan is byte-identical (idempotent), a de-declared server is pruned while
unmanaged content survives, and a non-homonto value is never clobbered. This is

```

Full source: openspec/changes/adapter-contract-codex-pilot/design.md

## openspec/changes/adapter-contract-codex-pilot/tasks.md

- Source: openspec/changes/adapter-contract-codex-pilot/tasks.md
- Lines: 1-52
- SHA256: abf03d8369f64932e4e11cf72087c6646f2b8a07f2167fe13a977562951d3a54

```md
## 1. TOML codec (internal/tomlutil)

- [ ] 1.1 `Get(doc, dottedPath) (canonicalJSON, present)`, `Set(doc, path,
  jsonValue)`, `Delete(doc, path)`, `Canonical(jsonValue)`, `EnsureRoot(doc)`
  over TOML via go-toml/v2, preserving unmanaged tables/keys.
- [ ] 1.2 Tests: set/get/delete round-trip; unmanaged keys preserved; canonical
  value form stable; empty/whitespace doc handled; nested `mcp_servers.<n>.env`.

## 2. Projection core / adapter contract (internal/adapter/structproj)

- [ ] 2.1 `Codec` interface + `Project(desired, disk, st, ns, codec)` producing
  create/update/delete/noop/adopt changes (mirrors the Claude/OpenCode diff loop).
- [ ] 2.2 `Apply(changes, disk, codec, res, st, ns)` writes only managed keys,
  records state, preserves unmanaged keys; `Observe(st, disk, ns, codec)`
  re-hashes recorded keys.
- [ ] 2.3 Tests against a JSON codec (adapting jsonutil) proving the core
  reproduces create/update/noop/adopt/delete + secret redaction semantics.

## 3. Codex pilot adapter (internal/adapter/codex)

- [ ] 3.1 Adapter: `Name()=="codex"`, `~/.codex/config.toml` path, `desired()`
  mapping declared MCPs → `mcp_servers.<name>` {command,args,env} (secret tokens
  preserved), Plan/Apply/ObserveHashes delegating to the core + TOML codec.
- [ ] 3.2 Secret safety: plan redacts secret-bearing values; apply resolves only
  after confirm (reuse secret.Resolver). Test plan never prints a secret.

## 4. Config + engine wiring

- [ ] 4.1 Add `codex` to the known tool set (target validation); keep default
  targets claude+opencode so existing configs are unaffected (codex is opt-in).
- [ ] 4.2 Engine constructs the Codex adapter; plan/status/doctor include it.

## 5. Compatibility fixture suite

- [ ] 5.1 `testdata/` real `config.toml` with unmanaged keys + a user's own
  `[mcp_servers.other]`.
- [ ] 5.2 Suite: apply projects the declared server; unmanaged key + user server
  preserved; second plan byte-identical (idempotent); de-declared server pruned
  while unmanaged survives; non-managed value never clobbered (conflict-safe).

## 6. Optional same-behavior migration (guarded)

- [ ] 6.1 If it keeps every existing Claude/OpenCode test green, route their MCP
  structured-file projection through structproj to remove the duplication in
  place; otherwise leave them and record the migration as a follow-up.

## 7. Docs, ADR, gate

- [ ] 7.1 ADR: adapter contract (format-agnostic projection core + Codec).
- [ ] 7.2 Guide/README: Codex as a supported target; the adapter-contract shape.
- [ ] 7.3 Roadmap item 11 → done (or partial with follow-up) with evidence.
- [ ] 7.4 Full gate green: `go test -race ./...`, `./scripts/gate.sh`.

```

## openspec/changes/adapter-contract-codex-pilot/specs/adapter-contract/spec.md

- Source: openspec/changes/adapter-contract-codex-pilot/specs/adapter-contract/spec.md
- Lines: 1-52
- SHA256: 8ba45e598bdd873c0fd0ca1a95fa9e4434c7ea8a7fe865e1df3a79b7b929d055

```md
## ADDED Requirements

### Requirement: Format-agnostic managed-key projection core

homonto SHALL provide a projection core that owns the managed-key control flow
for a structured config file, parameterized by a format Codec so a new adapter
supplies only its file path, key mapping, and codec. The core SHALL produce the
same create, update, delete, noop, and adopt changes the built-in adapters
produce, write only managed keys while preserving unmanaged content, and
re-hash recorded keys for drift detection.

#### Scenario: Declared key projects as create then noop

- **GIVEN** a managed key declared for a tool whose file lacks it
- **WHEN** plan runs, then apply, then plan again
- **THEN** the first plan is a create, apply writes the key, and the second plan is a noop

#### Scenario: De-declared key is pruned

- **GIVEN** a managed key recorded in state but no longer declared
- **WHEN** plan runs
- **THEN** the core emits a delete for that key and apply removes only it

#### Scenario: Unmanaged content is preserved

- **GIVEN** a config file holding keys homonto does not manage
- **WHEN** apply writes a managed key
- **THEN** every unmanaged key is preserved byte-for-byte outside the managed change

### Requirement: Codec abstracts the file format

The projection core SHALL depend only on a Codec that can get, set, delete, and
canonicalize a value at a key path in a document, and normalize an empty
document to an object root. A JSON codec and a TOML codec SHALL both satisfy the
Codec so the same core drives JSON- and TOML-configured tools.

#### Scenario: JSON and TOML codecs drive the same core

- **GIVEN** the projection core and equivalent desired state
- **WHEN** it runs with a JSON codec against a JSON file and a TOML codec against a TOML file
- **THEN** both produce equivalent managed-key changes and preserve unmanaged content

### Requirement: Adapter compatibility fixture contract

A conforming adapter SHALL be validated by a real-config compatibility fixture
that proves surgical merge, idempotency, pruning, and conflict safety.

#### Scenario: Compatibility fixture passes

- **GIVEN** a real config file with unmanaged user content and a managed declaration
- **WHEN** the fixture suite runs apply, re-plan, and de-declare
- **THEN** the managed key is projected, unmanaged content survives, the re-plan is byte-identical, the de-declared key is pruned, and a non-homonto value is never clobbered

```

## openspec/changes/adapter-contract-codex-pilot/specs/codex-adapter/spec.md

- Source: openspec/changes/adapter-contract-codex-pilot/specs/codex-adapter/spec.md
- Lines: 1-39
- SHA256: 4928eccafc309a742d521a74468fb516685e86fc3ca116ab850ce42d54c9e225

```md
## ADDED Requirements

### Requirement: Codex MCP projection

homonto SHALL project declared MCP servers targeting Codex into
`~/.codex/config.toml` as `mcp_servers.<name>` tables holding command, args, and
env, surgically and idempotently, built on the shared projection core and a TOML
codec. Unmanaged tables and keys in `config.toml` SHALL be preserved, and a
consecutive plan SHALL report no changes.

#### Scenario: MCP server projected to config.toml

- **GIVEN** an MCP server declared with a codex target against a config.toml holding an unrelated key
- **WHEN** apply runs
- **THEN** `mcp_servers.<name>` holds the command/args/env and the unrelated key is preserved

#### Scenario: De-declared Codex MCP server is pruned

- **GIVEN** a Codex `mcp_servers.<name>` previously written and recorded by homonto, no longer declared
- **WHEN** apply runs
- **THEN** that server table is removed and any unmanaged server table is preserved

#### Scenario: Codex projection is idempotent

- **GIVEN** a Codex MCP server already applied
- **WHEN** plan runs twice consecutively
- **THEN** both plans report no changes

### Requirement: Codex secret safety

Codex projection SHALL preserve the secret guarantees: plan SHALL NOT resolve or
print a secret value (it shows the token), and apply SHALL resolve secrets only
after confirmation, storing only the unresolved token plus a hash in state.

#### Scenario: Plan does not reveal a Codex secret

- **GIVEN** a Codex MCP server whose env references a secret token
- **WHEN** plan runs
- **THEN** the plan shows the token or a redaction, never the resolved secret value

```

## openspec/changes/adapter-contract-codex-pilot/specs/tool-adapters/spec.md

- Source: openspec/changes/adapter-contract-codex-pilot/specs/tool-adapters/spec.md
- Lines: 1-21
- SHA256: f1949d98b090ef858fe2098d5bd6b45b3c9d92cbe8a0f1bca0dd1ceece019072

```md
## ADDED Requirements

### Requirement: Codex is a supported adapter

Codex SHALL be a supported tool adapter alongside Claude and OpenCode, selected
when a resource targets `codex`. The default target set SHALL remain Claude and
OpenCode so existing configs are unaffected, and Codex projection SHALL be
opt-in per resource. The Codex adapter SHALL be built on the shared adapter
contract rather than duplicating the Claude or OpenCode control flow.

#### Scenario: Codex target is recognized

- **GIVEN** a resource that lists `codex` in its targets
- **WHEN** the config loads and plan runs
- **THEN** the Codex adapter produces its changes and unknown-target validation still rejects other unknown tools

#### Scenario: Default targets exclude Codex

- **GIVEN** a resource with no explicit targets
- **WHEN** it is projected
- **THEN** it targets Claude and OpenCode only, leaving Codex opt-in

```
