# Troubleshooting & caveats

Known limitations of the beta line, common gotchas, and their workarounds.

## Building & installing

**`go build .` fails with `build output "homonto" already exists and is a
directory`.** The output name collides with the `homonto/` content directory
next to `main.go` — and `go build -o homonto .` silently deposits the binary
*inside* that directory. Use `go install .`, `go run .`, or build to an
explicit path outside the content dir:

```bash
go build -o ./bin/homonto .
```

**Version prints empty / wrong after `go install`.** Release builds stamp the
version at link time:

```bash
go install -ldflags "-X github.com/noviopenworks/homonto/internal/cli.Version=1.2.3" .
```

**Installed a newer binary but tools still have old content.** Installing a
binary does not touch projected content. Run `homonto update` — it
re-materializes the embedded catalog at the running version and re-projects it.
`onto doctor` reports a **version skew** finding when the `onto` binary and the
homonto that installed its framework have drifted apart.

**I changed a model route but my agents still show the old model.** Fixed in
v0.2.0. A subagent's `model:` is stamped from `[models.<tool>.<role>]` at
materialization, and materialization used to be gated on the catalog version
alone — so a route change left the rendered agents frozen while the tool's own
`setting.model` moved, giving two different answers from one config. Upgrade and
run `homonto apply`. On an older binary the workaround is to force a
re-materialization:

```bash
rm -rf .homonto/catalog && homonto apply --yes
```

## Scripting

**"My script captures nothing."** homonto and onto print through cobra, which
writes to **stderr**. Redirect with `2>&1`.

**Exit codes.** By default commands exit `0`/non-zero. The richer taxonomy is
opt-in: `plan --exit-code` exits `2` on pending changes; `status --exit-code`
exits `2` on pending and `3` on drift. `--output json` on `plan`, `status`, and
`doctor` gives machine-readable output; on the onto side, `state --json`,
`gate --json`, `scale --json`, and `graph --json` do.

## Projection

**OpenCode comments disappeared.** Claude's files are plain JSON, but
OpenCode's `opencode.jsonc` supports comments — and any apply that *writes*
that file rewrites it as normalized JSON, removing all comments. A skills-only
or otherwise no-op apply does not write the file, so comments survive those.
Accepted for beta.

**"Conflict" reported on a skill/subagent link.** homonto never clobbers a
file that isn't its own symlink — a real file or a link pointing elsewhere at
the target path is reported instead of overwritten. Move the conflicting file
out of the way and re-apply.

**I moved/renamed my homonto repo and now everything conflicts.** Skill
symlinks store an **absolute** target, so after a move the existing links
point at the old path and `apply`/`status` report conflicts rather than
silently repointing — homonto never changes a symlink it can't prove it owns.
Delete the stale links and re-run `apply` to relink at the new location.

**A tool file was reported unparseable.** That adapter aborts and reports —
homonto never overwrites a file it cannot parse. Fix the JSON by hand (or
restore it) and re-apply; the other tool's apply is unaffected.

**Something I configured by hand got pruned.** Only resources homonto recorded
in state are ever pruned — but note that a declared resource matching disk is
*adopted* into state (see
[projection & state](projection-and-state.md)), after which removing it from
`homonto.toml` removes it from the tool too. That is the contract: the TOML is
the source of truth for everything it declares.

## Secrets

**`apply` aborts with a resolution error.** `${pass:…}` needs
[`pass`](https://www.passwordstore.org/) on `PATH` (and the entry present);
`${ENV_VAR}` needs the variable set at apply time. Nothing was written — apply
resolves all secrets before touching any file. `homonto doctor` flags a
missing `pass`.

## Scope of the adapters

- **Claude Code and OpenCode** are the full adapters.
- **Codex** (OpenAI Codex CLI) is a pilot: it projects **MCP servers only**,
  into `~/.codex/config.toml` `[mcp_servers.<name>]`, and is **opt-in** — a
  resource must list `codex` in its `targets`. Listing `codex` on a subagent
  has no effect.
- **Frameworks resolve via the builtin catalog only** (`onto`, `comet`,
  `superpowers`, `openspec`) — there are no remote framework sources.
- **Remote sources exist for subagents only**, and require a
  `digest = "sha256:…"` pin (see
  [remote source trust](remote-source-trust.md)). homonto never re-resolves a
  pin to newer content; updating is a deliberate config edit.

## `import` is narrow

`homonto import` reads **Claude's global MCP servers only** (`~/.claude.json`
`mcpServers`). It skips non-stdio (url/http) servers with a warning, redacts
env values that *look* like secrets into `${pass:…}` references
(best-effort), copies `command`/`args` verbatim, and imports no skills,
plugins, settings, or OpenCode config. Review its output before applying or
committing. It refuses to overwrite an existing config without `--force`.

## onto

**`onto new`/`advance`/`close` refuse to run.** The mutating commands require
the onto framework to be installed *by homonto* (`[frameworks.onto]` +
`homonto apply`). The read-only commands (`status`, `state`, `gate`, `scale`,
`graph`, `handoff`, `doctor`, `version`) always work.

**`advance` fails.** The error names the gate: a missing artifact for the
current phase, an unset evidence token (e.g. `isolation` before build), an
unchecked `tasks.md` item leaving build, or a dirty worktree entering close.
Run `onto gate <change>` to see the pending decision(s) and the exact
`onto set` that records each one.

**`close` fails.** Check, in order: the change is at phase `close`;
`verify.result == pass`; `close.merged == true` (run `onto merge-deltas`);
guides resolved (full workflow); every `deps` entry archived; worktree clean;
archive target not already present.

**Repeated verify failures.** `onto set verify-result fail` increments a
counter; at ≥3 rounds `onto doctor` reports it — the workflow expects a human
decision (accept the deviation or keep fixing) rather than an endless loop.

**Recovering after context compaction.** `onto handoff <change>` emits a
compact recovery pack (`--write` persists it) so a fresh agent session can
resume without re-deriving state.
