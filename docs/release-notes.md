# Release notes intro

This file is prepended to every GitHub release's auto-generated notes by the
`release` workflow (`--notes-file docs/release-notes.md --generate-notes`), so
every release states the accepted limitations up front. Keep it short; the
per-release changelog is generated automatically below it.

---

## What's in this release

This release ships **three binaries** — `homonto` (config projector), `onto`
(spec-driven workflow operator), and `to` (minimal coding-framework
bookkeeper) — for every supported OS/arch as separate archives under one
`SHA256SUMS`. `onto` and `to` each require `homonto` to have installed their
framework first (`[frameworks.onto]` / `[frameworks.to]` + `homonto apply`).

### New in v0.6.1 — lossless per-tool agent rendering

An audit of the rendered agents against both tools' real contracts found
and fixed four silent information losses (catalog 0.6.0, onto 0.4.1,
to 0.3.1):

- **Claude renders a denylist, not an allowlist.** The old `tools:`
  allowlist silently stripped every unlisted default (WebFetch, WebSearch,
  Skill, …) that the OpenCode variant kept. Claude now gets
  `disallowedTools:` covering exactly the denied intent — read-only denies
  `Edit, Write, NotebookEdit`, `bash: false` denies `Bash`, `spawn: []`
  denies `Agent`/`Task` — matching OpenCode's deny-by-exception model.
- **`steps` now reaches Claude as `maxTurns`** (it was dropped as
  "no concept"; Claude has one).
- **`dialogs: false` is now enforced in OpenCode** (`question: deny`);
  omitting the line left the question tool available in defiance of the
  declared intent. All eight specialist subagents in both frameworks are
  now `dialogs: false` — matching the protocol's "a subagent never prompts
  the user; it returns a `Questions:` section" rule, which is also the only
  behavior Claude can express (AskUserQuestion is never available to Claude
  subagents). The onto orchestrator (primary) keeps its dialogs.
- **The unrecognized `mode:` line is gone from Claude variants** (Claude
  has no such frontmatter field).

### New in v0.6.0 — four model tiers, project-scoped model settings & MCPs, closed tier names

