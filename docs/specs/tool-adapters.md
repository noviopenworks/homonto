# tool-adapters Specification

## Purpose
Defines how Claude Code and OpenCode adapters project the shared config model
into tool-specific files, symlink owned skills, preserve unmanaged values, prune
state-recorded removals, and handle real tool schemas safely.
## Requirements

### Requirement: Surgical merge preserves unmanaged keys

Each adapter SHALL write only the keys homonto manages and SHALL preserve all
unmanaged keys already present in a tool's file. A tool file that cannot be parsed
SHALL cause that adapter to abort and report, never to overwrite.

#### Scenario: Unmanaged keys survive apply
- **WHEN** a tool file contains keys homonto does not manage
- **THEN** those keys remain present with their values intact after apply;
  formatting and comments may be normalized by JSON/JSONC rewriting

#### Scenario: Unparseable file is not clobbered
- **WHEN** an existing tool file cannot be parsed
- **THEN** that adapter aborts and reports and does not write the file, while
  other tools still proceed

### Requirement: Claude Code projection

The Claude adapter SHALL project MCP servers into `~/.claude.json`
(`mcpServers.<name>`) and settings and plugins into `~/.claude/settings.json` — always at
the user's home, independent of skill scope. It SHALL link each owned skill as a symlink
under a skills directory selected by that skill resource's `scope`: `~/.claude/skills/`
for `user` scope and `<project>/.claude/skills/` for `project` scope, where `<project>` is
the directory of `homonto.toml`. Local-source skills (`source = "local:<name>"`) SHALL be
linked from `homonto/skills/<name>`; builtin-source skills SHALL be linked from the
bundled catalog path for that source.

#### Scenario: MCP and setting projected surgically
- **WHEN** apply runs with an MCP targeting claude and a claude setting
- **THEN** `mcpServers.<name>` is written to `~/.claude.json` and the setting to
  `~/.claude/settings.json`, with pre-existing unmanaged keys in both files intact

#### Scenario: Project scope links skills under the project root
- **GIVEN** a config with `[skills.<name>] scope = "project"`
- **WHEN** apply runs
- **THEN** the skill symlink is created under `<project>/.claude/skills/<name>` and nothing
  is added under `~/.claude/skills/`, while `~/.claude.json` and `~/.claude/settings.json`
  remain the projection targets for MCPs and settings

### Requirement: OpenCode projection

The OpenCode adapter SHALL project MCP servers into `opencode.jsonc`
(`mcp.<name>` with `type:"local"`, `command`, `enabled`, and `environment` when
env is set), settings as top-level keys, and plugins appended to the `plugin` array —
always at the user's home, independent of skill scope. It SHALL link each owned skill as a
symlink under a skills directory selected by that skill resource's `scope`:
`~/.config/opencode/skills/` for `user` scope and `<project>/.opencode/skills/` for
`project` scope, where `<project>` is the directory of `homonto.toml`. JSONC input SHALL be
normalized before editing; when homonto writes `opencode.jsonc`, all comments in that file
are removed by whole-document JSONC standardization.

#### Scenario: MCP projected with local shape and plugin appended
- **WHEN** apply runs with an MCP targeting opencode and an opencode plugin
- **THEN** `mcp.<name>.type` is `local` with the command, and the plugin is
  appended to the existing `plugin` array without duplicating existing entries

#### Scenario: Existing JSONC keys preserved
- **WHEN** `opencode.jsonc` has an unmanaged key and a comment
- **THEN** the unmanaged key survives after apply, but the comment is removed if
  the file is rewritten

#### Scenario: Project scope links skills under the project root
- **GIVEN** a config with `[skills.<name>] scope = "project"`
- **WHEN** apply runs
- **THEN** the skill symlink is created under `<project>/.opencode/skills/<name>` and
  nothing is added under `~/.config/opencode/skills/`

### Requirement: Owned content linked by symlink with conflict detection

Owned skills SHALL be linked (not copied) from their source into each tool's
skills directory at the location chosen by the skill resource's `scope`, and
pending link work SHALL be visible as plan changes: a missing link appears as a
create, a link pointing at the wrong target appears as an update, and a correct
link is a no-op. Local-source skills (`source = "local:<name>"`) SHALL be
linked from `homonto/skills/<name>`; builtin-source skills SHALL be linked from
the bundled catalog path for that source. `apply` SHALL create the links even
when they are the only pending changes, and SHALL record each applied link in
state (`skill.<name>`: desired target path plus applied hash) so drift detection
and pruning both see it. A skill removed from the config SHALL have its link
pruned only when the existing path is a symlink pointing into homonto's managed
content directory. If the target already exists and is not homonto's link, the
adapter SHALL report a conflict and SHALL NOT clobber it — for creation and for
pruning alike.

