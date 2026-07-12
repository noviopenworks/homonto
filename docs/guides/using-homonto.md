# Using homonto

homonto declaratively manages the configuration of your AI CLI tools — today
**Claude Code** and **OpenCode**. You describe the MCP servers, skills, plugins,
and settings you want in one `homonto.toml`, and homonto projects that desired
state into each tool's own config files, surgically and reversibly.
The repository builds two binaries: `homonto` (this declarative
installer/projector) and `onto`, the spec-driven workflow operator — both from
source (see [the onto workflow guide](onto-workflow.md)).

It is Terraform-shaped: **edit config → `plan` → `apply`** — `homonto.toml` is
the source of truth. Every resource, agents included, is declarative: agents are
`[subagents.<name>]` tables reconciled by `plan`/`apply`/`status`/`doctor`, with
no separate imperative command group. The legacy `[agents.<name>]` table still
parses but folds into a copy-mode `[subagents.<name>]` at load.

## Install

Build from source (Go 1.23+):

```
go install .
```

> Note: avoid a bare `go build .` or `go build -o homonto .` at the repo root —
> the output name `homonto` collides with the `homonto/` content directory next
> to `main.go` (a bare build fails with `go: build output "homonto" already
> exists and is a directory`, and `-o homonto` silently deposits the binary
> inside the content dir). Use `go install .`, `go run .`, or `go build ./...`.
> If you need a local binary in the repo, write it outside the content dir,
> e.g. `go build -o ./bin/homonto .`.

Release builds stamp the version at link time:

```
go install -ldflags "-X github.com/noviopenworks/homonto/internal/cli.Version=1.2.3" .
```

> Note: homonto prints its output through cobra, which writes to **stderr**.
> When capturing output in scripts, redirect with `2>&1`.

## Quickstart

```
homonto init            # scaffold homonto.toml, .gitignore, .env.example, homonto/skills/
$EDITOR homonto.toml    # declare what you want
homonto plan            # preview the diff — writes nothing
homonto apply           # review the plan, confirm, and project it
```

## The `homonto.toml` model

One file, parsed into a tool-agnostic desired state. All sections are optional.

```toml
# MCP servers. Key = server name.
[mcps.codegraph]
command = ["codegraph", "serve", "--mcp"]   # required, non-empty

[mcps.brave]
command = ["npx", "-y", "server-brave"]
env     = { BRAVE_API_KEY = "${pass:ai/brave}" }   # secret reference, not a literal
targets = ["claude"]                                # default: both tools

# Owned skills — explicit per-resource tables. Local source resolves from
# homonto/skills/<name> (next to homonto.toml). scope is required.
[skills.graphify]
source = "local:graphify"          # local:<name> → homonto/skills/<name>
scope = "user"                     # user: ~/.claude, ~/.config/opencode
                                   # project: <repo>/.claude, <repo>/.opencode

# Per-tool plugins — one table per plugin.
[plugins.claude.claude-hud]
source = "claude-hud@official"     # name@marketplace
# enabled = false                  # optional; omit → enabled
# config = { compact = true }      # optional; claude only → pluginConfigs.<source>.options

[plugins.opencode.opencode-quota]
source = "@slkiser/opencode-quota" # npm package name

# Per-tool settings, merged into each tool's settings file.
[settings.claude]
model = "opus"

[settings.opencode]
model = "anthropic/claude-opus-4-8"
```