**`review` is the fourth model tier.** Model routes are now `architectural`
(orchestrate/design), `coding` (implement), `review` (judge others' work),
and `trivial` (cheap lookups) — and a model-backed config must declare all
four per enabled tool (**breaking**: existing three-route configs fail at
load until a `[models.<tool>.review]` block is added). The onto and to
reviewers and skeptics now run on the review tier instead of borrowing the
architectural one, in both Claude Code and OpenCode; the catalog is bumped
to 0.5.0 and re-materializes on the next apply.

**Route-derived default-model keys follow scope.** When every model-backed
resource (framework, command, subagent) enabled for a tool is
project-scoped, the `[models.<tool>.*]`-derived default-model keys now
project into the project-level config the tool merges over its global one
(`<repo>/opencode.jsonc` `model`/`small_model`;
`<repo>/.claude/settings.json` `model`) instead of the global file — one
repository's workflow models no longer become every other session's
defaults, and two repositories no longer fight over the same global keys.
Previously-applied global keys are pruned automatically on the next
`apply`. Any user-scope model-backed resource, and all explicit
`[settings.<tool>]` keys, keep today's global projection.

**MCP servers take a `scope`.** `[mcps.<name>] scope = "project"` projects
the server into the project-level config (Claude Code `<repo>/.mcp.json`;
OpenCode `<repo>/opencode.jsonc`) instead of the global one, so a
repository's servers no longer run in every other session. Default stays
`user` (global, today's behavior); codex remains user-scope only and a
project-scoped codex target fails at load. A previously-global server whose
scope changes migrates automatically on the next `apply`.

**Tier and role names are enforced.** `[models.<tool>.<level>]` with a
level outside `architectural`/`coding`/`trivial` now fails at load naming
the offender, and an agent frontmatter `role:` outside the same three tiers
fails at render — both were silent no-ops before (an unknown role rendered
the agent with no model at all).

### New in v0.5.1 — documentation rewrite

Docs only; the binaries are identical to v0.5.0. The README and every living
guide were rewritten for accuracy and directness: the source matrix is now
stated correctly everywhere (frameworks accept `builtin:`, `local:`, and
digest-pinned `remote:`; onto and `to` are mutually exclusive), stale
"`to` is planned" claims are gone, and the reference guides were re-checked
against the shipped binaries' command surfaces.

### New in v0.5.0 — live task lists, hardened `to`, principle guides

**The task list is live state — in both frameworks.** Discovered work is
appended to the checklist (with its files and verification) *before* its code
is written; checkoffs ride each task's own commit; tasks are never renumbered
or deleted (`SUPERSEDED` instead), so a fresh session always resumes from the
first unchecked task. onto gets this in onto-build, its templates, the
presets, and the subagent protocol (implementers report discovered work, the
coordinator appends it); `to` gets the same discipline adapted to its plan
contract.

**`to` grew teeth without growing ceremony:**

- **Plan contract**: every task carries `Files:` / `Change:` / `Verify:`
  fields plus a whole-change `Final Verify:` line; notes and verification
  evidence live in the same archived `plan.md`. `to doctor` diagnoses
  contract violations (line-numbered), wedged archives, and version skew;
  `--quiet` is the enforcement hook primitive.
- **Crash convergence**: an interrupted `done`/`abandon` no longer wedges a
  change — re-running the same command completes the archive.
- **Date-prefixed archives** (`docs/tasks/archive/<date>-<name>/`) free
  change names for reuse; mutating commands take a fail-fast workspace lock.
- **`to done --evidence "<text>"`** records what was asserted, verbatim and
  unchecked, so a real verification is distinguishable in the archive.
- **`to handoff`** now excerpts what a resuming session needs: the plan head,
  every unchecked task contract, `Final Verify:`, and bounded notes.

**Docs**: the `to` guides split into [workflow concepts](https://github.com/noviopenworks/homonto/blob/main/docs/guides/to-workflow.md)
and a [command reference](https://github.com/noviopenworks/homonto/blob/main/docs/guides/to-reference.md)
(mirroring onto's pair), and two principle guides —
[YAGNI](https://github.com/noviopenworks/homonto/blob/main/docs/guides/yagni.md) and
[KISS](https://github.com/noviopenworks/homonto/blob/main/docs/guides/kiss.md) —
map where each framework structurally enforces building only what's needed,
simply. Framework versions: onto 0.3.2, to 0.2.0; catalog 0.4.0.

### New in v0.4.0 — the `to` framework

`to` is the minimal coding framework for LLMs: **plan → do → done**, a
bookkeeper binary (`init`, `new`, `status`, `phase`, `done --verified`,
`abandon`, `handoff`; structured `--json` output on each of those workflow
commands), and the `builtin:to` catalog
framework — a `/to` dispatcher, three phase skills, a vendored `to-no-slop`,
and four **sequential-only** specialist subagents adapted from onto. Changes
live under `docs/tasks/` and archive on done. Design and rationale:
`docs/to-framework-design.md`.

Two deliberate properties to know before adopting it:

- **onto and `to` are mutually exclusive.** Declaring both frameworks in one
  `homonto.toml` fails at load — pick one workflow per repository (onto for
  evidence-gated enterprise changes, `to` for simple development). There is no
  escalation path between their state formats.
- **`to done --verified` is self-asserted.** The binary records the checkbox;
  it observes no evidence. The verification rigor lives in the `to-done`
  skill (real verify run + a single adversarial skeptic pass), not in a gate.

### Breaking in v0.3.0 — comet, openspec, and superpowers removed

The catalog now ships **only homonto-native frameworks**: `onto` (and, since
v0.4.0, `to`) — plus the loose framework-agnostic
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
- **The bundled catalog ships only homonto-native content**: the `onto` and
  `to` frameworks (mutually exclusive) plus the loose framework-agnostic
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
[`docs/guides/troubleshooting.md`](https://github.com/noviopenworks/homonto/blob/main/docs/guides/troubleshooting.md) for details.
