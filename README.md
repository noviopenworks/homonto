# homonto

Declarative config for your AI coding tools. Describe your MCP servers, skills,
plugins, and settings once in `homonto.toml`; `homonto apply` projects them into
**Claude Code** and **OpenCode** through a terraform-style plan/confirm/apply
pipeline. Secrets are **referenced, never stored** — resolved only at apply time.
The public v0.1 release gate is dual-binary: `homonto` remains the deterministic
installer/projector, and `onto` will ship beside it as the spec-driven workflow
operator.

## Install

The first public tag is intentionally pending until both binaries are ready.
Once a release is tagged, install a specific version directly:

```bash
go install github.com/noviopenworks/homonto@v0.1.0-rc.1   # a tagged release
```

Tagged releases also ship prebuilt `homonto` and `onto` binaries for
Linux/macOS/Windows (amd64 and arm64) with a `SHA256SUMS` file, attached to the
GitHub release.

Until the first tag lands you can install/run the current `homonto` source
instead:

```bash
go install github.com/noviopenworks/homonto@main   # current main branch
go install .                                        # from a checked-out repo
```

## Quickstart

```bash
homonto init            # scaffold homonto.toml, .gitignore, .env.example, homonto/skills/
$EDITOR homonto.toml    # declare your MCPs / skills / plugins / settings
homonto plan            # dry run: show the diff, write nothing
homonto apply           # plan → confirm [y/N] → write (use --yes to skip prompt)
```

Other commands:

| Command | What it does |
|---|---|
| `homonto status` | Show managed values that would be reset or recreated on apply |
| `homonto doctor` | Health check: `pass` present? tool dirs present? owned skill content and both tool links present? |
| `homonto --version` | Print the build version |

`--config <path>` selects a different config file for plan/apply/status/doctor/import.
`init` instead takes an optional target directory and always writes
`homonto.toml` inside that directory.

Experimental adoption helper: `homonto import` bootstraps Claude global MCP
servers into `homonto.toml` with best-effort env redaction. It is deliberately
narrow and is not part of the main quickstart path.

## `homonto.toml`

```toml
[mcps.codegraph]
command = ["codegraph", "serve", "--mcp"]
targets = ["claude", "opencode"]          # default: all tools

[mcps.brave]
command = ["npx", "-y", "@modelcontextprotocol/server-brave-search"]
env = { BRAVE_API_KEY = "${pass:ai/brave}" }

[skills.graphify]
source = "local:graphify"                 # local:<name> → homonto/skills/<name>
scope = "project"                         # required: user | project (no default)

[plugins]
claude = ["claude-hud@official"]          # marketplace entries
opencode = ["@slkiser/opencode-quota"]    # npm packages

[settings.claude]
model = "opus"

[settings.opencode]
model = "anthropic/claude-opus-4-8"

[models.claude.architectural]             # required for every model-enabled tool
model = "opus"
variant = "max"
```

The example is abbreviated — a complete config must also define
`models.claude.coding` and `models.claude.trivial`, and the same three levels
apply to every model-enabled target tool.

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

Skills you author live under `homonto/skills/` (the local provider root, next
to `homonto.toml`) and are **symlinked** into each tool, so editing
`homonto/skills/...` is instantly live everywhere. `apply` ensures the links
exist and point correctly; it never clobbers a file that isn't its own symlink
(reported as a conflict instead). A skills-only apply leaves tool JSON files
byte-for-byte untouched — adapters write a file only when a managed key inside
it actually changes — so OpenCode JSONC comments survive link-only applies.

### Skill scope — user or project

Each skill resource declares its own `scope` (`scope` is required; there is no
default). `scope` chooses where that skill is linked (it affects skills only;
MCP servers and settings always project into your global tool files):

- `scope = "user"` — links into `~/.claude/skills/` and
  `~/.config/opencode/skills/`.
- `scope = "project"` — links into the project itself, next to `homonto.toml`:
  `<repo>/.claude/skills/` and `<repo>/.opencode/skills/`. Use this to keep a
  project's skills in-repo instead of your global tool config.

Switching a skill's scope is clean: `plan` shows the link relocating from its
old location to the new one, and `apply` removes the old link as it creates the
new one, so no orphaned symlink is left behind. `status` and `doctor` report
against whichever location each skill's scope selects.

## Surgical merge

homonto writes **only the keys it manages** and preserves every unmanaged key in
each tool's file. Removal is declarative too: keys you remove from
`homonto.toml` are deleted from the tool files on the next apply (and
owned-skill links removed) — state tracks what homonto manages. A skills-only
apply leaves tool JSON files byte-for-byte untouched, since adapters write a file
only when a managed key inside it actually changes.

