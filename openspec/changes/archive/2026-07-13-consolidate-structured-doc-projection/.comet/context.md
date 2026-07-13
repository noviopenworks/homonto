# Comet Design Handoff

- Change: consolidate-structured-doc-projection
- Phase: design
- Mode: compact
- Context hash: 8696469ecab9b795623ba4725c7b5740e1b3b097268e3eee522c25933937c8e4

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/consolidate-structured-doc-projection/proposal.md

- Source: openspec/changes/consolidate-structured-doc-projection/proposal.md
- Lines: 1-58
- SHA256: 0ccef2a2ad377efc06bf728e665851b28e81904d54d970e597ff1320541524b6

```md
# Consolidate Claude/OpenCode structured-document projection onto structproj

## Why

Roadmap finding F40: the `claude` (1037 lines) and `opencode` (999 lines)
adapters each re-implement security-sensitive managed-key projection —
diffing desired values against disk and recorded state to emit
create/update/adopt/delete/noop changes, writing only managed keys while
preserving unmanaged content, and re-hashing recorded keys for drift. The
Codex adapter already proved the better shape (item 11): a shared
`internal/adapter/structproj` core plus a per-format `Codec`, so Codex is
129 lines. Two ~1000-line adapters duplicating this logic is the largest
remaining maintenance and correctness risk in the adapter layer — a fix or
audit must be made in three places, and the two big adapters can silently
drift from the contract the conformance suite pins.

## What Changes

Migrate the **structured-document** portion of `claude` and `opencode` onto
`structproj.Project` / `Apply` / `Observe`, following the Codex template:

- Add a shared **JSON codec** (`structproj.Codec` backed by the existing
  `internal/jsonutil` primitives — `ObjectRoot`, `GetJSON`, `SetJSON`,
  `DeleteJSON`, `Canonical`). Both JSON adapters share one codec; no new
  format code.
- For each adapter, group its managed structured-doc keys by the document
  they live in and supply a `pathFor` per document, then replace the
  bespoke diff/apply/observe loops with `structproj` calls per document:
  - **claude**: `.claude/settings.json` (`setting.*`); `.claude.json`
    (`mcp.*`, `plugin.*`, `pluginconfig.*`, `marketplace.*`).
  - **opencode**: `opencode.json` (`mcp.*`, `setting.*`).
- Delete the duplicated structured-doc projection logic from both adapters.

The **file-projection** surface (skills/commands/subagents symlinks,
inactive dirs, copy-subagents via `copyfile`) is explicitly out of scope and
stays in each adapter unchanged — it needs a separate shared contract
(documented follow-on).

## Impact

- **Specs:** `adapter-contract` gains a requirement that the built-in JSON
  adapters project structured documents through the shared core via a shared
  JSON codec.
- **Behavior:** none. This is a pure refactor pinned by the existing
  `internal/adapter/conformance` suite plus every `claude`/`opencode` test.
  Plan/apply/observe output must be byte-for-byte identical.
- **Risk:** medium — security-sensitive projection/adopt/redaction logic. The
  conformance suite (create/update/delete, adopt, drift, secret
  non-resolution, malformed docs, foreign-content) and the adapters' own
  ~2000 lines of tests are the safety net; migration is per-document and
  verified green after each step.

## Non-goals

- File-projection / symlink / prune / copy-subagent consolidation (the
  higher-risk second surface — separate change).
- Any change to `structproj`'s projection semantics.
- Any adapter behavior change, schema change, or new capability.

```

## openspec/changes/consolidate-structured-doc-projection/design.md

- Source: openspec/changes/consolidate-structured-doc-projection/design.md
- Lines: 1-82
- SHA256: d610d2d9952cdc63eb9ef1ac63905546ba67c148f5887d7cfde4ddc6b6bc1556

[TRUNCATED]

```md
# Design — consolidate structured-doc projection

## High-level approach

Follow the Codex template (`internal/adapter/codex/codex.go`), generalized
to (a) a JSON codec and (b) multiple documents per adapter.

### Shared JSON codec

`internal/jsonutil` already exposes every primitive `structproj.Codec`
requires. Add one small adapter type (in `structproj` or a new
`internal/adapter/jsoncodec`, decided in build) mapping:

| structproj.Codec | jsonutil |
|------------------|----------|
| `EnsureRoot`     | `ObjectRoot` (normalize empty→`{}`) |
| `Get`            | `GetJSON` |
| `Set`            | `SetJSON` |
| `Delete`         | `DeleteJSON` |
| `Canonical`      | `Canonical` |

Both `claude` and `opencode` share this one codec (both are JSON).

### Per-document namespaces

`structproj.Project(tool, prefix, desired, disk, st, codec, pathFor)` acts on
**one document** with **one key prefix**. Each adapter maps its managed keys
to documents:

- **claude**
  - `settings.json` ← keys with prefix `setting.`; `pathFor` → the settings
    JSON path for that key.
  - `.claude.json` ← prefixes `mcp.`, `plugin.`, `pluginconfig.`,
    `marketplace.`; `pathFor` → `mcpServers.<n>`, `enabledPlugins.<source>`,
    `pluginConfigs.<source>`, `extraKnownMarketplaces.<name>`.
- **opencode**
  - `opencode.json` ← prefixes `mcp.`, `setting.`.

Because `structproj.Project`/`Observe`/`Apply` filter recorded keys by
`strings.HasPrefix(k, prefix)`, a single document holding several prefixes is
handled by calling the trio once per prefix against that document (the
existing multi-prefix docs), OR by a prefix that is the empty-string-free
common cut. Build step decides the cleanest split; the invariant is that the
union of per-namespace outputs equals today's flat output.

### Migration order (per adapter, each step green before the next)

1. Introduce the shared JSON codec + its unit test.
2. claude: route `setting.*` (settings.json) through structproj; delete that
   branch of the bespoke loop; run claude + conformance suites.
3. claude: route `.claude.json` prefixes through structproj; delete those
   branches; run suites.
4. opencode: route `opencode.json` prefixes through structproj; delete the
   bespoke loop; run suites.
5. Confirm file-projection code paths in both adapters are untouched.

### Correctness invariant

`structproj` "reproduces the built-in adapters' semantics exactly, including
secret-safe redaction of Old" (its own doc). The migration is behavior-
preserving iff, for every fixture in the conformance suite and every existing
claude/opencode test, plan/apply/observe output is unchanged. Any diff is a
migration bug, fixed before proceeding — never by editing a test to match.

## Key risk (surfaced in open)

Does `structproj` cover every structured-doc behavior the two adapters have
today — specifically the **adopt** path (disk matches desired but state is
stale), **secret-bearing** desired values (never read/expose on-disk value),
and **Old redaction** for updates/deletes of unknown provenance? Reading
`structproj.Project`, all three are already implemented identically to the
claude loop. If build uncovers a claude/opencode structured-doc behavior with
no structproj equivalent (e.g. a canonicalization quirk), the change pauses:
extend `structproj` minimally (additive) or document the divergence — it does
**not** silently change adapter behavior.

## Alternatives considered

- **Full F40 (incl. file-projection) now** — rejected as too large/high-risk
  for one change; file-projection needs its own contract.

```

