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
