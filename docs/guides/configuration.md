# Configuration reference — `homonto.toml`

One file, parsed into a tool-agnostic desired state. All sections are optional;
an empty config is valid (and projects nothing). `homonto plan` / `apply` /
`status` / `doctor` / `import` accept `--config <path>` (default
`homonto.toml`).

Quick map of every table:

| Table | Declares | Reference |
|---|---|---|
| `[mcps.<name>]` | MCP servers | [MCP servers](#mcp-servers--mcpsname) |
| `[skills.<name>]` | Skills (symlinked) | [Skills](#skills--skillsname) |
| `[commands.<name>]` | Slash commands | [Commands](#commands--commandsname) |
| `[subagents.<name>]` | Agent definitions | [Subagents](#subagents--subagentsname) |
| `[frameworks.<name>]` | Bundled framework installs | [Frameworks](#frameworks--frameworksname) |
| `[models.<tool>.<route>]` | Model routes (required with the above) | [Model routes](#model-routes--modelstoolroute) |
| `[plugins.<tool>.<name>]` | Per-tool plugins | [Plugins](#plugins--pluginstoolname) |
| `[settings.<tool>]` | Per-tool settings | [Settings](#settings--settingstool) |
| `[tui.opencode]` | OpenCode TUI settings | [TUI](#tui--tuiopencode) |
| `[marketplaces.claude.<name>]` | Claude plugin marketplaces | [Marketplaces](#marketplaces--marketplacesclaudename) |
| `[agents.<name>]` | Legacy — folds into `[subagents.<name>]` | [Legacy agents](#legacy-agents--agentsname) |

## Common concepts

**Targets.** Most resources take an optional `targets` list selecting which
tools they project into. Valid values: `"claude"`, `"opencode"`, and (for MCPs
only, as an opt-in pilot) `"codex"`. Omitted means **both** `claude` and
`opencode`. A typo like `targets = ["claud"]` is rejected at load, not silently
ignored.

**Sources.** Skills, commands, subagents, and frameworks resolve their content
through a `source` string:

| Source | Resolves from |
|---|---|
| `builtin:<name>` | the catalog embedded in the binary (materialized under `.homonto/catalog/`) |
| `local:<name>` | your own content next to `homonto.toml` (`homonto/skills/<name>/`, `homonto/subagents/<name>.md`, …) |
| `remote:<url>` | a fetched, verified, pinned archive — **subagents only**, requires `digest` (see [remote source trust](remote-source-trust.md)) |

**Validation is fail-fast.** `homonto` rejects at load time, naming the
offender: an MCP with no command; an unknown target; a partial model-route set;
a settings key that collides with a structure homonto manages
(`settings.claude.enabledPlugins`, `settings.opencode.mcp`,
`settings.opencode.plugin`); a skill without a `scope`; a `remote:` source
without a `digest`; and names that would corrupt a JSON file (empty, or
index-like such as `"0"`/`"-1"`).

## MCP servers — `[mcps.<name>]`

```toml
[mcps.codegraph]
command = ["codegraph", "serve", "--mcp"]   # required, non-empty
env     = { API_KEY = "${pass:ai/key}" }    # optional; values may be secret references
targets = ["claude", "opencode", "codex"]   # optional; default: claude + opencode
```

| Field | Type | Required | Notes |
|---|---|---|---|
| `command` | array of strings | **yes** | first element is the executable, the rest are args |
| `env` | table of strings | no | values may hold `${pass:…}` / `${ENV_VAR}` references — never plaintext secrets |
| `targets` | array | no | default `["claude", "opencode"]`; add `"codex"` to opt into the Codex pilot |

Projection: Claude Code `mcpServers` (`type: stdio`), OpenCode `mcp`
(`type: local`), Codex `~/.codex/config.toml` `[mcp_servers.<name>]` (opt-in
only).

## Skills — `[skills.<name>]`

```toml
[skills.graphify]
source = "local:graphify"    # local:<name> → homonto/skills/<name>/, or builtin:<name>
scope  = "project"           # REQUIRED: user | project (no default)
targets = ["claude"]         # optional; default both
```

| Field | Type | Required | Notes |
|---|---|---|---|
| `source` | string | **yes** | `local:<name>` or `builtin:<name>` |
| `scope` | string | **yes** | `user` → `~/.claude/skills/`, `~/.config/opencode/skills/`; `project` → `<repo>/.claude/skills/`, `<repo>/.opencode/skills/` |
| `targets` | array | no | default both tools |

Skills are **symlinked**, not copied — editing `homonto/skills/<name>/` is
instantly live in every tool. Switching a skill's `scope` relocates the link
cleanly: `plan` shows the move, `apply` removes the old link as it creates the
new one. `scope` affects skills (and commands/subagents) only — MCP servers and
settings always project into the global tool files.

## Commands — `[commands.<name>]`

Slash commands, materialized as single files under
`.homonto/catalog/commands/` and linked into each tool's command directory —
Claude Code `.claude/commands/<name>.md`, OpenCode `.opencode/command/<name>.md`
(or the user-scope equivalents).

```toml
[commands.grill]
source = "builtin:grill"     # builtin:<name> | local:<name>
scope  = "project"           # user | project
```

Frameworks can also declare their own commands (e.g. onto ships `/onto`,
`/onto-open`, …); those project the same way without being declared here.

## Subagents — `[subagents.<name>]`

Agent definitions (markdown with frontmatter), projected into each tool's agent
directory. Fully declarative — reconciled by `plan`/`apply`/`status`/`doctor`
like every other resource; there is no imperative "agents" command group.

```toml
[subagents.review]
source = "builtin:onto-reviewer"   # builtin:<name> | local:<name> | remote:<url>
scope  = "project"                 # user | project (default: project)
mode   = "copy"                    # link (symlink, default) | copy (managed file)
targets = ["claude", "opencode"]   # optional; default both

[subagents.reviewer]               # a remote, pinned agent
source = "remote:https://example.com/reviewer.tar.gz"
digest = "sha256:<64 hex>"         # REQUIRED for remote:; verified before any write
```

| Field | Type | Required | Notes |
|---|---|---|---|
| `source` | string | **yes** | `builtin:` ships `onto-reviewer`, `onto-explorer`, `comet-navigator`; `local:` → `homonto/subagents/<name>.md`; `remote:` → pinned archive |
| `scope` | string | no | `user` \| `project` (default `project`) |
| `mode` | string | no | `link` (default) or `copy` — see [subagents](subagents.md) |
| `targets` | array | no | default both; `codex` has no effect (the pilot is MCP-only) |
| `digest` | string | remote only | `sha256:<64 hex>` content pin, required for `remote:` |
| `version` | string | no | informational until pinning is wired |

Where they land, link vs. copy semantics, and the tool-neutral `homonto:`
frontmatter block are covered in [subagents](subagents.md); the remote pipeline
in [remote source trust](remote-source-trust.md).

## Frameworks — `[frameworks.<name>]`

A framework is a bundled set of skills, commands, and subagents that install
together, with dependency expansion. Frameworks resolve through the **builtin
catalog only**: `onto`, `comet`, `superpowers`, `openspec`.

```toml
[frameworks.onto]
source = "builtin:onto"
scope  = "project"
```

Framework-declared commands and subagents project exactly like top-level ones
— do **not** also declare a framework's subagent in a `[subagents.*]` table
(the names collide). `homonto update` re-materializes installed frameworks at
the running binary's version.

## Model routes — `[models.<tool>.<route>]`

Any config that enables a model-backed resource (a framework, command, or
subagent) for a tool must declare **all three** routes for that tool —
`architectural`, `coding`, and `trivial`. A partial set is rejected at load.

```toml
[models.claude.architectural]
model = "opus"
variant = "max"

[models.claude.coding]
model = "sonnet"
effort = "normal"

[models.claude.trivial]
model = "haiku"
effort = "fast"

[models.opencode.architectural]
model = "anthropic/claude-opus-4-8"
variant = "high"
# … coding and trivial likewise
```

| Field | Required | Notes |
|---|---|---|
| `model` | **yes** | the tool's model identifier (Claude alias or OpenCode `provider/model` id) |
| `effort` | **one of the two** | effort hint |
| `variant` | **one of the two** | variant hint |

Every route needs a `model` **and** at least one of `effort` / `variant` — a
route carrying only a `model` is rejected at load
(`models.claude.architectural requires effort or variant`).

The routes are also **projected into each tool's default model**:
`architectural` → the tool's main model (Claude `settings.model`, OpenCode
`model`) and `trivial` → OpenCode's `small_model`. An explicit
`[settings.<tool>].model` always wins over the route-derived value. Subagents
declare a `role:` that maps to one of these routes (see
[subagents](subagents.md)).

## Plugins — `[plugins.<tool>.<name>]`

```toml
[plugins.claude.claude-hud]
source = "claude-hud@official"     # name@marketplace → enabledPlugins key
# enabled = false                  # optional; omit → enabled
# config = { compact = true }      # optional; claude only → pluginConfigs.<source>.options

[plugins.opencode.opencode-quota]
source = "@slkiser/opencode-quota" # npm package name → the `plugin` array entry
```

| Field | Type | Required | Notes |
|---|---|---|---|
| `source` | string | **yes** | Claude: `name@marketplace`; OpenCode: npm package name |
| `enabled` | bool | no | omit → enabled |
| `config` | table | no | Claude only; non-sensitive options → `pluginConfigs.<source>.options` |

## Settings — `[settings.<tool>]`

Arbitrary keys merged surgically into each tool's settings file
(`~/.claude/settings.json`; `opencode.jsonc`):

```toml
[settings.claude]
model = "opus"
theme = "dark"                            # TUI settings are top-level settings.json keys

[settings.opencode]
model = "anthropic/claude-opus-4-8"
```

Keys that collide with structures homonto manages are rejected at load:
`settings.claude.enabledPlugins`, `settings.opencode.mcp`,
`settings.opencode.plugin`. Because `hooks` is an ordinary Claude settings key,
you can declare tool hooks here too — see [enforcement](enforcement.md) for the
onto guard hook.

## TUI — `[tui.opencode]`

OpenCode keeps TUI settings in a separate file
(`~/.config/opencode/tui.json`); Claude's TUI settings are plain
`[settings.claude]` keys.

```toml
[tui.opencode]
theme = "gruvbox"
scroll_speed = 3
```

## Marketplaces — `[marketplaces.claude.<name>]`

Claude-only: entries for `extraKnownMarketplaces`.

```toml
[marketplaces.claude.official]
source = "github"                  # github | url | git-subdir | directory
repo = "anthropics/claude-plugins" # for github
# url = "…"                        # for url, git-subdir
# path = "…"                       # for git-subdir, directory
# auto_update = true               # optional
```

## Legacy agents — `[agents.<name>]`

The legacy `[agents.<name>]` table still parses but is folded into a copy-mode
`[subagents.<name>]` at load. Use `[subagents.<name>]` in new configs.

## A complete example

```toml
[mcps.codegraph]
command = ["codegraph", "serve", "--mcp"]

[mcps.brave]
command = ["npx", "-y", "@modelcontextprotocol/server-brave-search"]
env = { BRAVE_API_KEY = "${pass:ai/brave}" }

[skills.graphify]
source = "local:graphify"
scope = "project"

[frameworks.onto]
source = "builtin:onto"
scope = "project"

[subagents.review]
source = "builtin:onto-reviewer"
scope = "project"
mode = "copy"

[marketplaces.claude.official]
source = "github"
repo = "anthropics/claude-plugins"

[plugins.claude.claude-hud]
source = "claude-hud@official"

[plugins.opencode.opencode-quota]
source = "@slkiser/opencode-quota"

[settings.claude]
theme = "dark"

[settings.opencode]
model = "anthropic/claude-opus-4-8"

[tui.opencode]
theme = "gruvbox"

# Required because a framework and a subagent are enabled for both tools:
[models.claude.architectural]
model = "opus"
variant = "max"
[models.claude.coding]
model = "sonnet"
variant = "max"
[models.claude.trivial]
model = "haiku"
variant = "max"

[models.opencode.architectural]
model = "anthropic/claude-opus-4-8"
variant = "high"
[models.opencode.coding]
model = "anthropic/claude-sonnet-5"
effort = "medium"
[models.opencode.trivial]
model = "anthropic/claude-haiku-4-5"
effort = "medium"
```
