# Deep Review — homonto + onto (2026-07-04)

Four independent harsh reviewers (Go code audit with empirical
reproduction against the built binary, product/architecture critique,
onto-framework critique, docs-vs-reality audit) + synthesis. Nothing was
fixed as part of this review; this document is the record.

**Verdict: the tool does not work at its primary job, and nobody could
legally install it anyway.** Excellent secondary qualities — atomic
writes, happy-path secret hygiene, test discipline, documentation volume
— wrapped around a core that was never validated against reality. Every
verification this project ran (48 green tests, three adversarial verify
rounds, 30 fixed findings) checked the system against *its own
descriptions of itself*, never against the world. That is the root
cause; everything below is a symptom.

---

## The five findings that end the conversation

### 1. P0 — Claude adapter writes an invalid MCP schema

homonto projects `"command": ["npx", "-y", "..."]` into `mcpServers`.
Real Claude Code (verified against the live `~/.claude.json`:
`{"type":"stdio","command":"codegraph","args":["serve","--mcp"]}`)
requires `command` as a **string** with `args` separate. Every MCP
homonto manages for Claude Code is a server Claude cannot launch. Plan
looks clean, apply succeeds, status says "No drift" — nothing works.
It shipped because every test round-trips homonto's own fixtures; none
compares against a real tool file. (`internal/adapter/claude/claude.go`
`desired()`; the OpenCode array shape — correct for OpenCode — was
projected into Claude verbatim.)

### 2. P0 — `import` corrupts working configs

On a real Claude file, `importer.go:30-32` reads the string `command`
via gjson `.Array()` → one-element array, **silently dropping `args`**.
`{"command":"npx","args":["-y","@modelcontextprotocol/server-brave-search"]}`
becomes `command = ["npx"]`; apply then writes back the invalid shape
from finding 1. The advertised bootstrap (`import` → `apply`) takes a
working machine and breaks every MCP server on it, behind a confirm
prompt the user has no reason to distrust.

### 3. CRITICAL — `plan` prints resolved secrets when state is missing (reproduced)

Redaction of on-disk values is gated on `.homonto/state.json`
(`claude.go:80-86`, `opencode.go:104-110`): `if inState && ...` else
`Old` = the raw on-disk value — a **resolved secret** if a previous
apply wrote one. State goes missing by design: it is gitignored (fresh
clone / second machine = no state) and saved only after *all* adapters
succeed (`engine.go:97-107` — any partial apply = no state). Reproduced:
apply secret-backed MCP → drop state → edit key to literal →
`homonto plan` prints the secret in cleartext. Directly falsifies the
README guarantee "`plan` never resolves or prints a secret."
Compounding: `writeAtomic` always creates files 0644, loosening
pre-existing 0600 modes on the files that hold resolved secrets
(`adapter/*/util.go`) — reproduced.

### 4. CRITICAL — No pruning; "declarative" is false advertising

There is no `delete` action anywhere (`adapter.Change.Action` ∈
create/update/noop). Removing `[mcps.brave]` from homonto.toml leaves
`mcpServers.brave` in `~/.claude.json` forever — **with its resolved
plaintext API key** — invisible to plan/status/drift. `state.json`
entries are never garbage-collected. Deletion-drift (user runs
`claude mcp remove`) derives `create` → filtered → `status` prints
"No drift." Renamed skills leave dangling symlinks forever. This is an
upsert engine; terraform without destroy is not terraform.

### 5. FALSE — the install instruction

`go install github.com/noviopenworks/homonto@latest` cannot work for
anyone: the repo is **private**, `main` was never pushed (remote holds
one stale month-old feature branch), no tags, no releases, no CI, **no
LICENSE** (all-rights-reserved by default — nobody may legally use it),
and `Version` is a `const` (`cli/root.go:6`) that `-ldflags -X` cannot
stamp.

---

## Go audit — remaining findings (all reproduced unless noted)

- **CRITICAL — sjson path injection.** Names with dots
  (`[mcps."corp.internal"]`, plugin IDs `name@marketplace…` with dots)
  are spliced unescaped into gjson/sjson paths (`claude.go:158-162`,
  `jsonutil.go`). Reproduced: bogus nested `{"corp":{"internal":…}}`
  written into live config; plugin write **silently no-ops** while
  "Applied." prints; plan/apply read/write different keys → perpetual
  non-convergent `+ create` every run.
