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
