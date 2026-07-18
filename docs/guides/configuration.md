# Configuration reference — `homonto.toml`

One file, parsed into a tool-agnostic desired state. All sections are
optional; an empty config is valid and projects nothing. `homonto plan`,
`apply`, `status`, `doctor`, and `import` accept `--config <path>` (default
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
tools they project into. Valid values: `"claude"`, `"opencode"`, and (for
MCPs only, as an opt-in pilot) `"codex"`. Omitting the list means both
`claude` and `opencode`. A typo like `targets = ["claud"]` fails at load, not
silently.

**Sources.** Skills, commands, subagents, and frameworks resolve their
content through a `source` string:

| Source | Resolves from | Available for |
|---|---|---|
| `builtin:<name>` | the catalog embedded in the binary, materialized under `.homonto/catalog/` | skills, commands, subagents, frameworks |
| `local:<name>` | your own content next to `homonto.toml` (`homonto/skills/<name>/`, `homonto/subagents/<name>.md`, a framework root) | skills, commands, subagents, frameworks |
| `remote:<url>` | a fetched, verified, pinned archive; requires `digest` (see [remote source trust](remote-source-trust.md)) | subagents, frameworks |

**Validation is fail-fast.** homonto rejects at load time and names the
offender: an MCP with no command, an unknown target, a partial model-route
set, a settings key that collides with a structure homonto manages
(`settings.claude.enabledPlugins`, `settings.opencode.mcp`,
`settings.opencode.plugin`), a skill without a `scope`, a `remote:` source
without a `digest`, and names that would corrupt a JSON file (empty, or
index-like such as `"0"`/`"-1"`).

## MCP servers — `[mcps.<name>]`

```toml
[mcps.codegraph]
command = ["codegraph", "serve", "--mcp"]   # required, non-empty
env     = { API_KEY = "${pass:ai/key}" }    # optional; values may be secret references
targets = ["claude", "opencode", "codex"]   # optional; default: claude + opencode
scope   = "project"                         # optional; user (default) | project
```

| Field | Type | Required | Notes |
|---|---|---|---|
| `command` | array of strings | **yes** | first element is the executable, the rest are args |
| `env` | table of strings | no | values may hold `${pass:…}` / `${ENV_VAR}` references — never plaintext secrets |
| `targets` | array | no | default `["claude", "opencode"]`; add `"codex"` to opt into the Codex pilot |
| `scope` | string | no | `user` (default) → global tool config; `project` → the project-level config the tool merges over it |

Projection at user scope: Claude Code `~/.claude.json` `mcpServers`
(`type: stdio`), OpenCode global `opencode.jsonc` `mcp` (`type: local`),
Codex `~/.codex/config.toml` `[mcp_servers.<name>]` (opt-in only). At
project scope: Claude Code `<repo>/.mcp.json` `mcpServers`, OpenCode
`<repo>/opencode.jsonc` `mcp` — the server runs only in that repository's
sessions instead of everywhere. Codex is user-scope only; a project-scoped
server targeting codex fails at load. Switching an applied server's scope
migrates it on the next `apply` (pruned from the old file, written to the
new one).

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

Skills are **symlinked**, not copied, so editing `homonto/skills/<name>/` is
instantly live in every tool. Switching a skill's `scope` relocates the link
cleanly: `plan` shows the move, and `apply` removes the old link as it
creates the new one. `scope` affects skills, commands, subagents, and
[MCP servers](#mcp-servers--mcpsname) directly; explicit `[settings.<tool>]`
keys always project into the global tool files, while the route-derived
default-model keys follow the model-backed resources' scope — see
[model routes](#model-routes--modelstoolroute).

## Commands — `[commands.<name>]`

Slash commands, materialized as single files under
`.homonto/catalog/commands/` and linked into each tool's command directory:
Claude Code `.claude/commands/<name>.md`, OpenCode
`.opencode/command/<name>.md` (or the user-scope equivalents).

```toml
[commands.grill]
source = "builtin:grill"     # builtin:<name> | local:<name>
scope  = "project"           # user | project
```

Frameworks declare their own commands too (onto ships `/onto`, `/onto-open`,
…); those project the same way without being declared here.

## Subagents — `[subagents.<name>]`

Agent definitions (markdown with frontmatter), projected into each tool's
agent directory. Fully declarative: reconciled by
`plan`/`apply`/`status`/`doctor` like every other resource. There is no
imperative "agents" command group.

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
| `source` | string | **yes** | `builtin:` ships `onto-reviewer`, `onto-explorer`, `onto-implementer`, `onto-skeptic` (and the `to-*` twins); `local:` → `homonto/subagents/<name>.md`; `remote:` → pinned archive |
| `scope` | string | no | `user` \| `project` (default `project`) |
| `mode` | string | no | `link` (default) or `copy` — see [subagents](subagents.md) |
| `targets` | array | no | default both; `codex` has no effect (the pilot is MCP-only) |
| `digest` | string | remote only | `sha256:<64 hex>` content pin, required for `remote:` |
| `version` | string | no | informational until pinning is wired |

Where they land, link vs. copy semantics, and the tool-neutral `homonto:`
frontmatter block: [subagents](subagents.md). The remote pipeline:
[remote source trust](remote-source-trust.md).

## Frameworks — `[frameworks.<name>]`

A framework is a bundled set of skills, commands, and subagents that install
together, with dependency expansion. The builtin catalog ships exactly the
two homonto-native frameworks, `onto` and `to`, and they are **mutually
exclusive**: declaring both fails at load (one workflow per repository).
Beyond `builtin:`, a framework source may be `local:<path>` (a framework root
in your repo) or `remote:<url>` with a required `digest = "sha256:…"` pin.
Third-party workflow stacks are not bundled.

```toml
[frameworks.onto]        # or [frameworks.to] — never both
source = "builtin:onto"
scope  = "project"
```

Framework-declared commands and subagents project exactly like top-level
ones. Do **not** also declare a framework's subagent in a `[subagents.*]`
table; the names collide. `homonto update` re-materializes installed
frameworks at the running binary's version.

## Model routes — `[models.<tool>.<route>]`

Any config that enables a model-backed resource (a framework, command, or
subagent) for a tool must declare **all four** routes for that tool:
`architectural` (orchestrate/design), `coding` (implement), `review` (judge
others' work — the reviewer and the skeptic run here), and `trivial` (cheap
lookups). A partial set fails at load.

```toml
[models.claude.architectural]
model = "opus"

[models.claude.coding]
model = "sonnet"
effort = "medium"

[models.claude.review]
model = "opus"
effort = "high"

[models.claude.trivial]
model = "haiku"
effort = "low"

[models.opencode.architectural]
model = "anthropic/claude-opus-4-8"
variant = "high"
# … coding, review, and trivial likewise
```

| Field | Required | Notes |
|---|---|---|
| `model` | **yes** | the tool's model identifier |
| `effort` | no | how hard to think |
| `variant` | no | which variant of the model |

A route naming just a `model` is complete; `effort` and `variant` are
optional.

### The two tools spell these differently

The fields are declared neutrally and rendered per tool because each tool
accepts only its own half:

| | Claude Code | OpenCode |
|---|---|---|
| `model` | an alias (`opus`, `sonnet`, `haiku`, `fable`, `opusplan`) or a full id (`claude-opus-4-8`) | `provider/model` |
| `variant` | **has no field** — rendered *into* the model string as `opus[1m]`, and only an **alias** can take one; `1m` is the only documented variant | a **first-class `variant:` field** taking any variant your provider defines |
| `effort` | a real frontmatter field: `low`, `medium`, `high`, `xhigh`, `max` | **no such concept** — declaring it is a config error |

Each value is validated against the tool that will receive it, so a setting
the tool would silently ignore becomes a load error naming the offender:

```
parse config: models.claude.coding effort "normal" is not a Claude effort level (low, medium, high, xhigh, max)
parse config: models.claude.architectural variant "1m" needs a model alias (…) — Claude takes no variant on the full model id "claude-opus-4-8"
parse config: models.opencode.coding sets effort "high", but OpenCode has no effort setting — use variant, or drop it
```

The routes also project into each tool's default model: `architectural` →
the tool's main model (Claude `settings.model`, OpenCode `model`) and
`trivial` → OpenCode's `small_model`. **Where** those keys land follows the
scope of the model-backed resources that required the routes: when every
model-backed resource (framework, command, subagent) enabled for a tool is
project-scoped, the keys project into the project-level config the tool
merges over its global one (`<repo>/opencode.jsonc`;
`<repo>/.claude/settings.json`), so one repository's workflow models never
become another session's defaults. Any user-scope model-backed resource
keeps them in the global file, as before. An explicit
`[settings.<tool>].model` always wins over the route-derived value — and
suppresses the project-level twin entirely, which would otherwise override
it in the tool's merge order. Subagents declare a `role:` that maps to one
of these routes (see [subagents](subagents.md)).

The four tier names are closed: a `[models.<tool>.<level>]` block naming
any other level fails at load —

```
parse config: models.opencode.reviewing is not a model tier; valid tiers are "architectural", "coding", "review", "trivial" (agents pick one via their role)
```

— and an agent frontmatter `role:` outside the same four tiers fails at
render instead of silently emitting an agent with no model.

### Retuning one agent — `[subagents.<name>.<tool>]`

A tier is the default for every agent of that role. To retune a single
agent, declare a per-tool block under its name; each field set there wins
over the tier, **field by field**, so an effort-only override keeps the
tier's model:

```toml
[models.claude.review]
model = "opus"
variant = "1m"
effort = "high"          # the default for every review agent

[subagents.onto-skeptic.claude]
effort = "max"           # …but the skeptic thinks harder
```

`onto-skeptic` renders as `model: opus[1m]` + `effort: max`;
`onto-reviewer`, on the same tier, stays at `high`.

There is **no `[subagents.onto-skeptic]` declaration** above, and that is
the point: `onto-skeptic` is installed by `[frameworks.onto]`, and a
framework's subagent may not be re-declared explicitly (that collision is an
error). A per-tool block with **no `source`** therefore reads as *tune this
agent*, not *declare it*. It projects nothing and never collides with the
framework that owns the agent. For a subagent you declare yourself, add the
block under your own `[subagents.<name>]` entry the same way.

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

Keys that collide with structures homonto manages fail at load:
`settings.claude.enabledPlugins`, `settings.opencode.mcp`,
`settings.opencode.plugin`. Because `hooks` is an ordinary Claude settings
key, you can declare tool hooks here too — see
[enforcement](enforcement.md) for the workflow guard hook.

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

The legacy `[agents.<name>]` table still parses but folds into a copy-mode
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
[models.claude.coding]
model = "sonnet"
[models.claude.review]
model = "opus"
[models.claude.trivial]
model = "haiku"

[models.opencode.architectural]
model = "anthropic/claude-opus-4-8"
variant = "high"
[models.opencode.coding]
model = "anthropic/claude-sonnet-5"
[models.opencode.review]
model = "anthropic/claude-opus-4-8"
[models.opencode.trivial]
model = "anthropic/claude-haiku-4-5"
```
