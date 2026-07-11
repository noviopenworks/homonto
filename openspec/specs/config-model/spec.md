# config-model Specification

## Purpose
Defines the `homonto.toml` desired-state model shared by adapters: MCP servers,
explicit framework/skill/command/subagent resources, per-tool plugins, per-tool
settings, target selection, model routing, and unresolved secret references.
## Requirements
### Requirement: Declarative config as single source of truth

`homonto` SHALL parse a single `homonto.toml` file into one tool-agnostic
desired-state model covering MCP servers, explicit framework/skill/command/
subagent resources, per-tool plugins, per-tool settings, and model routing. All
downstream stages SHALL operate on this model, never on raw TOML.

#### Scenario: Parse a complete config
- **WHEN** `homonto.toml` declares MCP servers, explicit resource tables
  (`[frameworks.<name>]`, `[skills.<name>]`, `[commands.<name>]`,
  `[subagents.<name>]`), per-tool `[plugins]`, per-tool `[settings]`, and needed
  `[models.<tool>.<level>]` routes
- **THEN** the loader returns a model exposing each MCP's command/env/targets,
  the resources (each with source, scope, and targets), the per-tool plugin
  lists, the per-tool settings maps, and model routes

#### Scenario: Missing config file is an error
- **WHEN** the config path does not exist
- **THEN** `Load` returns an error rather than an empty model

### Requirement: MCP target defaulting

An MCP server declared without an explicit `targets` list SHALL apply to all
supported tools; an MCP with an explicit `targets` list SHALL apply only to
those tools. Every listed target MUST name a supported tool (`claude` or
`opencode`); `config.Load` SHALL reject an unknown target name, naming the
offending value and the valid set.

#### Scenario: No targets means all tools
- **WHEN** an MCP entry omits `targets`
- **THEN** its effective targets are `["claude", "opencode"]`

#### Scenario: Explicit targets are honored
- **WHEN** an MCP entry sets `targets = ["claude"]`
- **THEN** its effective targets are exactly `["claude"]`

#### Scenario: Unknown target is rejected
- **GIVEN** an MCP entry with `targets = ["claud"]` (a typo)
- **WHEN** the config is loaded
- **THEN** `Load` returns an error naming `"claud"` and the valid targets
  `claude` and `opencode`, rather than silently projecting the MCP nowhere

### Requirement: Secret references preserved as unresolved tokens

The config model SHALL retain secret references (`${pass:…}`, `${ENV}`) verbatim
as unresolved tokens; parsing SHALL NOT resolve them.

#### Scenario: Env value with a pass reference
- **WHEN** an MCP `env` value is `"${pass:ai/brave}"`
- **THEN** the parsed model stores `"${pass:ai/brave}"` unchanged

### Requirement: Config input validation

`config.Load` SHALL reject a declared MCP that has no command, and SHALL reject
a per-tool settings key that collides with a structure homonto manages in that
tool's file — naming the offending entry in each case — so that unprojectable
or colliding config fails fast at load rather than being silently ignored at
apply.

#### Scenario: MCP without a command is rejected
- **GIVEN** an MCP entry with no `command` (or `command = []`)
- **WHEN** the config is loaded
- **THEN** `Load` returns an error naming that MCP, because an MCP with no
  command cannot be projected to any tool

#### Scenario: Reserved settings key is rejected
- **GIVEN** a `settings.claude` key `enabledPlugins`, or a `settings.opencode`
  key `mcp` or `plugin`
- **WHEN** the config is loaded
- **THEN** `Load` returns an error naming the reserved key, because homonto
  manages that structure in the same tool file

#### Scenario: Non-colliding settings keys still load
- **GIVEN** settings keys that do not collide (e.g. `settings.claude.model`, or
  `settings.opencode.enabledPlugins`, which is reserved only for claude)
- **WHEN** the config is loaded
- **THEN** `Load` accepts them

### Requirement: Explicit resource declarations

`homonto` SHALL parse frameworks, skills, commands, and subagents as explicit
per-resource tables. Every resource SHALL declare `source` and `scope`. Scope
SHALL be either `user` or `project`; there is no default. Source SHALL be either
`builtin:<name>` or `local:<name>` in the first release.

#### Scenario: Parse explicit resources
- **WHEN** `homonto.toml` declares `[skills.graphify]` with `source = "local:graphify"` and `scope = "project"`
- **THEN** the loader returns a skill resource named `graphify` with local source `graphify` and project scope

