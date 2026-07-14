# Subagents — how they work

A **subagent** is an agent definition (a markdown file with frontmatter) that
homonto projects into each tool's agent directory. Subagents are declared as
`[subagents.<name>]` resources and are fully **declarative** — reconciled by
`plan` / `apply` / `status` / `doctor` exactly like skills and commands. There is
no separate imperative "agents" command group.

```toml
[subagents.code-reviewer]
source = "builtin:code-reviewer"   # builtin | local | remote
scope  = "project"                 # user | project (default: project)
mode   = "link"                    # link (default) | copy
targets = ["claude", "opencode"]   # optional; default: both
```

## Sources

| `source` | Resolves from | Notes |
|---|---|---|
| `builtin:<name>` | the bundled catalog (`.homonto/catalog/subagents/<name>.md`) | ships: `code-reviewer`, `codebase-explorer`, `comet-navigator` |
| `local:<name>` | `homonto/subagents/<name>.md` (next to `homonto.toml`) | your own agent files |
| `remote:<url>` | a fetched, verified, cached archive (`.homonto/remote/subagents/<name>.md`) | **requires a `digest` pin** — see below |

Frameworks can also declare their own subagents; those materialize and project
the same way (e.g. comet ships `comet-navigator`).

## link vs copy mode

- **`mode = "link"` (default)** — the agent file is **symlinked** into each
  tool's agent directory. Editing the catalog/local source is instantly live
  everywhere. `apply` never clobbers a real file or a foreign symlink — it
  reports a conflict instead.
- **`mode = "copy"`** — the agent is projected as a **real managed file** you can
  edit in place. `apply` keeps it in sync, detects drift against a recorded
  content hash, and **backs up a local edit to `<path>.bak` before overwriting**.
  De-declaring it prunes the file.

The legacy `[agents.<name>]` table still parses but folds into a copy-mode
`[subagents.<name>]` at load.

## Where they land — scope and targets

`scope` selects the directory (default `project`); `targets` selects the tools
(default both):

| Tool | `scope = "user"` | `scope = "project"` |
|---|---|---|
| Claude Code | `~/.claude/agents/<name>.md` | `<repo>/.claude/agents/<name>.md` |
| OpenCode | `~/.config/opencode/agent/<name>.md` | `<repo>/.opencode/agent/<name>.md` |

Subagents project into **Claude Code and OpenCode only**. The Codex pilot adapter
handles MCP servers only, so listing `codex` in a subagent's `targets` has no
effect.

## Model routes are required

A tool that gains a subagent (or a framework/command) must declare **all three**
model routes for that tool — `[models.<tool>.architectural]`,
`[models.<tool>.coding]`, `[models.<tool>.trivial]`. A partial set is rejected at
load, naming the offender. This is validated for every target the subagent
projects into.

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
```

## The agent file

The projected file is materialized **verbatim**. A subagent's frontmatter uses
the agent format, e.g.:

```markdown
---
name: code-reviewer
description: Use to review a diff or set of changes for correctness, security,
  and clarity before merging; reports findings ranked by severity.
mode: subagent
---

# Instructions for the agent…
```

## Per-tool frontmatter (read-only in both tools)

Claude Code and OpenCode express a subagent's tool access differently, and the
two forms **cannot share one file** — Claude uses a `tools:` allowlist string
while OpenCode uses a `permission:` map, and OpenCode rejects a string `tools:`.
So a builtin subagent that must be enforced-read-only in both tools declares its
intent once, tool-neutrally, in a `homonto:` frontmatter block:

```markdown
---
name: code-reviewer
description: ...
mode: subagent
homonto:
  read_only: true   # deny edits/writes
  bash: false       # optional; false denies bash too (default: allowed)
  dialogs: true     # allow the interactive question/dialog tool
---
<prompt body>
```

On `apply`, homonto renders that block into each tool's native fields and links
each adapter to its own variant (`<name>.claude.md` / `<name>.opencode.md` under
the materialized catalog):

| Neutral intent | Claude (`tools:` allowlist) | OpenCode (`permission:` map) |
|---|---|---|
| `read_only: true` | omit `Edit`/`Write` (e.g. `Read, Grep, Glob`) | `edit: deny` |
| `bash: false` | omit `Bash` | `bash: deny` |
| `dialogs: true` | (AskUserQuestion is built in) | `question: allow` |

The prompt body is single-source (never duplicated), and the neutral block and
its comments are stripped from the rendered files. Subagents without a `homonto:`
block are projected verbatim (a plain symlink to the shared file), unchanged.

## Remote subagents are pinned and fail-closed

A `remote:` source **requires** `digest = "sha256:<64 hex>"`. On `apply`, homonto
fetches the archive → validates it (rejecting path traversal, symlinks, and
decompression bombs) → matches the digest pin → checks revocation → caches it,
and writes a tool file **only after every check passes**:

```toml
[subagents.reviewer]
source = "remote:https://example.com/reviewer.tar.gz"
digest = "sha256:…"                # REQUIRED; verified before any write
scope  = "project"
```

Pins are recorded in `.homonto/remote.lock.json`; content is cached under
`.homonto/cache/remote/` for offline, reproducible applies. See
[`remote-source-trust.md`](remote-source-trust.md).

## Lifecycle

- **plan / apply** — create, update, or delete the projected agent as the
  declaration changes; each write is atomic.
- **status** — reports drift (a managed agent changed on disk) and pending
  edits (declaration changed but not yet applied).
- **prune** — remove a `[subagents.<name>]` block and the next `apply` deletes
  its projected file/link. Only resources homonto recorded in state are pruned.
- **doctor** — verifies each subagent's content plus **both tools' links**.