- **HIGH — skill-name path traversal.** `own = ["../../../escaped"]`
  escapes `$HOME` via `filepath.Join` and created a self-referencing
  symlink (ELOOP). No single-path-element validation; `link.Link` never
  guards src==dst.
- **HIGH — multi-adapter apply is not transactional.** Reproduced:
  opencode write fails → claude already wrote (with resolved secret) →
  error return → `State.Save` never runs. System half-applied, zero
  record; next plan hits finding 3. Also: secrets are resolved **twice**
  per apply (pre-resolve + adapter), doubling pass/GPG prompts.
- **MEDIUM** — plan output ordering nondeterministic between runs (map
  iteration; reproduced 3 orderings) — defeats eyeball-diffing the
  confirm prompt. `readStandardized` mishandles valid-but-non-object
  JSON roots. Drift conflates "user edited homonto.toml" with "disk
  drifted" ("will reset on apply" wording is backwards for the former).
- **LOW** — duplicated `util.go` between adapters (every bug above ×2);
  `mustJSON` swallows marshal errors; unsalted `Hash` lets a state.json
  reader correlate identical secrets across keys; raw Go errors as CLI
  UX; adapter-skipped warnings leave exit code 0 (CI can't detect).
- **Test gaps**: no tests for dotted keys, mode preservation, traversal,
  partial-failure state, ordering stability, non-object roots,
  missing-state secret leak (the one secrecy test only covers the
  state-present branch). `resolvejson_test.go` is genuinely strong.

## Product/architecture — remaining findings

- **Drift semantics dishonest**: three invisible classes — managed-key
  deletion, wholesale rewrite by the tool (Claude Code rewrites
  `.claude.json` constantly; no locking/mtime check → clobber race, no
  ADR covers choosing a runtime state file as a managed surface), and
  the multi-machine story (state machine-local; "safe to share" is moot;
  no per-machine profiles). Plan→confirm→apply TOCTOU: apply executes
  the stale confirmed set.
- **Plugin "management" flips `enabledPlugins` flags pointing at
  plugins that were never installed** (Claude side); identical TOML
  surface has different semantics per tool (OpenCode genuinely
  installs). Hooks/permissions/statusline/memory/agents/commands:
  unmanageable or vapor.
- **Value proposition vs chezmoi/jq script**: today homonto delivers the
  commodity parts (symlinks, JSON upsert, secret interpolation — all
  achievable with chezmoi natively) and botches the differentiating
  parts (authoritative tool schemas, pruning, tool-aware drift).
  Conditional reject at design review until the adapter layer is
  genuinely authoritative.
- **Roadmap grades**: v1.1 credible/low-value; v1.2 built on
  no-pruning sand; v1.3 trivial; v2 agent lifecycle is
  roadmap-as-fiction (three-way merges proposed by a tool that cannot
  delete a JSON key). Missing entirely, in the order users hit them:
  pruning, multi-machine/profiles, **project-scoped config**
  (`.mcp.json`, per-repo `.claude/settings.json`), more adapters
  (Cursor/Codex/Gemini CLI/Zed), import parity.
- **Dogfood ratio**: 8 onto skills + 184K of content/ + a 14K workflow
  spec vs 2-4K product specs (one Purpose literally "TBD") — the repo
  reads as "a markdown SDLC framework with a config tool attached."

## onto framework — "a prose cathedral with no load-bearing walls"

- **CRITICAL — no light path for small features.** Fix = bugs, tweak =
  non-behavior text; a `--version` flag is "new behavior" → full
  workflow: ~13-14 files (~9k tokens) of process read before one line of
  code, 9 artifacts, 5-6 gates, two mandatory skeptics (`verify.mode:
  full` is unconditional for `workflow: full`), a guides obligation, an
  ADR. Realistic cost: 150-300k tokens for `println(VERSION)`. Preset
  triggers (fix: 3+ files; tweak: any config-key add) are tripwires.
  The workflow trains its own abandonment.
- **CRITICAL — enforcement regressed from comet's scripts to wishes.**
  ADR 0005's own Consequences admit it. Everything skippable is a no-op
  nothing detects: metrics stamps, notes.md every-turn updates, dual
  checkoffs, template conformance re-reads, byte-identical table diffs.
  "Never trust conversation history" is an instruction an LLM executes
  exactly when it didn't need it. The gate-capped rebuild boundary table
  defends against a threat git already covers; dead weight.
- **HIGH — "self-contained" is false.** HALT-on-missing-rtk (a token
  *discount*) is indefensible; graphify is an external product with a
  URL in the halt text. The self-containment grep
  (`openspec|comet|docs/superpowers`) was constructed to be unable to
  find the two hard dependencies that exist. ADR 0006 rejected an
  in-repo lint subcommand as a "binary dependency" — in a Go binary
  project. The one place a script would convert 60 lines of lint prose
  into enforcement was ruled out on principle.
- **HIGH — redundancy is a drift factory with receipts.** Rules live in
  up to 5 copies; ~34 of the 30+ findings across three verify rounds
  were the copies disagreeing; and a live contradiction survived all
  three rounds: **ADR 0007 still says skipped adversarial passes are
  "recorded deviations"; the references say the opposite.** The
  adversarial-verify economics are self-dealing: complexity manufactures
  the defect supply the expensive skeptics then consume (~450k tokens
  for a docs-only change, 3 rounds).
- **The flagship verification report violates its own evidence rule** —
  rows cite prose ("dispatcher §3.1 rule", "R3 skeptic item 1/12")
  instead of executed commands. If the authors do pointer-evidence
  mid-adversarial-review, every future agent will.
- **Metrics axis is dead on arrival**: dates, not durations — five
  identical values for any same-day change; read by nothing.
- Gate pre-authorization semantics are too subtle to survive
  compaction (the archived `decisions.directive` is already a verbatim
  quote *plus* an interpretive annotation, because verbatim alone is
  unusable).
- **Genuinely good (keep these)**: file-state phase derivation with
  gates-win-upward; fresh-context adversarial review as a mechanism (it
  found ~30 real defects in-session review missed); reference-file
  progressive disclosure (the one place single-source-of-truth was
  achieved).
- **Minimal better design**: keep derivation + delta-spec merge
  semantics + notes.md + ONE optional skeptic; move lint/derivation
  into a ~150-line `homonto lint` subcommand (the checklists are
  literally already specs for it); add a feature-lite path (small
  features ≤ ~5 files, no new capability); demote rtk/graphify to
  warnings; delete metrics/deps machinery until needed.

## Docs-vs-reality — the lies, worst first

| Claim | Verdict | Reality |
|---|---|---|
| `go install …@latest` | FALSE | private repo, main unpushed, no tags, no LICENSE |
| "inline comments inside rewritten regions may not survive" | MISLEADING | **every comment in the whole file is destroyed on any write** (hujson Standardize strips document-wide; verified; the test seeds `// keep me` and never asserts on it) |
| owned "skills/commands/rules/agents" | MISLEADING | only skills exist (config has `Skills.Own` only) |
| import "SHALL read the current Claude/OpenCode setup" | FALSE | reads `~/.claude.json` mcpServers only; targets hardcoded `["claude"]`; redaction misses `*_SECRET`/`*_PASSWORD`/`DATABASE_URL`/`glpat-` |
| archive contract "moved verbatim… never edited" | FALSE in effect | onto-close moves ADR drafts out first → every archive ships dangling `adr/*.md` references in its own design.md; the lint exempts archives from the dangling check |
| ADR 0004 "shared atomic-write helper" | MISLEADING | three copy-pasted implementations; no fsync before rename |
| `--config` works "for any command" | MISLEADING | `init` ignores it entirely |
| guide "no workflow CLIs" + "halts, no degraded mode" | SELF-CONTRADICTORY | hard-requires two external CLIs; the skill itself has an explicit degraded fallback the guide denies |

Verified TRUE and undersold: plan is a pure dry run; two-phase
confirm-gated apply with state-last; secret→literal transition
redaction; token+hash-only state; fault isolation with warnings;
relative-content-dir absolutization; init never overwrites.

Hygiene: LICENSE missing, CI missing, `.gitignore` fine, go.mod matches
remote, symlink drift invisible to `status` (skill links never recorded
in state).

---

## Priority fix order

1. Claude MCP schema (`type`/`command`/`args`) + import args-drop — the
   product is broken; import is a trap
2. Secret leak on missing state + 0644 mode loosening — security
3. Pruning/deletion — or delete the word "declarative"
4. sjson path escaping + skill-name validation — corruption/traversal
5. Push main, LICENSE, tag, CI, `var Version` — existence
6. Real-schema conformance suite — fixtures from actual `claude mcp add`
   / OpenCode output; the missing test class that let P0 through three
   "all pass" verifications
7. onto: light path for small features; rtk/graphify halts → warnings;
   collapse rule copies (consider `homonto lint`); fix ADR 0007
   contradiction; ship archives without dangling refs