**Model routes.** Any config that enables a model-backed tool must declare
**all three** routes for it — `[models.<tool>.architectural]`,
`[models.<tool>.coding]`, and `[models.<tool>.trivial]` — a partial set is
rejected at load. See the roadmap's [capability
matrix](../roadmap.md#implemented-capability-matrix).

**Targets.** An MCP with no `targets` applies to both `claude` and `opencode`;
an explicit list restricts it. Only `claude` and `opencode` are valid — a typo
like `targets = ["claud"]` is rejected at load, not silently ignored.

**Skill scope.** Each `[skills.<name>]` resource declares its own required
`scope` (skills only — MCPs and settings always stay global). `user` links into
`~/.claude/skills/` and `~/.config/opencode/skills/`; `project` links into the
repo itself, `<repo>/.claude/skills/` and `<repo>/.opencode/skills/`, to keep a
project's skills in-repo. Switching a skill's scope relocates its link cleanly —
`plan` shows the move, `apply` removes the old link as it makes the new one, and
`status`/`doctor` follow each skill's declared scope. Any value other than
`user`/`project` is rejected at load.

**Validation is fail-fast.** `homonto` rejects, at load time and naming the
offender: an MCP with no command; an unknown target; a settings key that
collides with a structure homonto manages in that tool's file
(`settings.claude.enabledPlugins`, `settings.opencode.mcp`,
`settings.opencode.plugin`); and names that would corrupt a JSON file (empty,
or index-like such as `"0"`/`"-1"`).

## Secret references

Never put a plaintext secret in `homonto.toml`. Use a reference; homonto keeps
it unresolved everywhere except the moment of writing, and never stores the
resolved value:

- `${pass:path/to/secret}` — resolved via the [`pass`](https://passwordstore.org)
  password store (run `homonto doctor` to check `pass` is on your `PATH`).
- `${VAR}` — resolved from the environment (errors if the variable is unset).

`plan` never resolves secrets or contacts the backend; `apply` resolves **all**
of them up front and aborts before writing anything if any resolution fails.
The recorded state stores only a hash of the resolved value, so
`.homonto/state.json` is safe to share; it is generated state and the scaffolded
`.gitignore` excludes `.homonto/` by default.

## Commands

| Command | What it does |
|---|---|
| `homonto init [dir]` | Scaffold a starter repo; never overwrites existing files. |
| `homonto plan` | Print the diff between desired and on-disk state. Writes nothing, resolves no secrets. |
| `homonto apply` | Print the plan, confirm (`[y/N]`, or `--yes`), then write atomically. |
| `homonto status` | Report drift and pending config changes (see below). |
| `homonto doctor` | Health checks: `pass` on `PATH`, tool config locations, and each owned skill's content + both tool symlinks. |
| `homonto version` | Print the build version. |

A persistent `--config` flag (default `homonto.toml`) points at your file.

### Experimental import

Already have MCP servers configured in Claude Code? `homonto import` can
bootstrap a starter `homonto.toml` from `~/.claude.json` `mcpServers`.
It refuses to overwrite an existing config unless you pass `--force`, and it
redacts env values that look like secrets into `${pass:…}` references. It is
deliberately partial: Claude global MCP servers only, no OpenCode import, no
skills/plugins/settings import, and command/args are copied verbatim for review.

### `plan` and `apply`

The plan is a Terraform-style diff: `+` create, `~` update, `-` delete;
unchanged keys are silent. `apply` is two-phase — resolve every secret first,
then write — and each file is written atomically (temp + rename), so an
interrupted run never leaves a half-written file. State is saved per adapter, so
a failure in the second tool never loses the first tool's applied records.

### `status`: drift vs. pending

`status` distinguishes two different things:

- **Drift** — a value changed *on disk, outside homonto* since the last apply
  (or a managed value was deleted). Reported per key:
  `claude setting.model drifted (will reset on apply)`.
- **Pending** — edits you made in `homonto.toml` that haven't been applied yet.
  The disk still matches the last apply, so this is *not* drift; it is reported
  as a count: `1 config change(s) awaiting apply (run \`homonto apply\`)`.

When neither is present: `No drift.`

## How projection works

- **Surgical merge.** Each adapter writes only the keys homonto manages and
  preserves every unmanaged key already in the tool's file. An unparseable tool
  file makes that adapter abort and report — never overwrite. (Writing
  `opencode.jsonc` normalizes it and drops comments.)
- **Skills are symlinked**, not copied, from `homonto/skills/<name>` into each
  tool's skills directory, with conflict detection (a real file or a link
  pointing elsewhere is reported, never clobbered).
- **Pruning.** Remove a resource from `homonto.toml` and the next apply deletes
  it from the tool files. Only resources homonto recorded in state are ever
  pruned — your own hand-added keys are never touched.
- **Adoption.** If a resource you declare already exists on disk exactly as
  homonto would write it, `apply` records it into state quietly (no file write,
  no diff line, no prompt — "Reconciled N pre-existing resource(s) into state.")
  so it becomes visible to drift and pruning. See
  [`status-and-adoption`](status-and-adoption.md).
- **State** lives in `.homonto/state.json` next to your config: the last-applied
  snapshot (unresolved desired value + a hash of the applied value, never a
  plaintext secret).

## Known limitations

- `import` covers Claude global MCP servers only; OpenCode, and Claude
  settings/plugins/skills, are not imported. Non-stdio (url/http) servers are
  skipped with a warning. Command/args are copied verbatim — review before
  sharing.
- `[frameworks.X]` resolves through the bundled builtin catalog and projects
  skills, commands, and subagents (with dependency expansion, including
  framework-declared `[commands]` and `[subagents]` tables) into Claude Code
  and OpenCode. `[commands.X]` (builtin or local) materialize as single files
  under `.homonto/catalog/commands/` and link into each tool's command
  directory. `[subagents.X]` (builtin or local, including required
  model-route validation) materializes verbatim as single files under
  `.homonto/catalog/subagents/` and links into each tool's agent directory —
  Claude Code's `agents/` (user: `~/.claude/agents/`, project:
  `.claude/agents/`) and OpenCode's `agent/` (user:
  `~/.config/opencode/agent/`, project: `.opencode/agent/`) — with `doctor`
  verifying both tools' links.
- The standalone `onto` binary ships alongside `homonto` (built from
  `cmd/onto/`); it operates the spec-driven workflow (`init`, `new`, `status`,
  `advance`, `close`, `doctor`). See [the onto workflow guide](onto-workflow.md).
- Writing `opencode.jsonc` removes comments (whole-document JSONC
  normalization).
- Skill symlinks store an absolute target, so if you **move or rename your
  homonto repo**, existing links point at the old path (now outside the content
  root) and `apply`/`status` report a conflict instead of silently repointing —
  homonto never changes a symlink it can't prove it owns. Delete the stale links
  and re-run `apply` to relink at the new location.
- CLI output is written to stderr; redirect with `2>&1` when scripting.
