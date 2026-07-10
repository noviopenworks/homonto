# subagent-projection Specification

## Purpose
TBD - created by archiving change subagent-projection. Update Purpose after archive.
## Requirements
### Requirement: Builtin and local subagent source resolution

A subagent resource SHALL resolve its content by source scheme: `[subagents.<name>] source = "builtin:<name>"` resolves from the embedded catalog at `catalog/subagents/<name>.md` (materialized to `.homonto/catalog/subagents/<name>.md` on apply), and `source = "local:<name>"` resolves from `homonto/subagents/<name>.md`. Subagents are single Markdown files, not directories. Every subagent resource SHALL declare a `scope` (`user` or `project`) exactly as skills and commands do.

#### Scenario: Builtin subagent resolves from materialized catalog

- **GIVEN** a config with `[subagents.code-reviewer] source = "builtin:code-reviewer"` and `scope = "user"`
- **WHEN** apply runs
- **THEN** `catalog/subagents/code-reviewer.md` is materialized to `.homonto/catalog/subagents/code-reviewer.md` and the subagent link targets that file

#### Scenario: Local subagent resolves from homonto/subagents

- **GIVEN** a config with `[subagents.mine] source = "local:mine"` and `scope = "project"`
- **WHEN** apply creates the subagent link
- **THEN** the symlink target is `homonto/subagents/mine.md`

### Requirement: Single-file verbatim subagent materialization

Homonto SHALL materialize builtin subagent content as single files from the embedded catalog to `.homonto/catalog/subagents/<name>.md` before creating subagent symlinks, version-gated on the same catalog version tracked in state as skills and commands. The materialized file SHALL be byte-for-byte identical to the embedded catalog source: Homonto SHALL NOT rewrite the subagent's frontmatter, and SHALL NOT inject a resolved model route into the projected file. Re-materialization SHALL occur only when the catalog version changes or the target file is missing, and the catalog version SHALL be recorded only after a successful materialization.

#### Scenario: First subagent materialization

- **GIVEN** no `.homonto/catalog/subagents/code-reviewer.md` exists
- **WHEN** apply runs with a config declaring a builtin subagent `code-reviewer`
- **THEN** `.homonto/catalog/subagents/code-reviewer.md` is written byte-for-byte from the embedded catalog

#### Scenario: Version-gated subagent skip

- **GIVEN** `.homonto/catalog/subagents/code-reviewer.md` exists and state records the current catalog version
- **WHEN** apply runs again with the same binary
- **THEN** the subagent is not re-materialized and the link is a no-op

#### Scenario: Model route is not injected

- **GIVEN** a config whose `[models.<tool>.<level>]` routes are defined and a builtin subagent is declared
- **WHEN** apply materializes and links the subagent
- **THEN** the projected file's content equals the catalog source and contains no Homonto-injected model value

### Requirement: Subagent projection into tool agent directories

Owned subagents SHALL be linked (not copied) into each tool's agent directory at the location chosen by the resource's `scope`: Claude Code at `~/.claude/agents/<name>.md` (user) or `<repo>/.claude/agents/<name>.md` (project), and OpenCode at `~/.config/opencode/agent/<name>.md` (user) or `<repo>/.opencode/agent/<name>.md` (project). Claude Code uses the plural `agents/` directory and OpenCode uses the singular `agent/` directory. Pending link work SHALL appear as plan changes (create / update / no-op). `apply` SHALL record each applied subagent link in state and SHALL prune a de-declared subagent's link only when it is a symlink pointing into a homonto-managed root (`homonto/subagents/` or `.homonto/catalog/subagents/`); a real file or foreign link SHALL be reported as a conflict and never clobbered. A per-resource `scope` switch SHALL appear as a relocation that removes the old-scope link as it creates the new one.

#### Scenario: Builtin subagent links into both tools

- **GIVEN** a config with `[subagents.code-reviewer] source = "builtin:code-reviewer"` targeting claude and opencode
- **WHEN** apply runs
- **THEN** `~/.claude/agents/code-reviewer.md` and `~/.config/opencode/agent/code-reviewer.md` are symlinks into `.homonto/catalog/subagents/code-reviewer.md`

#### Scenario: Idempotent subagent link

