# Subagents

A **subagent** is an agent definition (a markdown file with frontmatter) that
homonto projects into each tool's agent directory. Subagents are declared as
`[subagents.<name>]` resources and are fully **declarative** — reconciled by
`plan` / `apply` / `status` / `doctor` exactly like skills and commands. There
is no separate imperative "agents" command group.

```toml
[subagents.onto-reviewer]
source = "builtin:onto-reviewer"   # builtin | local | remote
scope  = "project"                 # user | project (default: project)
mode   = "link"                    # link (default) | copy
targets = ["claude", "opencode"]   # optional; default: both
```

## Sources

| `source` | Resolves from | Notes |
|---|---|---|
| `builtin:<name>` | the bundled catalog (materialized at `.homonto/catalog/subagents/<name>.md`) | ships: `onto-reviewer`, `onto-explorer`, `onto-implementer`, `onto-skeptic` |
| `local:<name>` | `homonto/subagents/<name>.md` (next to `homonto.toml`) | your own agent files |
| `remote:<url>` | a fetched, verified, cached archive | **requires a `digest` pin** — see below |

Frameworks can also declare their own subagents; those materialize and project
the same way (e.g. onto ships `onto-implementer` plus the two specialists).
Don't re-declare a framework's subagent in a top-level `[subagents.*]` table —
the names collide.

## link vs. copy mode

- **`mode = "link"` (default)** — the agent file is **symlinked** into each
  tool's agent directory. Editing the catalog/local source is instantly live
  everywhere. `apply` never clobbers a real file or a foreign symlink — it
  reports a conflict instead.
- **`mode = "copy"`** — the agent is projected as a **real managed file** you
  can edit in place. `apply` keeps it in sync, detects drift against a recorded
  content hash, and **backs up a local edit to `<path>.bak` before
  overwriting**. De-declaring it prunes the file.

The legacy `[agents.<name>]` table still parses but folds into a copy-mode
`[subagents.<name>]` at load.

## Where they land — scope and targets

`scope` selects the directory (default `project`); `targets` selects the tools
(default both):

| Tool | `scope = "user"` | `scope = "project"` |
|---|---|---|
| Claude Code | `~/.claude/agents/<name>.md` | `<repo>/.claude/agents/<name>.md` |
| OpenCode | `~/.config/opencode/agent/<name>.md` | `<repo>/.opencode/agent/<name>.md` |

Subagents project into **Claude Code and OpenCode only**. The Codex pilot
adapter handles MCP servers only, so listing `codex` in a subagent's `targets`
has no effect.

## Model routes are required

A tool that gains a subagent (or a framework/command) must declare **all
three** model routes for that tool — `[models.<tool>.architectural]`,
`[models.<tool>.coding]`, `[models.<tool>.trivial]`. A partial set is rejected
at load, naming the offender. This is validated for every target the subagent
projects into. See the
[configuration reference](configuration.md#model-routes--modelstoolroute).

An agent's `role:` picks its tier. To retune **one** agent, add a per-tool block
under its name — each field wins over the tier field by field, so this keeps the
tier's model and only thinks harder:

```toml
[subagents.onto-skeptic.claude]
effort = "max"
```

No `source` is needed (or allowed) when the agent comes from a framework: a
block with no source *tunes* the agent rather than declaring it. `model`,
`variant`, and `effort` render into each tool's own dialect — Claude brackets a
variant into the model (`opus[1m]`) and takes `effort:`; OpenCode has a separate
`variant:` field and no effort at all.

## The agent file

The projected file is materialized **verbatim**. A subagent's frontmatter uses
the agent format:

```markdown
---
name: onto-reviewer
description: Use to review a diff or set of changes for correctness, security,
  and clarity before merging; reports findings ranked by severity.
mode: subagent
---

# Instructions for the agent…
```

## Per-tool frontmatter (the `homonto:` block)

Claude Code and OpenCode express an agent's capabilities differently, and the
two forms **cannot share one file** — Claude uses a `tools:` allowlist string
while OpenCode uses a `permission:` map and `mode`, and OpenCode rejects a
string `tools:`. So a builtin subagent declares its intent once,
tool-neutrally, in a `homonto:` frontmatter block, and `apply` renders each
tool's native dialect:

```markdown
---
name: onto-implementer
description: ...
mode: subagent
homonto:
  role: coding        # model tier → stamped from [models.<tool>.coding]
  read_only: false    # deny edits/writes when true
  bash: false         # optional; false denies bash (default: allowed)
  dialogs: true       # allow the interactive question/dialog tool
  spawn: []           # delegation topology: agents this one may dispatch
  primary: true       # OpenCode primary agent; the Claude variant is skipped
  steps: 60           # OpenCode iteration budget
---
<prompt body>
```

Rendering, by explicit parity tier:

| Neutral intent | Claude (`tools:` allowlist) | OpenCode (`permission:` / `mode`) |
|---|---|---|
| `read_only: true` | omit `Edit`/`Write` | `edit: deny` |
| `bash: false` | omit `Bash` | `bash: deny` |
| `dialogs: true` | (AskUserQuestion is built in) | `question: allow` |
| `role: <tier>` | `model: <claude route>` | `model: <opencode route>` |
| `spawn: []` | omit `Task` | `task: deny` |
| `spawn: [a,b]` | `Task` present (advisory) | `task:` globs allowing only `a`,`b` |
| `primary` / `steps` | *(no concept — Claude variant skipped)* | `mode: primary`, `steps:` |

`role` maps to the tool's model from the user's `[models.<tool>.<role>]` route
(so the same declaration yields `opus` in Claude and the OpenCode model id); a
missing route just omits `model:` (the agent inherits the default). The prompt
body is single-source (never duplicated); the neutral block and its comments
are stripped from the rendered files. Subagents without a `homonto:` block are
projected verbatim (a plain symlink to the shared file), unchanged.

The onto framework's three specialists show the division of labor: read-only
`onto-explorer` (trivial model) and `onto-reviewer` (architectural), and
the edit-capable `onto-implementer` (coding) — all `spawn: []` (they never
nest).

## Remote subagents are pinned and fail-closed

A `remote:` source **requires** `digest = "sha256:<64 hex>"`. On `apply`,
homonto fetches the archive → validates it (rejecting path traversal, symlinks,
and decompression bombs) → matches the digest pin → checks revocation → caches
it, and writes a tool file **only after every check passes**:

```toml
[subagents.reviewer]
source = "remote:https://example.com/reviewer.tar.gz"
digest = "sha256:…"                # REQUIRED; verified before any write
scope  = "project"
```

Pins are recorded in `.homonto/remote.lock.json`; content is cached under
`.homonto/cache/remote/` for offline, reproducible applies. See
[remote source trust](remote-source-trust.md).

## Lifecycle

- **plan / apply** — create, update, or delete the projected agent as the
  declaration changes; each write is atomic.
- **status** — reports drift (a managed agent changed on disk) and pending
  edits (declaration changed but not yet applied).
- **prune** — remove a `[subagents.<name>]` block and the next `apply` deletes
  its projected file/link. Only resources homonto recorded in state are pruned.
- **doctor** — verifies each subagent's content plus **both tools' links**.