#### Scenario: Missing scope is rejected
- **WHEN** a resource omits `scope`
- **THEN** `Load` returns an error naming that resource and the missing scope

### Requirement: Tool-specific model routing

For every model-enabled target tool, `homonto.toml` SHALL define all three model
levels: `architectural`, `coding`, and `trivial`. Each route SHALL include a
non-empty `model` and at least one of `effort` or `variant`. Homonto SHALL not
validate provider-specific model names or effort values beyond presence.

#### Scenario: Model routing for one tool
- **GIVEN** a config whose only model-enabled tool is `claude`
- **WHEN** the loader parses `[models.claude.architectural]`, `[models.claude.coding]`, and `[models.claude.trivial]`, each with a `model` plus `effort` or `variant`
- **THEN** the loader accepts the config and exposes the three routes keyed by tool and level

#### Scenario: Missing model level is rejected
- **WHEN** a model-enabled tool lacks one of the three levels, or a level omits `model`, or a level has neither `effort` nor `variant`
- **THEN** `Load` returns an error naming the offending tool and level

### Requirement: Local provider content root

Local provider content SHALL live under `homonto/` relative to the directory containing `homonto.toml`; generated state, cache, and the materialized builtin catalog SHALL live under `.homonto/` only. Current adapters resolve local-source skills (`source = "local:<name>"`) from `homonto/skills/<name>`, local-source commands from `homonto/commands/<name>.md`, and local-source subagents from `homonto/subagents/<name>.md`. Builtin-source skills resolve from the materialized `.homonto/catalog/skills/<name>/`, builtin-source commands from `.homonto/catalog/commands/<name>.md`, and builtin-source subagents from `.homonto/catalog/subagents/<name>.md`. Local framework content resolution beyond these resource kinds is part of future framework/catalog projection work and MUST NOT be claimed as installed behavior yet.

#### Scenario: Local skill resolves from homonto/

- **GIVEN** a config with `[skills.my-skill] source = "local:my-skill"`
- **WHEN** apply creates the skill link
- **THEN** the symlink target is `homonto/skills/my-skill/`

#### Scenario: Builtin skill resolves from materialized catalog

- **GIVEN** a config with `[skills.brainstorming] source = "builtin:brainstorming"`
- **WHEN** apply creates the skill link
- **THEN** the symlink target is `.homonto/catalog/skills/brainstorming/`

#### Scenario: Local command resolves from homonto/commands

- **GIVEN** a config with `[commands.mine] source = "local:mine"`
- **WHEN** apply creates the command link
- **THEN** the symlink target is `homonto/commands/mine.md`

#### Scenario: Builtin command resolves from materialized catalog

- **GIVEN** a config with `[commands.demo] source = "builtin:demo"`
- **WHEN** apply creates the command link
- **THEN** the symlink target is `.homonto/catalog/commands/demo.md`

#### Scenario: Local subagent resolves from homonto/subagents

- **GIVEN** a config with `[subagents.mine] source = "local:mine"`
- **WHEN** apply creates the subagent link
- **THEN** the symlink target is `homonto/subagents/mine.md`

#### Scenario: Builtin subagent resolves from materialized catalog

- **GIVEN** a config with `[subagents.code-reviewer] source = "builtin:code-reviewer"`
- **WHEN** apply creates the subagent link
- **THEN** the symlink target is `.homonto/catalog/subagents/code-reviewer.md`

### Requirement: Bundled catalog embedded in binary

Homonto SHALL bundle a catalog directory tree at `catalog/` embedded in the Go binary via `go:embed`. The catalog SHALL contain framework metadata under `catalog/frameworks/<name>/framework.toml` and skill content under `catalog/skills/<name>/`. The embedded catalog SHALL be read-only at runtime.

#### Scenario: Catalog is available without external files

- **GIVEN** a Homonto binary built from a repo containing `catalog/`
- **WHEN** the binary runs on a machine without the source repo
- **THEN** the catalog frameworks and skills are accessible from the embedded filesystem

### Requirement: Builtin skill source resolution

A skill resource with `source = "builtin:<name>"` SHALL resolve its content from the embedded catalog at `catalog/skills/<name>/`. The content SHALL be materialized to `.homonto/catalog/skills/<name>/` on apply so that filesystem symlinks can point at a real directory.

#### Scenario: Builtin skill materializes on first apply

- **GIVEN** a config declaring `[skills.brainstorming] source = "builtin:brainstorming"`
- **WHEN** the user runs `homonto apply`
- **THEN** `.homonto/catalog/skills/brainstorming/` is created with content from the embedded catalog, and the skill is symlinked from there into the tool skills directories

