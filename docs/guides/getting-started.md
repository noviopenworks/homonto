# Getting started

A hands-on walkthrough of `homonto` and `onto`. `homonto` projects your
`homonto.toml` into Claude Code and OpenCode, Terraform-style: `plan`, then
`apply`. `onto` gates a change through `open → design → build → verify →
close`. onto's mutating commands need the onto framework installed *by*
homonto first.

> A third binary, `to`, is the lightweight alternative to onto (`plan → do →
> done`, no gates). See the [to workflow guide](to-workflow.md) and the
> [to reference](to-reference.md). onto and `to` are an exclusive choice per
> repository; this walkthrough uses onto.

> Output goes to **stderr**. Redirect with `2>&1` when scripting.

## 1. Install

```bash
go install github.com/noviopenworks/homonto@latest           # homonto
go install github.com/noviopenworks/homonto/cmd/onto@latest  # onto
```

Or grab the prebuilt binaries and `SHA256SUMS` from the GitHub release
(Linux/macOS/Windows, amd64/arm64). From a checked-out repo use
`go install .`, not a bare `go build .`: the output name collides with the
`homonto/` content directory (see [troubleshooting](troubleshooting.md)).

Verify:

```console
$ homonto version
homonto v0.5.0
$ onto version
onto v0.5.0
```

## 2. homonto in five commands

```console
$ homonto init            # scaffold homonto.toml + .gitignore + .env.example (never overwrites)
$ $EDITOR homonto.toml    # declare MCPs / skills / plugins / settings
$ homonto plan            # show the diff — writes nothing, resolves no secrets
$ homonto apply           # confirm [y/N] (--yes to skip), then write atomically
$ homonto status          # report drift / pending / clean
```

A realistic first `homonto.toml`:

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

`plan` prints a Terraform-style diff (`+` create, `~` update, `-` delete) and
leaves secrets as unresolved tokens:

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

`apply` resolves every secret first and aborts before any write if one fails,
then writes surgically, keeping every key it does not manage. `status` tells
the three states apart:

```console
$ homonto status
1 config change(s) awaiting apply (run `homonto apply`)   # you edited the toml
claude setting.model drifted (will reset on apply)        # disk changed outside homonto
No drift.                                                 # everything matches
```

**Secrets** are referenced, never stored: `${pass:path}` (via
[`pass`](https://www.passwordstore.org/)) or `${ENV_VAR}`.
`.homonto/state.json` holds only the token plus a sha256 hash, so it is safe
to share. See [secrets](secrets.md).

**Health check:** `homonto doctor` verifies `pass` is on `PATH`, the tool
config locations exist, and each owned skill has intact content and both tool
links.

**Already using Claude Code?** `homonto import` bootstraps a starter toml from
Claude's global MCP servers only. It is experimental and narrow; review its
output before applying.

## 3. Your first owned skill

Skills you author live under `homonto/skills/` next to `homonto.toml` and are
**symlinked** into each tool, so editing the source is instantly live
everywhere:

```console
$ mkdir -p homonto/skills/my-notes
$ printf -- '---\nname: my-notes\ndescription: My note conventions\n---\n' > homonto/skills/my-notes/SKILL.md
$ homonto apply --yes
```

Each skill declares its own `scope` (required, no default): `user` links into
`~/.claude/skills/` and `~/.config/opencode/skills/`; `project` links into the
repo itself (`.claude/skills/`, `.opencode/skills/`). Switching scope is
clean: `plan` shows the link relocating, and `apply` removes the old link as
it creates the new one.

## 4. The onto workflow

Install the framework via homonto, then apply. Every subagent the framework
expands must declare **an explicit per-tool model**:

```toml
[frameworks.onto]
source = "builtin:onto"
scope = "project"

[subagents.onto.claude]
model = "opus"
[subagents.onto-explorer.claude]
model = "haiku"
[subagents.onto-reviewer.claude]
model = "opus"
[subagents.onto-implementer.claude]
model = "sonnet"
[subagents.onto-skeptic.claude]
model = "opus"
# targeting opencode too? add [subagents.<name>.opencode] blocks as well
```

```console
$ homonto apply --yes            # materializes the onto-* skills, commands, subagents

$ onto init && onto new add-search
$ onto advance add-search        # open → design
$ onto advance add-search        # error: cannot leave "design": missing design.md
$ printf '# Design\n' > docs/changes/add-search/design.md
$ onto set isolation add-search branch
$ onto advance add-search        # design → build
```

Each transition needs that phase's deliverables. They accumulate; this table
is the `full` workflow, and the `fix`/`tweak` presets run a reduced path:

| Leaving | Requires |
|---|---|
| `open` | `proposal.md` |
| `design` | + `design.md`, `tasks.md`, `isolation` set |
| `build` | + `plan.md` **and every `tasks.md` box checked** |
| `verify` | + `verification.md`, `verify-result = pass` |

`verify → close` also blocks on uncommitted work: this change's own artifacts
or any source path, but not another change's in-flight docs (`onto dirt
<change>` classifies each path, and the refusal names what blocks). Close has
its own evidence gates:

```console
$ onto close add-search          # error: change not merged (close.merged=false)
$ onto set close-merged add-search && onto set guides add-search updated
$ git add -A && git commit -q -m "close evidence"
$ onto close add-search          # archived to docs/changes/archive/2026-07-14-add-search
```

`close` also refuses while any dependency is unresolved (see `onto graph`).
Terminal states: archived via `onto close` (success) and `onto abandon`
(failure). Read-only inspectors: `onto status`, `doctor`, `state --json`,
`gate --json`, `scale`, `graph`, `handoff`, `dirt`.

Concepts and the skills side: [the onto workflow](onto-workflow.md). Every
command and gate: [onto reference](onto-reference.md).

## 5. Supported / not supported

| Supported | Notes |
|---|---|
| MCP servers, settings, skills, plugins, marketplaces, TUI settings | Claude Code + OpenCode, full |
| Frameworks (`[frameworks.*]`) | builtin `onto` or `to` (mutually exclusive); also `local:` roots and digest-pinned `remote:` sources |
| Commands, subagents (`builtin:` / `local:`) | subagents: `mode = link` (default) or `copy` |
| Remote sources (`remote:…`) | subagents and frameworks; **require `digest = "sha256:…"`**; fetched, verified, pinned, cached |
| Codex adapter | 🟡 pilot — **MCP only**, opt-in (`codex` in `targets`) → `~/.codex/config.toml` |
| `import` | 🟡 narrow — **Claude global MCP servers only** |

| Not supported (accepted for beta) | Detail |
|---|---|
| OpenCode JSONC comments | any apply that writes `opencode.jsonc` drops comments (no-op applies don't) |
| Non-stdio MCP in `import` | url/http servers skipped with a warning |
| Secrets without a backend | `${pass:…}` needs `pass` on `PATH`; `${ENV_VAR}` needs the var set |
| Moving/renaming the repo | skill symlinks are absolute — delete stale links and reapply after a move |
| Adapters beyond Claude / OpenCode / Codex-MCP | none |

## Where to next

- [Configuration reference](configuration.md) — every table and field of `homonto.toml`.
- [homonto CLI reference](cli-reference.md) — flags, exit codes, examples.
- [Projection & state](projection-and-state.md) — how apply, drift, adoption, and pruning work.
- [Troubleshooting & caveats](troubleshooting.md) — when something looks wrong.
