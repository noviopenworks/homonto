# homonto

Declarative config for your AI coding tools. Describe your MCP servers, skills,
plugins, and settings once in `homonto.toml`; `homonto apply` projects them into
**Claude Code** and **OpenCode** through a terraform-style plan/confirm/apply
pipeline. Secrets are **referenced, never stored** — resolved only at apply time.

## Install

```bash
go install github.com/noviopenworks/homonto@latest
```

homonto is pre-release: there are no version tags yet, so `@latest` installs
from the tip of `main`.

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
| `homonto doctor` | Health check: `pass` present? owned skills present and linked? |
| `homonto import` | Bootstrap `homonto.toml` from your current setup (redacts secrets) |
| `homonto --version` | Print the build version |

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

Skills you author live in `content/` and are **symlinked**
into each tool, so editing `content/...` is instantly live everywhere. `apply`
just ensures the links exist and point correctly; it never clobbers a file that
isn't its own symlink (reported as a conflict instead).

## Surgical merge & the JSONC caveat

homonto writes **only the keys it manages** and preserves every unmanaged key in
each tool's file. Removal is declarative too: keys you remove from
`homonto.toml` are deleted from the tool files on the next apply (and
owned-skill links removed) — state tracks what homonto manages. Claude's files
are plain JSON. OpenCode's `opencode.jsonc` supports comments, but homonto does
not preserve them: any apply that touches the file rewrites it as normalized
JSON, so **all comments in `opencode.jsonc` are removed** — a known, documented
limitation.

## How it works

`homonto.toml` is parsed into one tool-agnostic desired-state model; each tool is
an adapter (`Read` → `Plan` → `Apply`). Adding a new tool later is one adapter,
no engine changes. Writes are atomic (temp + rename); `state.json` is written
last so an interrupted apply always leaves each file valid.

## Development workflow

This repo is developed with **onto**, a self-contained markdown workflow
shipped from this very repo (`content/skills/onto*` — dogfooded via
`homonto apply`). Five phases (open → design → build → verify → close) plus
`/onto-fix` and `/onto-tweak` presets; artifacts live under `docs/`:

- `docs/adr/` — accepted architecture decisions
- `docs/specs/` — living capability specs (SHALL + scenarios)
- `docs/changes/` — active change workspaces (+ `archive/`)
- `docs/guides/` — user-facing guides

Start with `/onto`. Full guide: [docs/guides/onto-workflow.md](docs/guides/onto-workflow.md).
