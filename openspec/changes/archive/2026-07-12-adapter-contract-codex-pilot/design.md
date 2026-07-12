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
the reusable "does an adapter conform to the contract" template.

## Risks / Trade-offs

- **Core-extraction fidelity.** The core must reproduce the existing diff
  semantics exactly. Mitigation: build the core to the Codex tests first, and
  only attempt Claude/OpenCode migration if their full suites stay green;
  otherwise leave them and document the follow-up (the exit gate — a third
  adapter without duplication — is met by Codex-on-core regardless).
- **TOML value ↔ JSON state.** State/hashing is JSON-based; the TOML codec
  bridges by canonicalizing values to JSON. Structural TOML/JSON mismatches
  (e.g. integers vs floats) are normalized in `Canonical` and covered by tests.
- **Codex config schema drift.** Codex's `config.toml`/`mcp_servers` shape is
  external; the adapter targets the documented form and the fixture pins it, so
  a schema change is a localized, tested update.