- **WHEN** a subagent link already points at its materialized target
- **THEN** plan reports no change and a second apply is a no-op

#### Scenario: Conflict is reported, not clobbered

- **GIVEN** a real file already exists at the subagent's link destination
- **THEN** apply reports a conflict and leaves the existing file untouched

#### Scenario: De-declared subagent pruned only when it is our link

- **GIVEN** a subagent removed from `homonto.toml` whose link is a symlink into a homonto-managed root
- **WHEN** apply processes the delete
- **THEN** the link is removed; a real file or foreign link at that path is instead reported as a conflict and left untouched

#### Scenario: Scope switch relocates the link

- **GIVEN** a subagent whose `scope` changes from `user` to `project` (or the reverse)
- **WHEN** apply runs
- **THEN** the plan shows a relocation and apply removes the old-scope link while creating the new-scope link

### Requirement: Subagent adoption of pre-existing matching links

A correct-but-unrecorded subagent link — one already on disk pointing at its materialized or local content but absent from (or stale in) state — SHALL be adopted into state without rewriting the on-disk link, exactly as skill and command links are adopted, so a lost `state.json` can be rebuilt without a spurious change.

#### Scenario: Adopt an already-correct subagent link

- **GIVEN** a subagent link on disk that already points at its content but is not recorded in state
- **WHEN** apply runs
- **THEN** the link is left untouched and its record is added to state as an adoption (no create/update)

### Requirement: Framework subagent expansion

A `framework.toml` `[subagents]` table SHALL expand through `[frameworks.<name>] source = "builtin:<framework>"` into effective subagent resources with `source = "builtin:<subagent-name>"`, each inheriting the framework declaration's `scope` and `targets`, transitively across dependency frameworks and deduplicated by name, exactly as skills and commands expand. A subagent name colliding with an explicit `[subagents.X]` entry SHALL be a config error.

#### Scenario: Framework expands its subagents

- **GIVEN** `[frameworks.comet] source = "builtin:comet"` where the comet framework declares a subagent in its `[subagents]` table
- **WHEN** the config is loaded
- **THEN** the effective subagent set includes that subagent as a builtin-source subagent inheriting the framework's scope and targets

### Requirement: Subagent link doctor verification

`doctor` SHALL verify each recorded subagent link: a builtin subagent's materialized target under `.homonto/catalog/subagents/` SHALL exist, and the tool-side symlink SHALL be present and point at the expected source; a missing materialized file or broken link SHALL be reported like a broken skill or command link, for both Claude Code and OpenCode.

#### Scenario: Doctor reports a linked subagent

- **GIVEN** a builtin subagent materialized and linked into a tool
- **WHEN** `doctor` runs
- **THEN** it reports the subagent link as present and correct for that tool

### Requirement: Bundled real subagents

The first release of this capability SHALL ship real subagent content in `catalog/subagents/`, not only a placeholder: at least `code-reviewer` and `codebase-explorer` as framework-agnostic loose builtin subagents, plus one subagent declared in the `comet` framework's `[subagents]` table so framework-declared subagent expansion is exercised with real content. Each bundled subagent SHALL be a valid single-file agent definition for the tools it targets.

#### Scenario: Loose builtin subagents are projectable

- **GIVEN** the bundled catalog containing `code-reviewer` and `codebase-explorer`
- **WHEN** each is declared as `[subagents.X] source = "builtin:X"` and applied
- **THEN** it materializes and links into its targeted tools with no drift on a second status

#### Scenario: Comet framework subagent expands and projects

- **GIVEN** `[frameworks.comet]` enabled and the comet framework declaring a subagent in `[subagents]`
- **WHEN** apply runs
- **THEN** that subagent is materialized and linked into the framework's targeted tools alongside comet's skills

#### Scenario: Shared minimal frontmatter is valid for both tools

- **GIVEN** a bundled subagent targeting both Claude Code and OpenCode whose single file carries only minimal shared frontmatter (`name`, `description`, `mode: subagent`) and omits `model` and `tools`
- **WHEN** the same materialized file is linked into both tools
- **THEN** it is a valid agent definition for each tool, and each tool applies its own default model and tool set (no Homonto-injected model or tools)
