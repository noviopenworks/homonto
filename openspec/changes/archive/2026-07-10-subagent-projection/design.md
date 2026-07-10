## Context

Homonto projects skills and commands from a bundled `go:embed` catalog into
Claude Code and OpenCode through a proven foundation: version-gated
materialization to `.homonto/catalog/<kind>/`, scope-aware symlinking via
`internal/link` (multi-root, conflict-safe), per-resource scope with
relocation, adopt/prune, and `doctor` verification. `[subagents.X]` already
parses in `internal/config` and already participates in `validateModels` (any
tool a subagent targets must define all three `[models.<tool>.<level>]`
levels), but no adapter/engine/plan step projects it — subagents are the last
declared resource kind that is parsed and then ignored at apply.

The immediately preceding `command-projection` change established the exact
pattern this change follows. The deep technical design is refined further in
the Comet Design phase; this document fixes the high-level architecture.

## Goals / Non-Goals

**Goals:**

- Make `[subagents.X]` functional: materialize + symlink declared subagents into
  Claude Code (`agents/`) and OpenCode (`agent/`), scope-aware, with doctor
  verification and adopt/prune, reusing the skills/commands foundation.
- Expand framework-declared `[subagents]` tables transitively, deduplicated, with
  explicit-entry collision as a config error.
- Ship three real bundled subagents (`code-reviewer`, `codebase-explorer`, and a
  comet-framework subagent) so the machinery carries genuine content.
- Materialize verbatim — no frontmatter rewriting, no model injection.

**Non-Goals:**

- The `onto` binary (separate release-blocking work).
- Injecting resolved model routes into subagent frontmatter; per-subagent model
  overrides.
- Namespaced subagents (`<ns>:<name>`); flat `<name>` only.
- Remote/registry subagent sources.
- Changing skills/commands behavior or the existing model-route validation.

## Decisions

**D1 — Mirror the command pipeline, do not generalize it yet.** Add a parallel
`subagent.*` path rather than refactoring skills/commands/subagents into one
generic resource loop. Rationale: the command pattern is fresh, well-tested, and
low-risk to replicate; a premature generalization would touch the working
skills/commands paths. A later change may unify the three once all three exist.
Alternative (generic resource abstraction now) rejected as scope creep that
risks regressions in shipped behavior.

**D2 — Single-file verbatim materialization.** Reuse the command single-file
model: `catalog/subagents/<name>.md` → `.homonto/catalog/subagents/<name>.md`,
byte-for-byte, version-gated on the same catalog version. No `RemoveAll` needed
(single-file overwrite). Rationale: subagents are single Markdown files with
frontmatter, like commands; a symlink to verbatim content keeps edits live and
avoids per-tool file rewriting. Alternative (resolve model route into
frontmatter at apply) rejected: it breaks the symlink-clean model, forks
per-tool content, and edges into the per-subagent-model non-goal.

**D3 — Per-tool agent directory naming.** Claude Code uses `agents/` (plural);
OpenCode uses `agent/` (singular), consistent with OpenCode's singular
`command/`. Encode this in a path helper (`internal/subagentpath` or an
extension of `commandpath`) mapping `(tool, scope) → dir`, so the singular/plural
split lives in one place. The exact real-layout directories are confirmed by
fixtures in build (see Risks). User scope: `~/.claude/agents/`,
`~/.config/opencode/agent/`. Project scope: `<repo>/.claude/agents/`,
`<repo>/.opencode/agent/`.

**D4 — Framework `[subagents]` table.** Extend `framework.toml` parsing and
`ExpandSubagents` exactly like `[commands]`: inherit framework `scope`/`targets`,
transitive across dependencies, dedupe by name, explicit-entry collision is an
error. The `comet` framework's `framework.toml` gains a `[subagents]` entry
pointing at a real bundled subagent so expansion is exercised end-to-end.

**D5 — Real content, not a placeholder.** Unlike `command-projection` (one
placeholder), ship `code-reviewer` and `codebase-explorer` as loose builtin
subagents and one comet-framework subagent. `code-reviewer` and
`codebase-explorer` are declared standalone in `homonto.toml` for dogfood; the
comet subagent is exercised via `[frameworks.comet]` expansion. Each is authored
as a valid single-file agent definition (frontmatter + body) usable by the tools
it targets.

**D6 — State keys and reuse.** New `subagent.<name>` state keys, handled in
Plan/Apply/ObserveHashes identically to `command.<name>` (symlink hash
`Hash(dst + " -> " + src)`, adopt, orphan prune, scope-switch relocate).
`internal/link` and `internal/state` (catalog version) are reused unchanged;
`managedRoots()` gains the subagent catalog root only when set.

## Risks / Trade-offs

- **Exact tool agent file format/layout not yet fixture-confirmed** → Build
  starts by adding real-layout fixtures for Claude `agents/` and OpenCode
  `agent/` (mirroring the skills/commands fixtures) and asserts projection
  against them before wiring adapters; the path helper isolates any correction to
  one place.
- **Authoring three real subagents couples content to machinery** → Keep each
  subagent minimal but valid; the projection tests assert linking/no-drift, not
  subagent behavior, so content quality does not gate the machinery.
- **Parallel `subagent.*` code duplicates command logic** → Accepted per D1;
  duplication is localized and mirrors existing tested code. Unification is a
  deliberate later step.
- **Model validation already fires for subagent-targeted tools** → No change
  needed; verify existing `validateModels`/`EnabledModelTools` already counts
  subagents so enabling one without model routes fails clearly (add a test if a
  gap exists).

## Migration Plan

Additive only. New catalog tree, new state-key prefix, new adapter/doctor
branches; no changes to existing skill/command/MCP/settings behavior. Rollback is
removing the declarations and applying (prunes the links) or reverting the
change. Dogfood: declare the two loose subagents (and keep `[frameworks.comet]`),
`apply`, then confirm `status` → `No drift` and `doctor` reports both tools'
subagent links OK.

## Open Questions

- Confirm OpenCode's project-scope agent directory is `<repo>/.opencode/agent/`
  (singular) and Claude's is `<repo>/.claude/agents/` (plural) against real tool
  layout fixtures during build.
- Whether the comet-framework subagent should target both tools or Claude only
  for the first release (default: match how comet skills are targeted).
