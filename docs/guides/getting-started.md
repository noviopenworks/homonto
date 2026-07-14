# Getting started (v0.1.0)

Two binaries. `homonto` projects your `homonto.toml` into Claude Code / OpenCode
(Terraform-style: `plan` → `apply`). `onto` gates a change through
`open → design → build → verify → close`. `onto`'s mutating commands need the
`onto` framework installed *by* `homonto` first.

Full reference: [`using-homonto.md`](using-homonto.md) and
[`onto-workflow.md`](onto-workflow.md). (Output goes to **stderr** — redirect
`2>&1` when scripting.)

## Install

```bash
go install github.com/noviopenworks/homonto@v0.1.0          # homonto
go install github.com/noviopenworks/homonto/cmd/onto@v0.1.0 # onto
```

Or grab the prebuilt binaries + `SHA256SUMS` from the GitHub release
(linux/macOS/windows, amd64/arm64). From a checked-out repo use `go install .` —
**not** a bare `go build .` (the output name collides with the `homonto/` dir).

## homonto in five commands

```console
$ homonto init            # scaffold homonto.toml + .gitignore + .env.example (never overwrites)
$ $EDITOR homonto.toml    # declare MCPs / skills / plugins / settings
$ homonto plan            # show the diff — writes nothing, resolves no secrets
$ homonto apply           # confirm [y/N] (--yes to skip), then write atomically
$ homonto status          # report drift / pending / clean
```

A realistic `homonto.toml`:

```toml
[mcps.codegraph]
command = ["codegraph", "serve", "--mcp"]      # both tools by default

[mcps.brave]
command = ["npx", "-y", "@modelcontextprotocol/server-brave-search"]
env = { BRAVE_API_KEY = "${BRAVE_API_KEY}" }    # a reference, never a literal secret
targets = ["claude"]                            # restrict to Claude Code

[skills.my-notes]
source = "local:my-notes"                       # → homonto/skills/my-notes/
scope = "project"                               # required: user | project

[settings.claude]
model = "opus"
```

`plan` output (`+` create, `~` update, `-` delete; secrets stay tokens):

```console
$ homonto plan
claude:
  + mcp.brave = {"command":"npx","args":["-y","@modelcontextprotocol/server-brave-search"],"env":{"BRAVE_API_KEY":"${BRAVE_API_KEY}"},"type":"stdio"}
  + mcp.codegraph = {"command":"codegraph","args":["serve","--mcp"],"type":"stdio"}
  + setting.model = "opus"
  + skill.my-notes = …/.claude/skills/my-notes -> …/homonto/skills/my-notes
opencode:
  + mcp.codegraph = {"command":["codegraph","serve","--mcp"],"enabled":true,"type":"local"}
  + skill.my-notes = …/.opencode/skills/my-notes -> …/homonto/skills/my-notes
```

`apply` resolves every secret first (aborting before any write if one fails),
then writes surgically — keeping every key it doesn't manage. `status` tells the
three states apart:

```console
$ homonto status
1 config change(s) awaiting apply (run `homonto apply`)   # you edited the toml
claude setting.model drifted (will reset on apply)        # disk changed outside homonto
No drift.                                                 # everything matches
```

**Secrets** are referenced, never stored: `${pass:path}` (via `pass`) or
`${ENV_VAR}`. `.homonto/state.json` holds only the token + a sha256 hash, so it's
safe to share. `homonto doctor` checks `pass`, tool dirs, and skill links.
`homonto import` bootstraps a starter toml from **Claude global MCP servers only**.

## The onto workflow

Install the framework via homonto, then apply:

```toml
[frameworks.onto]
source = "builtin:onto"
scope = "project"

[models.claude.architectural]        # a tool with a framework needs all three routes
model = "opus"
variant = "max"
[models.claude.coding]
model = "sonnet"
variant = "max"
[models.claude.trivial]
model = "haiku"
variant = "max"
```

```console
$ homonto apply --yes            # materializes the onto-* skills

$ onto init && onto new add-search
$ onto advance add-search        # open → design
$ onto advance add-search        # error: cannot leave "design": missing design.md
$ printf '# Design\n' > docs/changes/add-search/design.md
$ onto set isolation add-search branch
$ onto advance add-search        # design → build
```

Each transition needs that phase's deliverables (they accumulate):

| Leaving | Requires |
|---|---|
| `open` | `proposal.md`, `tasks.md` |
| `design` | + `design.md`, `isolation` set |
| `build` | + `plan.md` **and every `tasks.md` box checked** |
| `verify` | + `verification.md`, `verify-result = pass` |

`verify → close` also blocks on a dirty worktree. Close has its own evidence gates:

```console
$ onto close add-search          # error: change not merged (close.merged=false)
$ onto set close-merged add-search && onto set guides add-search updated
$ git add -A && git commit -q -m "close evidence"
$ onto close add-search          # archived to docs/changes/archive/2026-07-14-add-search
```

`close` also refuses while any dependency is unresolved (see `onto graph`).
Terminal states: `close` (success) and `onto abandon` (failure). Read-only:
`onto status`, `doctor`, `state --json`, `graph`.

## Supported / not supported (v0.1.0)

| Supported | Notes |
|---|---|
| MCP servers, settings, skills, plugins, marketplaces, TUI settings | Claude Code + OpenCode, full |
| Frameworks (`[frameworks.*]`) | **builtin catalog only**: `onto`, `comet`, `superpowers`, `openspec` |
| Commands, subagents (`builtin:` / `local:`) | subagents: `mode = link` (default) or `copy` |
| Remote **subagent** sources (`remote:…`) | **require `digest = "sha256:…"`**; fetched, verified, pinned, cached |
| Codex adapter | 🟡 pilot — **MCP only**, opt-in (`codex` in `targets`) → `~/.codex/config.toml` |
| `import` | 🟡 narrow — **Claude global MCP servers only** |

| Not supported (accepted for beta) | Detail |
|---|---|
| OpenCode JSONC comments | any apply that writes `opencode.jsonc` drops comments (no-op applies don't) |
| Remote *framework* sources | frameworks resolve via the builtin catalog only |
| Non-stdio MCP in `import` | url/http servers skipped with a warning |
| Secrets without a backend | `${pass:…}` needs `pass` on `PATH`; `${ENV_VAR}` needs the var set |
| Moving/renaming the repo | skill symlinks are absolute — reapply after a move |
| Adapters beyond Claude / OpenCode / Codex-MCP | none |
