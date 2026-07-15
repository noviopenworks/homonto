# Release notes intro

This file is prepended to every GitHub release's auto-generated notes by the
`release` workflow (`--notes-file docs/release-notes.md --generate-notes`), so
every release states the accepted limitations up front. Keep it short; the
per-release changelog is generated automatically below it.

---

## What's in this release

This release ships **two binaries** — `homonto` (config projector) and `onto`
(spec-driven workflow operator) — for every supported OS/arch as separate
archives under one `SHA256SUMS`. `onto` requires `homonto` to have installed the
`onto` framework first (`[frameworks.onto]` + `homonto apply`).

### Breaking in v0.3.0 — comet, openspec, and superpowers removed

The catalog now ships **only homonto-native frameworks**: `onto` today, with a
second framework (`to`) planned — plus the loose framework-agnostic
skills/commands (`handoff`, `grilling`), which are a separate channel and
unaffected. A config declaring `[frameworks.comet]`, `[frameworks.openspec]`,
`[frameworks.superpowers]`, or `builtin:comet-navigator` now fails at load
with `catalog: unknown framework` / `unknown subagent`; remove the
declaration (their projected links are pruned on the next apply) or vendor
the content yourself via a `local:` framework / pinned `remote:` source.
v0.2.2 is the last release carrying them. Rationale: ADR 0015.

### New in v0.2.2 — dirty-workspace support

The close gate no longer treats every uncommitted path the same. `onto dirt
[change] [--json]` classifies each dirty path — `own` (the change's own
`docs/changes/<name>/` evidence), `change` (another change's docs), `source`
(everything else) — and `onto advance`/`onto close` now tolerate `change`
dirt: one change's in-flight artifacts no longer deadlock another change's
close. What does block (`own` + `source`) is listed right in the refusal
instead of a bare "dirty worktree blocks close". The onto skills gained a
dirty-workspace protocol (attribution stays with the agent; the binary owns
classification).

### Fixed in v0.2.1 — deep-review findings

**onto's terminal states are now actually terminal.** An abandoned change could
archive as a success, have its evidence tokens forged via `onto set`, and merge
its never-accepted deltas into the living specs; all three paths now refuse.
`merge-deltas` recovers from a crash between its per-file writes instead of
wedging the change forever; `onto scale` errors without a recorded base ref
instead of silently measuring an empty diff as "light"; dependency resolution
is an exact name match (dep `auth` is no longer satisfied by an archive named
`…-refactor-auth`); a close crash can no longer leave `archived: true` at the
original path; `doctor` skips abandoned changes and `--quiet` is now fully
quiet.

**homonto re-materializes when framework CONTENT changes.** Editing a `local:`
framework's resources — or repinning a `remote:` framework's digest, which is
how a patched resource ships — used to be ignored forever ("No changes"). The
materialize gate now digests source content. Related: `plan` surfaces a pending
re-materialization (text + `--exit-code` 2) instead of disagreeing with apply;
renamed/de-declared resources are GC'd from `.homonto/catalog/` instead of
lingering where the adapters' variant-preference could resurrect them; and a
per-subagent model override is validated no matter what the entry's `targets`
say (an unvalidated value could previously reach a live agent file), with
conflicting overrides for one builtin now a deterministic load error.

### Breaking in v0.2.0 — `effort` and `variant` now do something

They were **required by validation and projected nowhere**: homonto forced you
to write two fields it then discarded — and never checked, so real configs
filled up with values no tool accepts (`effort = "normal"`, `variant = "max"`,
even `effort = "n"`). Now they are **optional, validated, and actually
projected** into each tool's own dialect:

| | Claude Code | OpenCode |
|---|---|---|
| `variant` | rendered *into* the model string (`opus[1m]`); **alias-only**, `1m` is the only documented one | a first-class `variant:` field, any provider-defined value |
| `effort` | a real field: `low`, `medium`, `high`, `xhigh`, `max` | **no such concept** — declaring it is now an error |

**You may need to edit your config.** A route naming just a `model` is now
complete, so the simplest fix is to delete values you were only writing to
satisfy the old rule. Otherwise the loader tells you exactly what is wrong:

```
parse config: models.claude.coding effort "normal" is not a Claude effort level (low, medium, high, xhigh, max)
parse config: models.opencode.coding sets effort "high", but OpenCode has no effort setting — use variant, or drop it
```

**New:** retune one agent without restating its tier — each field wins field by
field, and no `source` is needed for an agent a framework installed:

```toml
[subagents.onto-skeptic.claude]
effort = "max"
```

### Breaking in v0.2.0 — onto's subagents are namespaced `onto-*`

Every resource the onto framework ships is now namespaced, so installing onto
cannot collide with another framework's — or your own — agent of the same
generic name. Two builtin subagents were renamed:

| Old | New |
|---|---|
| `builtin:code-reviewer` | `builtin:onto-reviewer` |
| `builtin:codebase-explorer` | `builtin:onto-explorer` |

If you declare either **standalone** in a `[subagents.*]` table, update its
`source` — an old name now fails at load with `catalog: unknown subagent`. If
you install them via `[frameworks.onto]`, apply handles the rename for you: the
old agent files are pruned and the new ones projected. (The onto skills, its
commands, and the `onto` dispatcher itself are unchanged; `onto` is the
namespace root.)

### Fixed in v0.2.0 — subagents now track their model routes

Changing a `[models.<tool>.<role>]` route did **not** re-render the subagents
stamped from it. The projected agents stayed frozen at the model they were first
materialized with, while the tool's own `setting.model` — re-read from the routes
on every apply — moved correctly: one config, two different answers. If you have
edited a model route since installing a framework or subagent, **upgrade and run
`homonto apply`** to re-stamp your agents; verify with
`grep '^model:' .homonto/catalog/subagents/*.md`.

Three related defects went with it: a deleted rendered agent variant is now
restored instead of stranding a dangling symlink that `plan`/`status`/`doctor`
all called healthy; `apply` now re-materializes the catalog even when the
projection plan is empty; and `doctor` no longer reports a permanent, unfixable
finding for an OpenCode-primary agent's by-design absent Claude variant.

## Known limitations

homonto is a young, deliberately narrow tool. For the v0.3 beta line:

- **OpenCode JSONC comments are not preserved** on any apply that writes
  `opencode.jsonc` (the file is rewritten as normalized JSON). Accepted for beta.
- **`import` is a narrow Claude MCP bootstrap** — Claude global MCP servers only,
  best-effort secret redaction, no skills/plugins/settings/OpenCode import.
- **The bundled catalog ships only homonto-native content**: the `onto`
  framework (a second, `to`, is planned) plus the loose framework-agnostic
  skills/commands. Frameworks resolve from the bundled catalog or a `local:`
  path only — there are no remote *framework* sources. Remote sources exist for
  **subagents** only, and require a `digest = "sha256:…"` pin; homonto never
  re-resolves a pin to newer content on its own.
- **Two full adapters:** Claude Code and OpenCode. **Codex** is an opt-in pilot
  that projects **MCP servers only**.
- **Secrets require `pass` or an env var** at apply time (`${pass:...}` /
  `${ENV_VAR}`).
- **Moving or renaming the repo** breaks skill symlinks (absolute targets):
  delete the stale links and re-apply.

See the README's "Caveats" section and
[`docs/guides/troubleshooting.md`](docs/guides/troubleshooting.md) for details.
