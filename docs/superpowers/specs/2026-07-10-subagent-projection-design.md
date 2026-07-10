---
comet_change: subagent-projection
role: technical-design
canonical_spec: openspec
---

# Subagent Projection — Technical Design

Deep technical refinement of the open-phase `design.md` for the
`subagent-projection` change. The OpenSpec delta specs
(`specs/subagent-projection`, `specs/config-model`,
`specs/framework-expansion`) remain the canonical WHAT; this document is the
canonical HOW.

## Context

Homonto projects skills and commands from a bundled `go:embed` catalog into
Claude Code and OpenCode through a proven foundation: version-gated single-file
or directory materialization to `.homonto/catalog/<kind>/`, scope-aware
symlinking via `internal/link` (multi-root, conflict-safe, adopt/prune), and
`doctor` verification. The archived `command-projection` change is the direct
precedent — subagent projection follows its shape almost exactly.

`[subagents.X]` already parses into `Config.Subagents map[string]Resource` and
already participates in `validateModels` (any tool a subagent targets must
define all three `[models.<tool>.<level>]` levels), but nothing materializes or
links it. Subagents are the last declared resource kind that is parsed and then
ignored at apply.

## Goals / Non-Goals

**Goals**

- Materialize + symlink declared subagents into Claude Code (`agents/`) and
  OpenCode (`agent/`), scope-aware, adopt/prune/relocate, doctor-verified,
  reusing the skills/commands foundation.
- Framework-declared `[subagents]` expansion (transitive, deduped, collision =
  error).
- Ship three real bundled subagents (`code-reviewer`, `codebase-explorer`, one
  comet-framework subagent).
- Verbatim materialization — no frontmatter rewrite, no model injection.

**Non-Goals**

- The `onto` binary; per-subagent model overrides / model-into-frontmatter;
  namespaced subagents; remote sources; changing skills/commands behavior or the
  existing model-route validation; per-tool subagent source variants.

## Decisions

### D1 — Mirror the command pipeline; do not generalize yet

Add a parallel `subagent.*` path rather than refactoring skills/commands/
subagents into one generic resource loop. The command pattern is fresh,
well-tested, and cheap to replicate; a premature abstraction would touch the
working skills/commands paths and risk regressions. Unification of the three
kinds is a deliberate later change once all three exist.

### D2 — Single-file verbatim materialization

`catalog/subagents/<name>.md` → `.homonto/catalog/subagents/<name>.md`,
byte-for-byte, version-gated on the same catalog version tracked in state. Reuse
the command single-file model (`MaterializeSubagents`, no `RemoveAll` needed — a
single-file overwrite fully replaces prior content). Homonto never rewrites the
subagent's frontmatter and never injects a resolved model route. Alternative
(resolve model into frontmatter at apply) rejected: breaks the symlink-clean
model, forks per-tool content, and edges into the per-subagent-model non-goal.

### D3 — Per-tool agent directory naming via `internal/subagentpath`

Claude Code uses `agents/` (plural) at both scopes; OpenCode uses `agent/`
(singular), consistent with its singular `command/`. A new sibling package
`internal/subagentpath` (mirroring `commandpath`/`skillpath`) maps
`(tool, scope) → dir`, isolating the singular/plural split in one place:

| tool | user scope | project scope |
|---|---|---|
| claude | `~/.claude/agents/` | `<repo>/.claude/agents/` |
| opencode | `~/.config/opencode/agent/` | `<repo>/.opencode/agent/` |

The exact real-layout directories are confirmed by fixtures at the start of
build (see Risks). A sibling package (not extending `commandpath`) keeps each
resource kind's path logic single-responsibility and matches the existing split.

### D4 — Framework `[subagents]` table

Extend `framework.toml` parsing (`Framework.Subagents`, name →
`subagents/<name>.md`, path validated against the embedded FS) and add
`ExpandSubagents` exactly like `ExpandCommands`: inherit framework
`scope`/`targets`, transitive across dependencies, dedupe by name, explicit-entry
collision is a config error. The `comet` framework's `framework.toml` gains a
`[subagents]` entry pointing at a real bundled subagent so expansion is exercised
end-to-end (both tools, matching comet skill targeting).

### D5 — Real content with minimal shared frontmatter

Unlike `command-projection` (one placeholder), ship `code-reviewer` and
`codebase-explorer` as loose builtin subagents plus one comet-framework
subagent. Each targets both tools, so each is a single verbatim file whose
frontmatter is the **minimal shared subset** valid for both parsers:

```yaml
---
name: <name>            # required by Claude Code; ignored/derived by OpenCode
description: <one line> # required by both
mode: subagent          # OpenCode agent mode; unknown-but-ignored key for Claude
---
<system-prompt body>
```

`model` and `tools` are **omitted**: they are the only hard-conflicting fields
(Claude `model` is an alias/`inherit`; OpenCode `model` is a full
`provider/model` id; Claude `tools` is a comma-string; OpenCode `tools` is a
boolean map). Omitting them makes Claude fall back to `inherit` and OpenCode to
its defaults — consistent with the verbatim/no-model-injection rule and a stated
non-goal (per-subagent model overrides). Authoritative field facts:

- Claude Code (`.claude/agents/*.md`): required `name`, `description`; optional
  `tools` (comma-string/array, omit → inherits all), `model` (alias/full-id/
  `inherit`, default `inherit`). Unknown-key behavior is undocumented but most
  likely silently ignored — must be verified (Risks).
- OpenCode (`.opencode/agent/*.md`): required `description`; `mode`
  (`primary`|`subagent`|`all`); optional `model` (full id), `tools` (map),
  `temperature`. `name` is derived from the filename.

Alternative (per-tool source files `<name>.claude.md` / `<name>.opencode.md`)
rejected: breaks the uniform single-file model and complicates catalog indexing,
materialization, linking, state keys, and doctor for richness that is a non-goal.

### D6 — State keys and reuse

New `subagent.<name>` state keys, handled in Plan/Apply/ObserveHashes identically
to `command.<name>`: symlink hash `Hash(dst + " -> " + src)`, adopt of a
correct-but-unrecorded link, orphan prune only when the link points into a
managed root, and scope-switch rendered as a relocation. `internal/link` and
`internal/state` (catalog version) are reused unchanged; `managedRoots()` gains
the subagent catalog root only when set (empty-root guard preserved).

## Component Boundaries

| Unit | Responsibility | Depends on |
|---|---|---|
| `internal/subagentpath` | `(tool, scope) → agent dir` | — |
| `internal/catalog` | embed, `Framework.Subagents` parse, `ExpandSubagents`, `MaterializeSubagents` (verbatim) | embedded FS |
| `internal/config` | `ExpandedSubagentEntriesForTool`, collision/cycle | catalog |
| `internal/engine` | orchestrate subagent materialization + `WithSubagentCatalogRoot` | catalog, adapters |
| `internal/adapter/{claude,opencode}` | plan/apply/adopt/prune/relocate `subagent.*` links | subagentpath, link, state |
| `internal/engine/status.go` (doctor) | verify subagent links + materialized files, both tools | state, subagentpath |
| `catalog/subagents/*` | three real bundled subagents | — |

## Risks / Trade-offs

- **Parser tolerance of the other tool's extra frontmatter key** (`mode` in
  Claude, `name` in OpenCode) is undocumented → **Mitigation:** build task 1.1
  adds real-layout fixtures and an empirical load check for both tools before
  adapters are wired; `subagentpath` and the single-file frontmatter isolate any
  correction. If a tool rejects the other's key, fall back to the smallest
  frontmatter that tool requires (Claude: `name` + `description`; OpenCode:
  `description` + `mode`) and, only if strictly necessary, revisit the per-tool
  file alternative — flagged as a design finding, not silently.
- **Parallel `subagent.*` duplicates command logic** → Accepted (D1); localized,
  mirrors tested code.
- **Authoring three real subagents couples content to machinery** →
  **Mitigation:** keep each minimal-but-valid; projection tests assert
  link/no-drift, not subagent behavior.
- **Model validation already fires for subagent-targeted tools** → No change
  needed; add a test asserting a subagent enabling a tool without model routes
  fails clearly, closing any gap in `EnabledModelTools`.

## Testing Strategy

Fixture-first, then bottom-up, ending in dogfood + full regression:

1. `subagentpath` unit tests (all tool/scope combos; singular/plural assertion).
2. Real-layout fixtures for Claude `agents/` and OpenCode `agent/` (both
   scopes) + empirical both-tools frontmatter load check.
3. Catalog: table parse, `ExpandSubagents` (transitive/dedup), verbatim
   single-file materialize (byte-for-byte, missing-file re-materialize,
   version-gated skip).
4. Config: expansion/inheritance/collision/target-filter; model-validation gap
   test.
5. Engine: first-apply materialization, version-gated skip, missing-file
   refresh; `WithSubagentCatalogRoot` wiring.
6. Adapters (both tools): create, idempotent re-apply, conflict-not-clobbered,
   de-declared prune, scope-switch relocate, adopt pre-existing link,
   `subagent.<n>` recorded.
7. Doctor: linked builtin subagent verified for both tools.
8. Dogfood: declare the two loose subagents (+ `[frameworks.comet]`),
   `apply --yes`, then `status` (No drift) and `doctor` (both tools OK).
9. Regression: `go test ./... -count=1`, `go test -race ./...`, `go vet ./...`,
   `go build ./...`, `gofmt -l .`.

## Migration Plan

Additive only: new catalog tree, new state-key prefix, new adapter/doctor
branches; no change to existing skill/command/MCP/settings behavior. Rollback is
removing the declarations and applying (prunes the links) or reverting the
change.

## Open Questions

None blocking. The tool-layout/frontmatter-tolerance confirmation is scheduled as
the first build task (fixtures) rather than left open.