Full source: openspec/changes/consolidate-structured-doc-projection/design.md

## openspec/changes/consolidate-structured-doc-projection/tasks.md

- Source: openspec/changes/consolidate-structured-doc-projection/tasks.md
- Lines: 1-23
- SHA256: 61f79290ad197fc2130739049c7e346175703219d824a254e51f5d9042276468

```md
# Tasks — consolidate-structured-doc-projection

## 1. Shared JSON codec
- [ ] Add a `structproj.Codec` backed by `internal/jsonutil` (EnsureRoot→
      ObjectRoot, Get→GetJSON, Set→SetJSON, Delete→DeleteJSON, Canonical→
      Canonical), shared by claude + opencode. TDD: codec unit test round-trips
      get/set/delete/canonical and normalizes an empty doc.

## 2. claude structured-doc migration
- [ ] Route `setting.*` (settings.json) through structproj.Project/Apply/
      Observe; delete the bespoke branch. claude + conformance suites green.
- [ ] Route `.claude.json` prefixes (mcp/plugin/pluginconfig/marketplace)
      through structproj; delete those branches. Suites green.

## 3. opencode structured-doc migration
- [ ] Route `opencode.json` prefixes (mcp/setting) through structproj; delete
      the bespoke loop. opencode + conformance suites green.

## 4. Confirm scope + verify
- [ ] File-projection paths (skills/commands/subagents symlinks, inactive
      dirs, copy-subagents) untouched in both adapters.
- [ ] `go test ./... -race`, `go vet`, `go build`, `openspec validate --all`
      green; plan/apply/observe output byte-identical (conformance suite).

```

## openspec/changes/consolidate-structured-doc-projection/specs/adapter-contract/spec.md

- Source: openspec/changes/consolidate-structured-doc-projection/specs/adapter-contract/spec.md
- Lines: 1-41
- SHA256: b4a06be4c54ce4ca6f4b4742784653317294d91053c306412a37d932b46c2ac5

```md
# adapter-contract

## ADDED Requirements

### Requirement: Built-in JSON adapters project structured documents through the shared core

The `claude` and `opencode` adapters SHALL project their structured-document
managed keys (JSON config documents) through the shared
`internal/adapter/structproj` core — `Project`, `Apply`, and `Observe` — via a
shared JSON `Codec` backed by `internal/jsonutil`, rather than each
re-implementing the diff/write/observe control flow. Each adapter maps its
managed keys to the document they live in and supplies a `pathFor` per
document. The structured-document projection behavior — create/update/adopt/
delete/noop diffing, managed-key-only writes preserving unmanaged content,
secret-safe `Old` redaction, and drift re-hashing — MUST be identical to the
prior bespoke implementation, as pinned by the shared conformance suite.

This requirement covers only structured-document projection; file-projection
surfaces (symlinked skills/commands/subagents, copy-mode subagents) are out of
its scope and remain adapter-owned.

#### Scenario: Claude routes settings and .claude.json keys through the core

- **WHEN** the `claude` adapter plans, applies, and observes its managed
  `setting.*`, `mcp.*`, `plugin.*`, `pluginconfig.*`, and `marketplace.*` keys
- **THEN** it does so through `structproj.Project` / `Apply` / `Observe` with a
  shared JSON codec, and the resulting changes, on-disk writes, and observed
  hashes are byte-for-byte identical to the prior implementation

#### Scenario: OpenCode routes opencode.json keys through the core

- **WHEN** the `opencode` adapter plans, applies, and observes its managed
  `mcp.*` and `setting.*` keys in `opencode.json`
- **THEN** it does so through the shared `structproj` core and the shared JSON
  codec, preserving unmanaged content and secret-safe redaction unchanged

#### Scenario: Shared JSON codec is used by both JSON adapters

- **WHEN** either JSON adapter projects a structured document
- **THEN** it uses the one shared JSON `Codec` (backed by `internal/jsonutil`),
  not a per-adapter reimplementation of the format primitives

```
