# homonto — Design Spec

**Date:** 2026-06-24
**Status:** Approved for planning

## Summary

`homonto` is a personal Go CLI that acts as the single declarative source of
truth for AI coding-tool configuration. You describe your MCP servers, skills,
plugins, and settings once in `homonto.toml`; `homonto apply` projects them into
each target tool (Claude Code and OpenCode for v1) via a terraform-style
plan/confirm/apply pipeline. Tokens and credentials are never stored in the
repo — they are referenced (`${pass:…}` / `${ENV}`) and resolved only at apply
time.

## Goals (v1)

- One declarative config (`homonto.toml`) drives **MCPs, skills, plugins, and
  settings** across **both** Claude Code and OpenCode (breadth-first, common
  denominator depth).
- **Surgical merge**: homonto owns only the keys it manages and preserves all
  unmanaged keys in each tool's files.
- **Plan/apply** with a diff and confirmation step; idempotent re-apply.
- **Secrets by reference only** — nothing secret ever enters the repo.
- **Own local content, reference external**: authored skills/commands/rules/
  agents live in the repo and are linked into each tool; marketplace plugins and
  MCP servers are declared by reference.
- **`import`** to bootstrap the config from an existing setup.

## Non-goals (v1)

- Imperative `add`/`remove` CLI mutators (config changes happen by editing
  `homonto.toml`).
- Tools beyond Claude Code and OpenCode (adapter interface makes them additive).
- Encrypted in-repo secrets (we use reference-only; encryption could come later).
- Deep per-tool feature coverage beyond the common-denominator fields.

## Decisions (from brainstorming)

| Decision | Choice |
|---|---|
| Core model | Declarative source of truth (tools are generated outputs) |
| Secrets | Reference, never store (`${pass:…}`, `${ENV}` fallback) |
| File-based content | Own local (skills/commands/rules/agents), reference external (plugins/MCPs) |
| Apply safety | Surgical merge + terraform-style plan/confirm |
| v1 scope | All 4 concepts, both tools, thin |
| Config format | TOML |
| Secret backend | `pass` (primary) + `${ENV}` zero-dep fallback |
| Architecture | Normalized model + per-tool adapters |

## Architecture

Approach: **normalized desired-state model + per-tool adapters** with shared
services. The binary parses `homonto.toml` into one tool-agnostic `DesiredState`
struct. Everything downstream operates on that struct, never on raw TOML. Each
target tool is an `Adapter` with `Read() / Plan(desired) / Apply(changes)`.
Shared services: a **secret resolver**, a **content linker** (symlinks), and a
**planner/printer** (diff + confirm).

```
homonto.toml ──▶ Parse ──▶ DesiredState ──▶ [ ClaudeAdapter, OpenCodeAdapter ]
                                                     │ Read → Plan → Apply
shared: SecretResolver · ContentLinker · Planner/Printer · StateStore
```

Adding a new tool later = implement one `Adapter`. No engine changes.

### Repo layout

```
homonto/
├── homonto.toml              # single source of truth (committed)
├── .env.example              # documents required env vars (committed)
├── content/                  # content you OWN (committed)
│   ├── skills/<name>/
│   ├── commands/<name>.md
│   ├── rules/<name>.md
│   └── agents/<name>.md
└── .homonto/                 # local state (gitignored)
    └── state.json            # last-applied snapshot, for drift detection
```

### `homonto.toml` (shape)

```toml
[mcps.codegraph]
command = ["codegraph", "serve", "--mcp"]
targets = ["claude", "opencode"]          # default: all tools

[mcps.brave]
command = ["npx", "-y", "@modelcontextprotocol/server-brave-search"]
env = { BRAVE_API_KEY = "${pass:ai/brave}" }

[skills]
own = ["graphify", "comet"]               # from content/skills/

[plugins]
claude = ["claude-hud@official"]          # marketplace entries
opencode = ["@slkiser/opencode-quota"]    # npm packages

[settings.claude]
model = "opus"

[settings.opencode]
model = "anthropic/claude-opus-4-8"
```

Secret references are kept as **unresolved tokens** in the model and resolved
only at the last moment, so plans never print secrets.

## Data flow — the apply pipeline

Six stages, each independently testable:

1. **Parse** `homonto.toml` → `DesiredState`.
2. **Read** (per adapter) current state from the tool's actual files.
3. **Plan** (per adapter) `diff(desired, current)` → `[]Change` (secrets still
   unresolved tokens).
4. **Print plan** → terraform-style diff + confirm `[y/N]`. Secrets shown as
   `${pass:ai/brave}`, never resolved.
5. **Resolve** `${…}` tokens via `pass`/env — only for confirmed changes, and
   **all at once** before any write (two-phase).
6. **Apply** → atomic file writes + symlink owned content; update
   `.homonto/state.json` last.