## Known limitations

homonto is a young, deliberately narrow tool. For the v0.1.0 beta line:

- **`onto` is release-blocking; its foundation and `onto init` have landed,
  not a release-complete binary.** A second `package main` at `cmd/onto`
  builds an `onto` binary alongside `homonto`, with an `onto version` command,
  a read-only `onto status` that reports each active change's phase from
  `docs/changes/*/onto-state.yaml` without touching `homonto.toml`, and an
  `onto init` that idempotently scaffolds the `docs/{changes,specs,adr,
  guides}` layout, gated behind the Homonto framework install. Phase-gate
  enforcement, `onto doctor`, and dual-binary release packaging are not
  implemented yet; the repo still dogfoods the markdown skills workflow
  today.
- **Framework skill, command, and subagent projection are all implemented.**
  `[frameworks.X]` resolves through the bundled builtin catalog (`onto`,
  `comet`, `superpowers`, `openspec`), expands dependencies, and
  materializes/links skills into Claude Code and OpenCode. `[commands.X]`
  (builtin or local, single-file materialization to
  `.homonto/catalog/commands/`) and framework-declared `[commands]` tables
  project the same way, into Claude Code (`.claude/commands/<name>.md`) and
  OpenCode (`.opencode/command/<name>.md` project-scoped, or the equivalent
  user-scope directories). `[subagents.X]` (builtin or local, single-file
  materialization to `.homonto/catalog/subagents/`) and framework-declared
  `[subagents]` tables project verbatim the same way, into Claude Code
  (`.claude/agents/<name>.md`) and OpenCode (`.opencode/agent/<name>.md`
  project-scoped, or the equivalent user-scope directories), with `doctor`
  verifying the links. Three real subagents ship in the catalog:
  `code-reviewer`, `codebase-explorer`, and `comet-navigator`.
- **OpenCode JSONC comments are not preserved.** Claude's files are plain JSON,
  but OpenCode's `opencode.jsonc` supports comments. Any apply that *writes*
  `opencode.jsonc` rewrites it as normalized JSON, so **all comments in that file
  are removed**. (A skills-only or otherwise no-op apply does not write the file,
  so comments survive those.) Accepted for beta.
- **`import` is a narrow Claude MCP bootstrap.** It reads Claude's global MCP
  servers only, redacts values that *look* like secrets into `${pass:...}`
  references (best-effort, not exhaustive), and preserves `command`/`args`
  verbatim. It does not import skills, plugins, settings, or OpenCode config.
  Treat its output as a starting point to review, not a complete migration.
- **Two tools only.** Claude Code and OpenCode are the only adapters today.
- **Secrets need `pass` or an env var.** `${pass:...}` references require
  [`pass`](https://www.passwordstore.org/) on `PATH`; `${ENV_VAR}` references
  require the variable to be set at apply time. `homonto doctor` flags a missing
  `pass`.

---

## For contributors

Everything below is about developing homonto itself — users don't need it.

### How it works

`homonto.toml` is parsed into one tool-agnostic desired-state model; each tool is
an adapter (`Read` → `Plan` → `Apply`) wired by the engine. Adding a new tool
requires a new adapter plus engine/config wiring. Writes are atomic (temp +
rename); state is persisted after each successful adapter so a later adapter
failure does not lose earlier records.

### Development workflow

This repo is developed with **Comet**: OpenSpec owns WHAT, Superpowers owns HOW,
and Comet state/scripts bind the phases together. New development starts with
`/comet`; active changes live under `openspec/changes/`, and deep technical
designs and implementation plans live under `docs/superpowers/`.

- `docs/adr/` — accepted architecture decisions
- `docs/specs/` — living capability specs (SHALL + scenarios)
- `openspec/changes/` — active Comet/OpenSpec change workspaces (+ archive)
- `docs/changes/` — legacy Onto workspaces (historical, do not open new work)
- `docs/guides/` — user-facing guides
- `docs/road-to-release.md` — the release gate; `docs/release-checklist.md` — how to cut a release

Start with `/comet`. Full guide: [docs/guides/comet-workflow.md](docs/guides/comet-workflow.md).

The older `docs/changes/` Onto workspaces are historical. Do not open new work
there.

Future agents should start with `/comet` — it inspects `openspec/changes/` and
each active change's `.comet.yaml` for current state — before trusting older
reviews or archived change artifacts.
