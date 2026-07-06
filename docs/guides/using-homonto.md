# Using homonto

homonto declaratively manages the configuration of your AI CLI tools — today
**Claude Code** and **OpenCode**. You describe the MCP servers, skills, plugins,
and settings you want in one `homonto.toml`, and homonto projects that desired
state into each tool's own config files, surgically and reversibly.

It is Terraform-shaped: **edit config → `plan` → `apply`**. There are no
imperative `add`/`remove` commands — the file is the source of truth.

## Install

Build from source (Go 1.23+):

```
go build -o homonto .
```

Release builds stamp the version at link time:

```
go build -ldflags "-X github.com/noviopenworks/homonto/internal/cli.Version=1.2.3" -o homonto .
```

> Note: homonto prints its output through cobra, which writes to **stderr**.
> When capturing output in scripts, redirect with `2>&1`.

## Quickstart

```
homonto init            # scaffold homonto.toml, .gitignore, .env.example, content/skills/
$EDITOR homonto.toml    # declare what you want
homonto plan            # preview the diff — writes nothing
homonto apply           # review the plan, confirm, and project it
```

Already have MCP servers configured in Claude Code? Bootstrap from them:

```
homonto import          # reads ~/.claude.json mcpServers into a starter homonto.toml
```

`import` refuses to overwrite an existing config unless you pass `--force`, and
it redacts env values that look like secrets into `${pass:…}` references. It is
deliberately partial — it imports Claude global MCP servers only; review the
generated file before sharing (command/args are copied verbatim).

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

# Owned skills — directories under content/skills/, symlinked into each tool.
[skills]
scope = "user"                    # user (default): ~/.claude, ~/.config/opencode
                                  # project: <repo>/.claude, <repo>/.opencode
own = ["graphify", "onto"]

# Per-tool plugins.
[plugins]
claude   = ["claude-hud@official"]
opencode = ["@slkiser/opencode-quota"]

# Per-tool settings, merged into each tool's settings file.
[settings.claude]
model = "opus"

[settings.opencode]
model = "anthropic/claude-opus-4-8"
```

**Targets.** An MCP with no `targets` applies to both `claude` and `opencode`;
an explicit list restricts it. Only `claude` and `opencode` are valid — a typo
like `targets = ["claud"]` is rejected at load, not silently ignored.

**Skill scope.** `[skills] scope` selects where owned skills are linked (skills
only — MCPs and settings always stay global). `user` (the default when omitted)
links into `~/.claude/skills/` and `~/.config/opencode/skills/`; `project` links
into the repo itself, `<repo>/.claude/skills/` and `<repo>/.opencode/skills/`,
to keep a project's skills in-repo. Switching scope relocates the links cleanly —
`plan` shows the move, `apply` removes the old link as it makes the new one, and
`status`/`doctor` follow the active scope. Any value other than `user`/`project`
is rejected at load.

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
The recorded state stores only a hash of the resolved value, so `.homonto/state.json`
is safe to commit.

## Commands

| Command | What it does |
|---|---|
| `homonto init [dir]` | Scaffold a starter repo; never overwrites existing files. |
| `homonto import` | Bootstrap `homonto.toml` from Claude global MCP servers (secret-redacting; `--force` to overwrite). |
| `homonto plan` | Print the diff between desired and on-disk state. Writes nothing, resolves no secrets. |
| `homonto apply` | Print the plan, confirm (`[y/N]`, or `--yes`), then write atomically. |
| `homonto status` | Report drift and pending config changes (see below). |
| `homonto doctor` | Health checks: `pass` on `PATH`, tool config locations, and each owned skill's content + both tool symlinks. |
| `homonto version` | Print the build version. |

A persistent `--config` flag (default `homonto.toml`) points at your file.

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
- **Skills are symlinked**, not copied, from `content/skills/<name>` into each
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
- Writing `opencode.jsonc` removes comments (whole-document JSONC
  normalization).
- CLI output is written to stderr; redirect with `2>&1` when scripting.
