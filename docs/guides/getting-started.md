# Getting started (v0.1.0)

A hands-on walkthrough of both binaries with real command output, followed by a
**supported / not-supported** matrix. If you want the reference-style details
instead, read [`using-homonto.md`](using-homonto.md) (homonto) and
[`onto-workflow.md`](onto-workflow.md) (onto).

> Every transcript below is real output from the v0.1.0 binaries. `homonto`
> writes through cobra to **stderr**, so redirect with `2>&1` when scripting.

---

## What the two binaries are

| Binary | Role | One-line mental model |
|---|---|---|
| `homonto` | Declarative config **projector** for AI coding tools | Terraform for your Claude Code / OpenCode config: edit `homonto.toml` → `plan` → `apply` |
| `onto` | Spec-driven workflow **operator** | A state machine that gates a change through `open → design → build → verify → close` |

`onto`'s mutating commands need the `onto` framework installed *by* `homonto`
first — so you always start with `homonto`.

---

## Install

```bash
# A tagged release (recommended):
go install github.com/noviopenworks/homonto@v0.1.0          # homonto
go install github.com/noviopenworks/homonto/cmd/onto@v0.1.0 # onto

# …or grab the prebuilt binaries + SHA256SUMS from the GitHub release
# (linux/macOS/windows, amd64/arm64).
```

```console
$ homonto version
homonto v0.1.0
$ onto version
onto v0.1.0
```

> Building from a checked-out repo: use `go install .` / `go build ./...`, **not**
> a bare `go build .` — the output name `homonto` collides with the `homonto/`
> content directory next to `main.go`.

---

## Part 1 — homonto in five commands

### 1. Scaffold

```console
$ homonto init
created .gitignore
created .env.example
created homonto.toml
created homonto/skills/.gitkeep
```

`init` never overwrites existing files. It leaves you a fully-commented
`homonto.toml` template. Nothing is projected yet.

### 2. Declare what you want

Edit `homonto.toml`. All sections are optional; here is a realistic starter:

```toml
[mcps.codegraph]
command = ["codegraph", "serve", "--mcp"]      # applies to both tools by default

[mcps.brave]
command = ["npx", "-y", "@modelcontextprotocol/server-brave-search"]
env = { BRAVE_API_KEY = "${BRAVE_API_KEY}" }    # a reference, never a literal secret
targets = ["claude"]                            # restrict to Claude Code only

[skills.my-notes]
source = "local:my-notes"                       # → homonto/skills/my-notes/
scope = "project"                               # required: user | project

[settings.claude]
model = "opus"
```

### 3. Preview — `plan` writes nothing and resolves no secrets

```console
$ homonto plan
claude:
  + mcp.brave = {"args":["-y","@modelcontextprotocol/server-brave-search"],"command":"npx","env":{"BRAVE_API_KEY":"${BRAVE_API_KEY}"},"type":"stdio"}
  + mcp.codegraph = {"args":["serve","--mcp"],"command":"codegraph","type":"stdio"}
  + setting.model = "opus"
  + skill.my-notes = …/.claude/skills/my-notes -> …/homonto/skills/my-notes
opencode:
  + mcp.codegraph = {"command":["codegraph","serve","--mcp"],"enabled":true,"type":"local"}
  + skill.my-notes = …/.opencode/skills/my-notes -> …/homonto/skills/my-notes
```

Note the secret stays a `${BRAVE_API_KEY}` token in the plan, and `brave` only
appears under `claude` (its `targets`). `+` = create, `~` = update, `-` = delete;
unchanged keys are silent.

### 4. Apply — confirm, then write atomically

```console
$ homonto apply           # add --yes to skip the [y/N] prompt
…same diff as plan…
Applied.
```

`apply` is two-phase: it resolves **every** secret first (aborting before any
write if one fails), then writes each file atomically (temp + rename). Managed
values land surgically — Claude MCP servers in `~/.claude.json`, settings in
`~/.claude/settings.json` — preserving every key homonto doesn't manage:

```console
$ cat ~/.claude/settings.json
{"model": "opus"}
```

### 5. `status` — drift vs. pending vs. clean

`status` distinguishes two different situations:

```console
# You edited homonto.toml but haven't applied — disk still matches last apply:
$ homonto status
1 config change(s) awaiting apply (run `homonto apply`)

# Something changed a *managed* value on disk, outside homonto, since last apply:
$ homonto status
claude setting.model drifted (will reset on apply)

# Everything matches:
$ homonto status
No drift.
```

### `doctor` — environment health

```console
$ homonto doctor
warn: `pass` not found on PATH (pass: references will fail)
ok: .claude (Claude Code) config location present
ok: .config/opencode (OpenCode) config location present
```

### Secrets — referenced, never stored

- `${pass:path/to/secret}` — resolved via [`pass`](https://www.passwordstore.org/).
- `${ENV_VAR}` — resolved from the environment (errors if unset).

`.homonto/state.json` records only the **unresolved token + a sha256 hash** of
the applied value — never plaintext — so it is safe to share, repeat applies are
no-ops, and out-of-band changes are still caught as drift. The scaffolded
`.gitignore` excludes `.homonto/` by default.

### Bootstrapping from an existing setup

```console
$ homonto import        # reads ~/.claude.json mcpServers into a starter homonto.toml
```

Deliberately narrow: **Claude global MCP servers only**, best-effort secret
redaction into `${pass:…}`, `command`/`args` copied verbatim. Review before use.

---

## Part 2 — the onto workflow

`onto` gates a change through five phases. The **binary owns the state and the
gates**; the `onto-*` skills (installed by `homonto apply`) drive the work inside
each phase.

```
open → design → build → verify → close
```

### 0. Install the framework via homonto

```toml
# homonto.toml
[frameworks.onto]
source = "builtin:onto"
scope = "project"

# A tool that gains a framework must declare all three model routes:
[models.claude.architectural]
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
$ homonto apply --yes     # materializes the onto-* skills into your tools
```

The read-only `onto` commands (`status`, `doctor`, `graph`, `state`, `version`)
work without any of this. The mutating ones (`init`, `new`, `advance`, `close`)
require the framework to be applied first.

### 1. Scaffold and create a change

```console
$ onto init
created docs/changes
created docs/specs
created docs/adr
created docs/guides

$ onto new add-search
created change "add-search" at docs/changes/add-search
  docs/changes/add-search/onto-state.yaml
  docs/changes/add-search/proposal.md
  docs/changes/add-search/tasks.md

$ onto status
add-search: open — skeleton ok
```

### 2. Advance through the gates

Each transition only succeeds once that phase's deliverables exist. The gate
**refuses and leaves the phase unchanged** otherwise:

```console
$ onto advance add-search
warning: worktree has uncommitted changes
add-search: open → design

$ onto advance add-search                 # design has no design.md yet
error: onto advance: cannot leave "design": missing design.md
```

Produce the deliverable, record the required state, then advance:

```console
$ printf '# Design\n' > docs/changes/add-search/design.md
$ onto set isolation add-search branch    # build requires a chosen isolation
add-search: updated
$ onto advance add-search
add-search: design → build
```

The required artifacts **accumulate** as a change advances:

| Leaving phase | Requires |
|---|---|
| `open` | `proposal.md`, `tasks.md` |
| `design` | + `design.md`, `isolation` set (`branch` or `worktree`) |
| `build` | + `plan.md` **and every `tasks.md` checkbox checked** |
| `verify` | + `verification.md`, `verify-result` = `pass` |

`verify → close` additionally **blocks on a dirty worktree** (a normal
transition only warns).

### 3. Inspect state and dependencies

```console
$ onto state add-search --json | head
{
    "schema_version": 1,
    "change": "add-search",
    "phase": "build",
    "workflow": "full",
    "isolation": "branch",
    ...
}

$ onto graph
add-search (407c1865, close)
```

### 4. Close (archive)

`close` has its own evidence gates beyond reaching the `close` phase:

```console
$ onto close add-search
error: onto close: change not merged (close.merged=false); mark the change merged before close

$ onto set close-merged add-search
$ onto set guides add-search updated
$ git add -A && git commit -q -m "add-search close evidence"
$ onto close add-search
add-search: archived to docs/changes/archive/2026-07-14-add-search
```

`close` also refuses while any dependency listed in `onto-state.yaml` is still
unresolved (no archived `docs/changes/archive/*-<dep>` exists) — that is the
dependency-aware ordering the `graph` command surfaces.

Two terminal states exist: `close` (success, archived) and `onto abandon <change>`
(the unsuccessful terminal state).

```console
$ onto doctor
healthy
```

---

## What is supported / what is not (v0.1.0)

### homonto — resources and adapters

| Capability | Status | Notes |
|---|---|---|
| MCP servers (`[mcps.*]`) | ✅ | stdio servers; `targets` restricts which tools |
| Settings (`[settings.claude]` / `[settings.opencode]`) | ✅ | surgically merged into each tool's settings file |
| Owned skills (`[skills.*]`) | ✅ | **symlinked** from `homonto/skills/<name>`; `scope` required (`user`/`project`) |
| Plugins (`[plugins.claude.*]` / `[plugins.opencode.*]`) | ✅ | Claude marketplaces + `pluginConfigs`; OpenCode `plugin` array |
| Marketplaces (`[marketplaces.claude.*]`) | ✅ | Claude only → `extraKnownMarketplaces` |
| TUI settings (`[tui.opencode]`) | ✅ | separate `~/.config/opencode/tui.json` |
| Frameworks (`[frameworks.*]`) | ✅ | **builtin catalog only**: `onto`, `comet`, `superpowers`, `openspec` |
| Commands (`[commands.*]`) | ✅ | `builtin:` / `local:`; single-file, linked into each tool |
| Subagents (`[subagents.*]`) | ✅ | `builtin:` / `local:`, `mode = link` (default) or `copy` |
| Remote subagent sources (`source = "remote:…"`) | ✅ | **requires `digest = "sha256:…"`**; fetched, verified, pinned, cached; see [`remote-source-trust.md`](remote-source-trust.md) |
| Adapter: **Claude Code** | ✅ full | all resource types |
| Adapter: **OpenCode** | ✅ full | all resource types |
| Adapter: **Codex** (OpenAI Codex CLI) | 🟡 pilot | **MCP servers only**, opt-in — a resource must list `codex` in `targets`; writes `~/.codex/config.toml` |
| `import` bootstrap | 🟡 narrow | **Claude global MCP servers only**; no skills/plugins/settings/OpenCode import |

### Not supported / accepted limitations in v0.1.0

| Not supported | Detail |
|---|---|
| Preserving **OpenCode JSONC comments** | any apply that *writes* `opencode.jsonc` rewrites it as normalized JSON — comments are dropped (a no-op / skills-only apply doesn't touch the file) |
| **Remote *framework* sources** | frameworks resolve through the bundled builtin catalog only (remote pinning applies to subagents, not frameworks) |
| **Non-stdio MCP** in `import` | url/http servers are skipped with a warning |
| **Secrets without a backend** | `${pass:…}` needs `pass` on `PATH`; `${ENV_VAR}` needs the var set at apply time |
| **Moving/renaming the homonto repo** | skill symlinks store absolute targets; after a move, `apply`/`status` report a conflict (never silently repoint) — delete the stale links and re-apply |
| Adapters beyond the three above | only Claude Code, OpenCode, and the Codex MCP pilot exist |

### onto — supported

| Capability | Status |
|---|---|
| Five-phase lifecycle with accumulating artifact gates | ✅ |
| Isolation / verify-result / close-merged / guides evidence gates | ✅ |
| Dirty-worktree gate (blocks `verify → close` and `close`) | ✅ |
| Dependency-aware close ordering + `graph` | ✅ |
| Archive on close; `abandon` terminal state | ✅ |
| Read-only `status` / `doctor` / `state --json` (config-independent) | ✅ |
| Requires the `onto` framework applied by homonto for mutating commands | ⚠️ by design |

---

## Where to go next

- [`using-homonto.md`](using-homonto.md) — full homonto reference
- [`onto-workflow.md`](onto-workflow.md) — full onto reference
- [`status-and-adoption.md`](status-and-adoption.md) — drift, pending, adoption, pruning
- [`remote-source-trust.md`](remote-source-trust.md) — how pinned remote sources are verified
