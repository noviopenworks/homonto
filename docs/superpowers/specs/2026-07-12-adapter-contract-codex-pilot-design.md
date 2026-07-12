---
comet_change: adapter-contract-codex-pilot
role: technical-design
canonical_spec: openspec
archived-with: 2026-07-12-adapter-contract-codex-pilot
status: final
---

# Adapter Contract + Codex Pilot — Technical Design

Deep refinement of `openspec/changes/adapter-contract-codex-pilot/design.md`.
Canonical behavior lives in the delta specs (`adapter-contract`, `codex-adapter`,
`tool-adapters`); this doc is the implementation plan and edge-case ledger.

## 1. Package layout

```
internal/tomlutil/            TOML codec: Get/Set/Delete/Canonical/EnsureRoot
internal/adapter/structproj/  Codec iface + Project/Apply/Observe engine
internal/adapter/codex/       Codex adapter (config.toml, mcp_servers.<n>)
  codex.go, codex_test.go, compat_test.go, testdata/config.toml
```

## 2. Codec contract (structproj)

```go
type Codec interface {
    EnsureRoot(doc []byte) ([]byte, error)                 // ""/ws → empty root
    Get(doc []byte, path string) (canonical string, ok bool)
    Set(doc []byte, path, jsonValue string) ([]byte, error)
    Delete(doc []byte, path string) ([]byte, error)
    Canonical(jsonValue string) string
}
```
`path` is codec-defined (JSON escaped segments; TOML dotted). Values crossing the
core are **JSON-encoded strings**, so state hashing (`secret.Hash(Canonical(v))`)
stays format-independent — the TOML codec converts TOML values ↔ canonical JSON.

## 3. Projection engine (structproj)

`Project(desired map[string]string, disk []byte, st, ns string, codec) ([]adapter.Change, error)`
reproduces the Claude/OpenCode loop:
- for each desired key `ns+"."+k`: read disk via `codec.Get`; compare against
  `desired[k]` and the recorded `state.Entry` → emit `create` (absent),
  `noop` (disk == want && state matches), `adopt` (disk == want but not in
  state), or `update`; secret-bearing values redact `Old`.
- for each state key under `ns` not in desired → `delete`.

`Apply(changes, disk, codec, res, st, ns)`: resolve secrets (already resolved by
engine two-phase), `codec.Set` each create/update, `codec.Delete` each delete,
record `state.Entry{Desired, Applied: Hash(Canonical(resolved))}`; write file
once. `Observe(st, disk, ns, codec)`: for each recorded key present on disk,
`Hash(codec.Get(...))`.

The JSON codec wraps existing `jsonutil` (Set/Delete/Get/Canonical/Standardize)
so this engine is behaviorally identical to today's adapters — verified by
porting a slice of the Claude MCP tests onto it.

## 4. tomlutil (TOML codec)

Backed by go-toml/v2. Represent the doc as a `map[string]any` tree
(unmarshal→edit→marshal) so unmanaged tables survive. `Get(path)`: walk dotted
segments, return `Canonical(jsonEncode(value))`, ok. `Set(path, jsonValue)`:
decode jsonValue to `any`, set at path creating intermediate tables, marshal.
`Delete(path)`: drop the leaf, prune emptied parent tables homonto created.
`Canonical`: round-trip through encoding/json for a stable form (normalize
number forms, sort object keys). `EnsureRoot`: empty/whitespace → `{}`→ empty
TOML. Comment preservation is a documented non-goal (matches OpenCode JSONC).

## 5. Codex adapter

- `Name() == "codex"`; file `~/.codex/config.toml`.
- `desired()`: for each `config.MCP` whose `TargetsOrAll()` (or explicit targets)
  includes `codex`, emit key `mcp_servers.<name>` → JSON object
  `{command, args, env}` matching Codex's documented MCP shape; env keeps
  unresolved `${...}` tokens (secret-safe).
- Plan/Apply/ObserveHashes delegate to structproj with `ns = "mcp_servers"` and
  the tomlutil codec. Secret redaction via existing `adapter.SecretRedaction` /
  `secret.Resolver`.
- Pilot scope is MCP; a `[settings.codex]` top-level mapping can ride the same
  engine later (kept minimal now).

## 6. Config + engine wiring

- `config`: add `codex` to the known-tool set used by target validation. Default
  `TargetsOrAll` stays `{claude, opencode}` (Codex opt-in), so no existing config
  changes behavior. A resource may list `codex` explicitly.
- `engine.Build`: append `codex.New(home)` to `Adapters`; it reads/writes
  `~/.codex/config.toml`. Plan/status/doctor iterate all adapters already.

## 7. Compatibility fixture (the contract test)

`testdata/config.toml` seeded with `model = "..."` and a user
`[mcp_servers.user_owned]`. `compat_test.go` asserts, via a real engine or the
adapter directly:
1. apply projects `[mcp_servers.declared]` and preserves `model` + `user_owned`;
2. second plan is byte-identical / all-noop (idempotent);
3. de-declaring `declared` prunes it, `user_owned` + `model` survive;
4. a pre-existing non-homonto `[mcp_servers.declared]` value is treated as a
   conflict/adopt per the core's rules, never silently clobbered.
This suite is the reusable "adapter conforms to the contract" template.

## 8. Migration of existing adapters (guarded, optional — task 6)

Attempt routing Claude's and OpenCode's MCP structured-file slice through
structproj. Acceptance: every existing adapter test stays green. If any behavior
diverges, revert the migration and record it as a follow-up — the exit gate (a
third adapter without duplicated control flow) is already met by Codex-on-core.

## 9. Test strategy

- tomlutil unit tests (set/get/delete/canonical/preserve/empty/nested env).
- structproj engine tests on a JSON codec (create/update/noop/adopt/delete +
  secret redaction) proving fidelity to today's behavior.
- codex adapter tests (projection, prune, idempotency, secret-safety) + the
  compatibility fixture suite.
- Gate: `go test -race ./...`, `./scripts/gate.sh`.

## 10. Edge cases

- Empty/absent `config.toml` → EnsureRoot yields an empty doc; first apply
  creates the file with only managed tables.
- TOML integer vs JSON number normalization handled in Canonical (round-trip).
- `env` map ordering: canonical form sorts keys so hashing is stable.
- Codex not installed / no `~/.codex`: apply creates the file; doctor reports
  presence like the other adapters.
