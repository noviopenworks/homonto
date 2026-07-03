# homonto

Declarative config for your AI coding tools. Describe your MCP servers, skills,
plugins, and settings once in `homonto.toml`; `homonto apply` projects them into
**Claude Code** and **OpenCode** through a terraform-style plan/confirm/apply
pipeline. Secrets are **referenced, never stored** — resolved only at apply time.

## Install

```bash
go install github.com/noviopenworks/homonto@latest
```

## Quickstart

```bash
homonto init            # scaffold homonto.toml, .gitignore, .env.example, content/
$EDITOR homonto.toml    # declare your MCPs / skills / plugins / settings
homonto plan            # dry run: show the diff, write nothing
homonto apply           # plan → confirm [y/N] → write (use --yes to skip prompt)
```

Other commands:

| Command | What it does |
|---|---|
| `homonto status` | Show drift: tool files vs. last-applied state |
| `homonto doctor` | Health check: `pass` present? owned skills present? |
| `homonto import` | Bootstrap `homonto.toml` from your current setup (redacts secrets) |

`--config <path>` selects a different config file for any command.

## `homonto.toml`

```toml
[mcps.codegraph]
command = ["codegraph", "serve", "--mcp"]
targets = ["claude", "opencode"]          # default: all tools

[mcps.brave]
command = ["npx", "-y", "@modelcontextprotocol/server-brave-search"]
env = { BRAVE_API_KEY = "${pass:ai/brave}" }

[skills]
own = ["graphify"]                        # from content/skills/

[plugins]
claude = ["claude-hud@official"]          # marketplace entries
opencode = ["@slkiser/opencode-quota"]    # npm packages

[settings.claude]
model = "opus"

[settings.opencode]
model = "anthropic/claude-opus-4-8"
```

## Secrets — referenced, never stored

Secret values live outside the repo and are referenced by token:

- `${pass:PATH}` — resolved via [`pass`](https://www.passwordstore.org/).
- `${ENV_VAR}` — resolved from the environment (zero-dependency fallback).

Guarantees:

- `plan` **never** resolves or prints a secret — it shows the `${...}` token.
- `apply` resolves secrets only **after** you confirm, **all at once before any
  file is written**; a missing reference aborts before touching anything.
- `.homonto/state.json` stores only the unresolved token plus a **sha256 hash**
  of the applied value — never plaintext — so it is safe to share and a repeat
  `apply` on a secret-backed value is a no-op (idempotent), while an out-of-band
  change to that value is still detected as drift.

## Owned content is symlinked

Skills/commands/rules/agents you author live in `content/` and are **symlinked**
into each tool, so editing `content/...` is instantly live everywhere. `apply`
just ensures the links exist and point correctly; it never clobbers a file that
isn't its own symlink (reported as a conflict instead).

## Surgical merge & the JSONC caveat

homonto writes **only the keys it manages** and preserves every unmanaged key in
each tool's file. Claude's files are plain JSON. OpenCode's `opencode.jsonc`
supports comments: all keys (managed and unmanaged) are preserved on merge, but
**inline comments inside rewritten regions may not survive** — this is a known,
documented limitation.

## How it works

`homonto.toml` is parsed into one tool-agnostic desired-state model; each tool is
an adapter (`Read` → `Plan` → `Apply`). Adding a new tool later is one adapter,
no engine changes. Writes are atomic (temp + rename); `state.json` is written
last so an interrupted apply always leaves each file valid.
