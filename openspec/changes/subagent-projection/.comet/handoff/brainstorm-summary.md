# Brainstorm Summary

- Change: subagent-projection
- Date: 2026-07-10

## Confirmed Technical Approach

Replicate the archived `command-projection` pipeline for a new `subagent.*`
resource path (no premature generalization of skills/commands/subagents):

- Catalog: `catalog/subagents/<name>.md` embedded via `go:embed all:subagents`;
  optional `[subagents]` table → `Framework.Subagents`; `ExpandSubagents`
  (transitive, deduped); **verbatim** single-file materialization to
  `.homonto/catalog/subagents/<name>.md`, version-gated on the shared catalog
  version. No frontmatter rewrite, no model injection.
- Config: `ExpandedSubagentEntriesForTool(tool)` (explicit + framework-expanded,
  scope/targets inheritance, explicit-vs-framework collision = error).
- Path: new `internal/subagentpath.Dir(tool, scope, home, projectRoot)` — Claude
  `.claude/agents/` (plural, both scopes); OpenCode `~/.config/opencode/agent/`
  (user) and `<repo>/.opencode/agent/` (project, singular).
- Adapters: `subagentsDir`/`inactiveSubagentsDir`/`subagentSource`/
  `subagentLinks`, plan/apply/adopt/prune/relocate for `subagent.<name>`,
  `ObserveHashes` symlink-hash, `managedRoots()` gains subagent root (non-empty
  guard), `WithSubagentCatalogRoot` engine wiring.
- Doctor: verify subagent links + materialized files for both tools.
- Bundled content: `code-reviewer` + `codebase-explorer` (loose builtin, both
  tools) and one comet-framework subagent (via `[frameworks.comet]`, both
  tools).

Frontmatter decision: a subagent targeting both tools uses ONE verbatim file
with **minimal shared frontmatter** — `name` + `description` + `mode: subagent`,
omitting `model` and `tools` so each tool applies its own defaults. Rationale:
`model` (alias vs full id) and `tools` (comma-string vs boolean map) are the only
hard conflicts between the two schemas; omitting them is consistent with the
confirmed "no model injection / verbatim" rule. `name` (Claude-required) and
`mode` (OpenCode-wanted) are additive keys the other tool should ignore.

Minor conventions settled: sibling `internal/subagentpath` package (mirrors
`commandpath`/`skillpath`); comet-framework subagent targets both tools (matches
comet skill targeting).

## Key Trade-offs and Risks

- Parallel `subagent.*` duplicates command logic → accepted (localized, mirrors
  tested code; unify the three kinds later).
- Parser tolerance of the other tool's extra frontmatter key (`mode` in Claude,
  `name` in OpenCode) is UNDOCUMENTED → build starts with a fixture/empirical
  check (task 1.1) before wiring adapters; `subagentpath` isolates any
  correction. Claude Code guidance: unknown keys most likely silently ignored,
  must be tested.
- Authoring 3 real subagents couples content to machinery → keep each
  minimal-but-valid; tests assert linking/no-drift, not subagent behavior.
- Model validation already fires for subagent-targeted tools (existing
  `validateModels`/`EnabledModelTools`) → confirm no gap; add a test.

## Testing Strategy

Fixture-first: real Claude `agents/` + OpenCode `agent/` layouts (both scopes).
Unit: parse/expand/dedup/materialize (byte-for-byte + no-model-injection);
config expansion/collision/target-filter + model-validation check; adapters both
tools (create/idempotent/conflict/prune/relocate/adopt); doctor. Dogfood: `apply`
→ `status` (No drift) → `doctor` (both tools OK). Full regression:
`go test [-race] ./...`, `go vet`, `go build`, `gofmt -l .`.

## Spec Patches

Add one boundary scenario to `specs/subagent-projection/spec.md` under the
"Bundled real subagents" requirement: a both-tools subagent's single
minimal-frontmatter file loads validly in both Claude Code and OpenCode.
(Supplements an acceptance scenario; no structural change to the delta spec.)