#### Scenario: Builtin skill is idempotent on re-apply

- **GIVEN** a builtin skill already materialized and symlinked
- **WHEN** the user runs `homonto apply` again without any config change
- **THEN** no re-materialization occurs and the skill symlink is a no-op

### Requirement: Catalog version tracking

Homonto SHALL track the catalog version in `.homonto/state.json`. When the embedded catalog version differs from the recorded version, builtin resources SHALL be re-materialized on the next apply.

#### Scenario: Catalog upgrade triggers re-materialization

- **GIVEN** a builtin skill materialized under catalog version `0.1.0`
- **WHEN** a newer binary with catalog version `0.2.0` runs `homonto apply`
- **THEN** the builtin skill content in `.homonto/catalog/skills/<name>/` is refreshed and the state version is updated

### Requirement: Materialized catalog is generated state

The `.homonto/catalog/` directory SHALL be treated as generated cache. It SHALL NOT be committed to version control. The scaffolded `.gitignore` SHALL exclude `.homonto/` including the catalog cache.

#### Scenario: Gitignore covers catalog cache

- **GIVEN** a repo initialized with `homonto init`
- **WHEN** builtin skills are materialized to `.homonto/catalog/`
- **THEN** `git status` reports no untracked files under `.homonto/catalog/`

### Requirement: Plugin declaration model

Plugins SHALL be declared as per-tool, per-plugin tables
`[plugins.<tool>.<name>]` (tool ∈ {`claude`, `opencode`}). Each plugin table
SHALL carry:

- `source` (required, non-empty string): the tool-native plugin identifier —
  for `claude` the `name@marketplace` key used in `enabledPlugins`; for
  `opencode` the npm package name or local plugin path placed in the `plugin`
  array.
- `enabled` (optional boolean, default `true`): `false` marks the plugin
  disabled.
- `config` (optional table): non-sensitive per-plugin settings. `config` is
  supported only for `claude` plugins (projected to `pluginConfigs.<source>.options`);
  a non-empty `config` on an `opencode` plugin SHALL be rejected at load, because
  OpenCode has no per-plugin config location on disk.

The declaration name (the table key) and the `source` SHALL be validated with
the same key-validation guard applied to other config keys. A plugin whose
`source` is empty SHALL be rejected. Two plugin declarations under the same tool
sharing one `source` SHALL be rejected (their projections would collide). The
reserved-key guards SHALL be preserved and extended: `settings.claude.enabledPlugins`,
`settings.claude.pluginConfigs`, and `settings.opencode.plugin`/`mcp` remain
rejected because homonto manages those structures.

#### Scenario: Parse plugin declaration tables

- **GIVEN** a config with `[plugins.claude.claude-hud]` (`source = "claude-hud@official"`, `enabled = true`) and `[plugins.opencode.quota]` (`source = "@slkiser/opencode-quota"`, no `enabled`)
- **WHEN** the config is parsed
- **THEN** it yields a Claude plugin `claude-hud` with source `claude-hud@official` enabled, and an OpenCode plugin `quota` with source `@slkiser/opencode-quota` whose enabled defaults to true

#### Scenario: A plugin without a source is rejected

- **GIVEN** a `[plugins.claude.x]` table with no `source` (or an empty `source`)
- **WHEN** the config is parsed
- **THEN** it is rejected with an error identifying the plugin

#### Scenario: enabled defaults to true and false disables

- **GIVEN** one plugin with `enabled = false` and one with `enabled` omitted
- **WHEN** the config is parsed
- **THEN** the first is disabled and the second is enabled (default true)

#### Scenario: Reserved plugin settings keys still rejected

- **GIVEN** a `settings.claude` key `enabledPlugins` or `pluginConfigs`, or a `settings.opencode` key `plugin`
- **WHEN** the config is parsed
- **THEN** it is rejected as reserved (homonto manages plugins there)

#### Scenario: A Claude plugin config is parsed

- **GIVEN** a `[plugins.claude.hud]` with `source = "hud@official"` and `config = { api_endpoint = "https://x", max_workers = 4 }`
- **WHEN** the config is parsed
- **THEN** the plugin carries that config map

#### Scenario: An OpenCode plugin config is rejected

- **GIVEN** a `[plugins.opencode.q]` with `source = "q"` and a non-empty `config`
- **WHEN** the config is parsed
- **THEN** it is rejected with an error explaining OpenCode has no per-plugin config