Key properties:

- **Secrets resolve after confirm, never before** — plan output and logs are
  always safe to share; a no-op apply never touches `pass`.
- **Atomic writes** (temp + rename) — an interrupted apply never leaves a
  half-written file.
- **Drift detection** — `Read` compares current managed keys against
  `state.json`; out-of-band changes are flagged in the plan
  (`~ model: opus → sonnet (drifted, will reset to opus)`).
- **`homonto plan`** runs stages 1–4 only (pure dry run).
- **Idempotent** — a second apply with no changes prints `No changes` and exits
  without touching `pass` or any file.

## Adapters — concept → file mapping

| Concept | Claude Code | OpenCode |
|---|---|---|
| MCP | `~/.claude.json` → `mcpServers.<name>` | `opencode.jsonc` → `mcp.<name>` (`type:"local"`, `command:[…]`, `enabled`) |
| Skill (owned) | symlink `content/skills/<n>` → `~/.claude/skills/<n>` | symlink → `~/.config/opencode/skills/<n>` |
| Command/Rule/Agent (owned) | symlink into `~/.claude/{commands,rules,agents}/` | symlink into `~/.config/opencode/{commands,rules}/` |
| Plugin (referenced) | merge `enabledPlugins` + `extraKnownMarketplaces` in `settings.json` | merge npm pkg into `plugin[]` in `opencode.jsonc` |
| Settings | merge managed keys into `settings.json` | merge managed keys into `opencode.jsonc` |

Notes:

- **"Plugin" differs per tool** — Claude plugins are marketplace entries;
  OpenCode plugins are npm packages. They stay under separate per-tool keys in
  `homonto.toml` rather than being force-unified.
- **Owned content = symlinks, not copies** — editing `content/…` is instantly
  live in every tool; apply just ensures links exist and point correctly. A
  `--copy` flag may be added later.
- **JSONC caveat** — `opencode.jsonc` has comments. The adapter preserves all
  keys (managed + unmanaged) on merge, but inline comments in rewritten regions
  may not survive. Claude's files are plain JSON, so unaffected. Documented in
  the README.

## CLI surface

```
homonto init       scaffold a new repo (homonto.toml, content/, .gitignore, .env.example)
homonto import     bootstrap homonto.toml + content/ from the current Claude/OpenCode setup
homonto plan       dry run — show the diff, write nothing
homonto apply      plan → confirm → write (main command)
homonto status     show drift: tool files vs last-applied state
homonto doctor     health check: tools found? pass available? symlinks intact? dangling refs?
```

- **`import`** is the adoption on-ramp: read existing tool files, generate a
  starter `homonto.toml`, pull authored skills/commands into `content/`. One-time
  bootstrap, then live in declarative-land.
- **No imperative `add`/`remove`** — config changes happen by editing
  `homonto.toml` then `apply`. One source of truth.

## Error handling — fail safe, never half-apply

- **Two-phase apply**: resolve all secrets for confirmed changes first; if any
  reference is missing, abort **before writing any file** and name the missing
  ref.
- **Atomic per-file writes**; `state.json` written last so a crash leaves every
  individual file valid and the next apply reconciles.
- **Unparseable existing config** → that adapter aborts and reports; other tools
  proceed. Never overwrite a file homonto could not first read.
- **Symlink conflicts** (target exists and isn't our link) → reported, not
  clobbered; needs `--force` or manual fix.
- **Dangling references** (skill missing from `content/`, unknown marketplace) →
  surfaced at `plan`, before any write.
- **Missing tool** (Claude/OpenCode not installed) → adapter skipped with a
  notice; `doctor` reports it.

## Testing — TDD

- **Unit, table-driven** per stage: parser (TOML→model), resolver (mocked
  `pass`/env), planner (desired vs current → changes).
- **Golden-file tests** for surgical merge: fixture tool file + `homonto.toml` →
  asserted merged output, proving unmanaged keys/comments survive.
- **Integration**: `apply` against a temp `$HOME` with fake `~/.claude` +
  `~/.config/opencode`; assert files, symlinks, **idempotency** (2nd apply =
  no-op), and **drift detection**.
- **Secret-safety test**: assert `plan` output never contains a resolved secret
  value.

## Stack

Go + `cobra` (CLI), `pelletier/go-toml/v2` (TOML), `tidwall/sjson` for surgical
JSON edits (preserves the untouched document), `tailscale/hujson` to normalize
JSONC before editing. Plain `state.json`, no database.

## Open questions / future

- Encrypted in-repo secrets (age/sops) as an alternate backend.
- Additional secret resolvers (1Password `op`, OS keychain).
- More tool adapters (Codex, Cursor, Gemini) via the adapter interface.
- `--copy` mode for environments that dislike symlinks.
- Profiles / per-machine overlays.