#### Scenario: Idempotent link creation

- **WHEN** a skill symlink does not yet exist
- **THEN** plan lists a create for that link, apply creates it, and a second
  plan/apply reports no change for that link

#### Scenario: Skills-only config still applies

- **GIVEN** a config whose only content is owned skills declared as explicit
  `[skills.<name>]` resources
- **WHEN** the user runs `homonto apply` and confirms
- **THEN** the plan shows one create per missing link and apply creates
  every link (it does not short-circuit as "no changes")

#### Scenario: Relative local content dir yields absolute link targets

- **GIVEN** homonto invoked from any working directory with the default
  `homonto/` local provider root
- **WHEN** apply creates skill links
- **THEN** every symlink target is an absolute path resolved against the
  config file's directory, and the link does not dangle

#### Scenario: Conflict is reported, not clobbered

- **WHEN** the link target exists as a real file or points elsewhere
- **THEN** apply reports a conflict and leaves the existing file untouched

#### Scenario: Applied link recorded in state

- **WHEN** apply creates a skill link
- **THEN** state contains a `skill.<name>` record, and `homonto status`
  reports drift if the link is later changed out-of-band

#### Scenario: De-declared skill pruned only when it is our link

- **GIVEN** a skill resource removed from `homonto.toml` whose target path is a
  real file (or a symlink pointing outside homonto's local provider root)
- **WHEN** apply processes the resulting delete
- **THEN** the path is left untouched and a conflict is reported; only a
  symlink into homonto's managed content is removed

### Requirement: Claude MCP schema fidelity

The Claude adapter SHALL emit each MCP server in the schema real Claude
Code writes: `type` `"stdio"`, `command` as a **string** naming the
executable, `args` as the remaining argv array (omitted when empty), and
`env` omitted when empty — matching the output of `claude mcp add`.
Schema conformance SHALL be asserted by fixtures taken from real tool
files, not from homonto's own round-trips.

#### Scenario: Real-schema emission

- **GIVEN** an MCP whose command is `["npx", "-y", "some-server"]`
- **WHEN** apply projects it into `~/.claude.json`
- **THEN** `mcpServers.<name>` contains `"type": "stdio"`,
  `"command": "npx"` (a string), and `"args": ["-y", "some-server"]`

#### Scenario: Legacy array shape self-heals

- **GIVEN** an on-disk `mcpServers` entry in the old all-in-`command`
  array shape
- **WHEN** `homonto plan` runs
- **THEN** the entry is reported as an update to the real schema, and the
  next apply rewrites it

### Requirement: Declarative pruning

State-recorded keys (prefixes `mcp.`, `setting.`, `plugin.`, `skill.`) SHALL
be planned as deletes when absent from the desired config and, on apply,
removed from the tool files and from state. Keys on disk that were never
recorded in state SHALL never be deleted.

#### Scenario: Removed MCP pruned

- **GIVEN** an MCP that was applied (recorded in state) and is then
  removed from `homonto.toml`
- **WHEN** the user runs plan and confirms apply
- **THEN** the plan shows a delete for that key, apply removes it from the
  tool file, and state no longer contains it

#### Scenario: Drift not mistaken for orphan

- **GIVEN** a key present on disk but absent from both the config and
  state (the user's own, or drifted in out-of-band)
- **WHEN** plan and apply run
- **THEN** the key is neither planned as a delete nor removed — pruning
  keys off state records only

### Requirement: Injection-safe key handling

Dynamic path segments SHALL be escaped per the JSON-path library's rules —
every MCP name, setting key, plugin name, and skill name used in a read or
write path — so names containing `.`, `*`, `?`, or `\` land as single
literal keys and converge on re-apply.

#### Scenario: Dotted name lands as a literal key

- **GIVEN** an MCP named `a.b`
- **WHEN** apply projects it
- **THEN** the tool file gains one literal key `"a.b"` (not a nested
  `a` → `b` object), and a second plan reports it as a noop

### Requirement: Deterministic plan output

Two consecutive plans SHALL render identically over an unchanged config
and unchanged disk state; adapters SHALL sort desired keys and delete keys
before emitting changes.

#### Scenario: Consecutive plans are byte-identical

- **GIVEN** any config with multiple MCPs, settings, and skills
- **WHEN** `homonto plan` runs twice with nothing changed in between
- **THEN** the two rendered outputs are identical

### Requirement: Adapters adopt pre-existing matching keys

Each adapter SHALL, on apply, record in state a declared non-secret key that is
present on disk, equal to its desired value, and absent from state — rather than
leaving it untracked — so that pruning and drift detection both see it. The
claude and opencode adapters SHALL behave identically in this respect, including
opencode plugins recorded by array membership. Adoption SHALL NOT modify the
tool file (the on-disk value already matches desired) and SHALL never apply to
secret-bearing keys.

#### Scenario: Claude adopts a pre-existing MCP

- **GIVEN** an MCP declared for claude whose `~/.claude.json` entry already
  equals the desired projection and which is absent from state
- **WHEN** apply runs
- **THEN** state gains an `mcp.<name>` record for claude, `~/.claude.json` is
  left byte-unchanged, and a later removal of that MCP from config prunes it

#### Scenario: OpenCode adopts a pre-existing setting and plugin

- **GIVEN** an opencode setting and an opencode plugin already present in
  `opencode.jsonc` matching desired, both absent from state
- **WHEN** apply runs
- **THEN** state gains `setting.<key>` and `plugin.<name>` records for opencode,
  `opencode.jsonc` is left byte-unchanged (its comments preserved, because
  adoption writes no tool file), and both become pruneable on later removal
  from config

### Requirement: Per-resource skill scope and relocation

Each owned skill's link destination SHALL be selected by that skill resource's `scope`:
`user` scope links under the user's home tool directory, `project` scope under the project
root (the directory of `homonto.toml`). MCP servers and settings are unaffected by scope.
When a skill's location changes because its `scope` was switched, each adapter SHALL
relocate the link rather than orphan it: `plan` renders the move as a single relocate
change for `skill.<name>` (old location → new location), and `apply` removes the managed
symlink at the now-inactive scope location and creates it at the active one. This
inactive-location removal — including when a skill is de-declared and its scope switched
in the same apply — SHALL follow the pruning conflict rule: only a symlink pointing into
homonto's managed content directory is removed, an absent path is a no-op, and a real file
or foreign link is left untouched.

#### Scenario: Switching scope relocates the link
- **GIVEN** a skill applied under `user` scope (linked at the home location) whose config is
  then changed to `[skills.<name>] scope = "project"`
- **WHEN** the user runs plan and confirms apply
- **THEN** plan shows a relocate for `skill.<name>` from the home location to
  `<project>/.claude/skills/<name>` (and the OpenCode equivalent), apply creates the
  project-location link and removes the home-location link, and a second plan reports no
  change

#### Scenario: Relocation prune only touches homonto's own link
- **GIVEN** a scope switch where the inactive-scope path holds a real file or a foreign
  symlink (not homonto's link into managed content)
- **WHEN** apply processes the relocation
- **THEN** that path is left untouched and is not removed — the prune removes only a symlink
  pointing into homonto's managed content directory, and an absent path is a no-op

#### Scenario: De-declaring a skill while switching scope leaves no orphan
- **GIVEN** a skill applied at one scope that is then, in a single apply, both removed from
  `homonto.toml` and had its `scope` switched (so its link physically sits at the
  now-inactive scope)
- **WHEN** apply processes the delete
- **THEN** the link is removed from the location it actually occupies — the delete prunes
  both the active and the (managed) inactive scope location — leaving no orphan; a foreign
  file at either location is left untouched

### Requirement: Skill links are adopted like other managed keys

Each adapter SHALL, on apply, record in state a correct-but-unrecorded skill link — one whose
symlink already exists and points at the owned content but which is absent from state (or whose
recorded hash is stale) — rather than leaving it untracked, mirroring MCP/setting/plugin
adoption. Adoption SHALL NOT modify the link on disk, and it SHALL make apply run (via the
adoption path) so that a lost `state.json` is rebuilt for a skills-only config and the link
remains prunable and drift-detectable afterward.

#### Scenario: Correct-but-unrecorded skill link is adopted
- **GIVEN** an owned skill whose link is already correct on disk but whose `skill.<name>` state
  record is missing (e.g. `.homonto/state.json` was deleted) — even in a skills-only config
- **WHEN** the user runs apply
- **THEN** the link is left unchanged on disk, state regains the `skill.<name>` record, and a
  subsequent removal of that skill from config prunes the link
