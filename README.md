# homonto

Declarative config for your AI coding tools. Describe your MCP servers, skills,
plugins, and settings once in `homonto.toml`; `homonto apply` projects them into
**Claude Code** and **OpenCode** through a terraform-style plan/confirm/apply
pipeline. Secrets are **referenced, never stored** — resolved only at apply time.

## Install

homonto is pre-release: there are no version tags or release artifacts yet. From
a checked-out repo, install the current source with:

```bash
go install .
```

If the repository is accessible remotely, `go install
github.com/noviopenworks/homonto@main` installs the current main branch.

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
| `homonto status` | Show managed values that would be reset or recreated on apply |
| `homonto doctor` | Health check: `pass` present? tool dirs present? owned skill content and Claude links present? |
| `homonto import` | Bootstrap Claude global MCP servers into `homonto.toml` (best-effort env redaction) |
| `homonto --version` | Print the build version |

`--config <path>` selects a different config file for plan/apply/status/doctor/import.
`init` instead takes an optional target directory and always writes
`homonto.toml` inside that directory.

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
ensures the links exist and point correctly; it never clobbers a file that isn't
its own symlink (reported as a conflict instead). A skills-only apply leaves
tool JSON files byte-for-byte untouched — adapters write a file only when a
managed key inside it actually changes — so OpenCode JSONC comments survive
link-only applies.

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
an adapter (`Read` → `Plan` → `Apply`) wired by the engine. Adding a new tool
requires a new adapter plus engine/config wiring. Writes are atomic (temp +
rename); state is persisted after each successful adapter so a later adapter
failure does not lose earlier records.

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

Future agents should start with [docs/NEXT_AGENT.md](docs/NEXT_AGENT.md) before
trusting older reviews or archived change artifacts.
