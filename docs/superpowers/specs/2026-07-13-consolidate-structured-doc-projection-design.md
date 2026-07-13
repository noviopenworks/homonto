---
comet_change: consolidate-structured-doc-projection
role: technical-design
canonical_spec: openspec
status: draft
---

# consolidate-structured-doc-projection — Technical Design

Deep design for F40 (structured-doc slice). OpenSpec is the canonical spec;
this document is the technical design and defers to `openspec/changes/
consolidate-structured-doc-projection/specs` for normative requirements.

## Problem

`claude` (1037 LOC) and `opencode` (999 LOC) each re-implement structured-
document managed-key projection: the diff loop (create/update/adopt/delete/
noop), managed-key-only writes preserving unmanaged content, secret-safe `Old`
redaction, and drift re-hashing. `internal/adapter/structproj` (from item 11,
proven by Codex at 129 LOC) already owns exactly this control flow behind a
`Codec`. The duplication is a correctness and audit hazard: three copies of
security-sensitive logic that can drift from the conformance-suite contract.

## Approach (confirmed scope: structured-doc surface only)

Generalize the Codex template to (a) a shared JSON codec and (b) multiple
documents per adapter.

### 1. Shared JSON codec

Add one type implementing `structproj.Codec` over `internal/jsonutil`:

```
EnsureRoot -> jsonutil.ObjectRoot (empty/whitespace -> {} root)
Get        -> jsonutil.GetJSON
Set        -> jsonutil.SetJSON
Delete     -> jsonutil.DeleteJSON
Canonical  -> jsonutil.Canonical
```

Location decided in build: a new `internal/adapter/jsoncodec` (importable by
both adapters without a claude↔opencode dependency) is the leading option; a
value type in `structproj` is the fallback. Both JSON adapters share the one
codec — no per-adapter format code.

### 2. Per-document namespaces

`structproj.Project(tool, prefix, desired, disk, st, codec, pathFor)` operates
on one document + one key prefix, filtering recorded keys by
`HasPrefix(key, prefix)`. Map each adapter's managed keys to documents and
call the trio (Project/Apply/Observe) once per (document, prefix) namespace:

- **claude**
  - `.claude/settings.json`: prefix `setting.` → `pathFor` = the settings
    path (today's `current()`/Apply mapping for settings keys).
  - `.claude.json`: prefixes `mcp.`, `plugin.`, `pluginconfig.`,
    `marketplace.` → `pathFor` = `mcpServers.<name>`,
    `enabledPlugins.<source>`, `pluginConfigs.<source>`,
    `extraKnownMarketplaces.<name>`.
- **opencode**
  - `opencode.json`: prefixes `mcp.`, `setting.`.

The union of per-namespace `Project` outputs must equal today's single flat
change list; the merged Apply writes must equal today's document bytes; the
merged Observe map must equal today's `ObserveHashes` for structured keys.

### 3. Deletion of duplicated logic

After each namespace is routed through structproj and its suite is green,
delete the corresponding branch of the adapter's bespoke diff/apply/observe
code. File-projection branches stay.

## Invariants / safety

- **Behavior-preserving**: pinned by `internal/adapter/conformance` (all
  adapters) + every claude/opencode test (~2000 LOC). Any output diff is a
  migration bug, fixed in code — never by editing a test.
- **Secret safety unchanged**: `structproj` already never reads/exposes an
  on-disk value for a secret-bearing desired value and redacts `Old` of
  unknown provenance — identical to the claude loop.
- **Per-step green**: TDD/verify after each namespace; a red step blocks the
  next.

## Key risk & resolution rule

If build finds a claude/opencode structured-doc behavior with no `structproj`
equivalent (a canonicalization or adopt quirk), pause: extend `structproj`
**additively** (small, contract-preserving) or document the divergence. Never
silently change adapter behavior to fit the core.

## Out of scope

File-projection (symlinked skills/commands/subagents, inactive dirs,
copy-subagents), any `structproj` semantic change, any behavior/schema change.
Those are a separate, higher-risk change.
