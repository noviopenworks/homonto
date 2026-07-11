# Design proposal — `[agents]` / `[subagents]` reconciliation (roadmap item 9)

**Status:** Proposed (for review) · **Date:** 2026-07-11 · **Author:** implementation agent

This is a **design proposal for your approval**, not implemented work. It maps
the real overlap between Homonto's two agent-declaration models, lays out three
reconciliation options with trade-offs, and recommends one. The migration is
potentially config-breaking, so nothing ships until you pick a direction.

## 1. The two models today (grounded in source)

Homonto has **two independent ways to put an agent definition into a tool's
agent directory** (`~/.claude/agents/`, `~/.config/opencode/agent/`, or their
project-scoped equivalents):

| | `[subagents.<name>]` | `[agents.<name>]` |
|---|---|---|
| Type | `config.Resource` (`Source`, `Scope`, `Targets`) | `config.Agent` (`Source`, `Version`, `Targets`, `Mode`) |
| Managed by | the **declarative** engine: `plan` / `apply` / `status` | the **imperative** `homonto agents …` command group |
| State | none (recomputed from config each apply) | `.homonto/agents-lock.json` + content-addressed base blobs |
| Projection | materialize from catalog + **symlink** | **copy or link** into the tool dir |
| Scope | `user` **or** `project` (per resource) | **`user` only** (hardcoded `subagentpath.Dir(tool, "user", …)`) |
| Framework expansion | yes (`[frameworks.X]` expands into subagents) | no |
| Versioning | no | yes (`version`) |
| Local-edit survival | none (symlink → source is the truth) | **three-way merge** of local edits with a source/catalog upgrade (base-blob store, `.merged` sidecar) |
| Cleanup | prune-on-undeclare via `apply` | `agents prune` / `agents gc` |

Both are real, shipped, and separately spec'd (`subagent-projection`,
`agent-lifecycle`).

## 2. The concrete problems

1. **Silent collision.** There is **no cross-validation** between the two maps.
   `[subagents.foo]` (scope `user`) and `[agents.foo]` both resolve to
   `~/.claude/agents/foo.md`. Nothing at load rejects this; at runtime `apply`
   symlinks the path while `agents add` copies over it (or reports a foreign-file
   conflict) — undefined, order-dependent behavior over one file.
2. **Scope asymmetry.** `[subagents]` is scope-aware; `[agents]` is user-only.
   A user who wants a *project-scoped* lifecycle-managed agent cannot express it.
3. **Two mental models with no stated boundary.** Nothing tells a user when to
   use declarative projection vs the imperative lifecycle, so the choice is
   arbitrary and the overlap invites the collision above.
4. **Conflict resolution is not recoverable.** A three-way-merge conflict writes
   `<dst>.merged` and stops, but there is no command to *accept* the resolved
   file as the new base — the user must hand-edit the lockfile or re-add.

## 3. Goals / non-goals

**Goals:** one coherent story for "an agent file managed by Homonto"; no
silent collision; scope parity; an explicit, recoverable conflict resolution;
a migration path that never silently drops or corrupts a user's agent.

**Non-goals (this increment):** remote agent sources (v1 non-goal); changing the
merge algorithm; blob GC (already shipped, `3529ce7`).

## 4. Options

### Option A — Full unification: one `[agents]` model absorbs `[subagents]`

Make `[agents]` the single agent model: add `scope`, make it framework-
expandable, and route everything through one management path. Deprecate
`[subagents]` with an auto-migration (`[subagents.X] source/scope/targets` →
`[agents.X]` with `mode=link`).

- **Sub-choice A-decl:** the unified model stays **declarative** — `apply` gains
  the lifecycle (blob store, three-way merge at apply time, lockfile). The
  `agents` command group becomes thin wrappers over apply.
- **Sub-choice A-imp:** the unified model becomes **imperative** — framework
  agents expand into lifecycle installs done outside `apply`.

**Pro:** cleanest end state — one model, one directory owner, scope everywhere.
**Con:** large, breaking rewrite; A-decl means teaching `apply` merge/lockfile
semantics (a major change to the deterministic pipeline); A-imp breaks the
declarative framework model. High risk right before a first release.

### Option B — Bounded reconciliation: two models, hard boundary (recommended)

Keep both models but make them **provably non-overlapping** and reach parity:

1. **Scope parity** — add `scope` (`user`|`project`) to `config.Agent`; thread
   it through `agents add/update/doctor/prune/gc` (replace the hardcoded
   `"user"`). `[agents]` and `[subagents]` then have the same scope semantics.
2. **Collision guard at load** — reject (a) any name declared in **both**
   `[agents]` and `[subagents]`, and (b) any `[agents]`/`[subagents]` pair that
   would resolve to the **same tool path** (same effective name × scope × target).
   Collision becomes impossible by construction, surfaced at load with the two
   offending declarations named.
3. **Stated boundary (docs + spec)** — `[subagents]` = *stateless declarative
   projection* for catalog/framework agents used as-is; `[agents]` = *stateful
   lifecycle* for agents you locally customize and want preserved across
   upgrades. One sentence each, in the guide + both specs.
4. **Explicit promotion** — `homonto agents adopt <name>` converts a declared
   subagent into a lifecycle `[agents]` entry (records the current content as the
   base) for when a user starts editing a previously-projected agent. No forced
   migration; existing configs keep working.
5. **Recoverable conflicts** — `homonto agents resolve <name>` accepts the
   hand-edited `<dst>.merged` as the new live file + base (closes problem 4).

**Pro:** fixes every stated defect; small, additive, reversible; preserves both
UXes and all shipped functionality; safe before a first release. **Con:** the
"coherence" is enforced by guards + documentation, not by a single model — full
unification (A) remains a later step.

### Option C — Collapse into `[subagents]` + apply

Give `[subagents]` optional `version` + opt-in merge so `apply` does three-way
merge for mutable subagents; delete the `[agents]` command group.

**Pro:** unifies under the declarative model. **Con:** discards the just-shipped
v2 lifecycle (add/update/doctor/prune/gc/merge/adopt) — a large functional
regression. Not recommended.

## 5. Recommendation vs decision

The proposal *recommended* Option B. **The maintainer chose Option C** (collapse
into `[subagents]` + `apply`), with **auto-supersede** migration and a **project**
default scope. Section 8 details the Option-C plan under those choices. The
implementation recommendation stands recorded (B was lower-risk), but C is the
approved direction; the key trade-off C accepts is a breaking removal of the
`homonto agents` command surface in exchange for a single declarative model.

## 6. Approved decisions (2026-07-11)

- **Direction: C** — one declarative `[subagents]` + `apply` model; the imperative
  `[agents]` group is removed and its value (versioning, three-way merge of local
  edits, base blobs) is folded into `apply`.
- **Migration: auto-supersede** — an existing `[agents.<name>]` is read as an
  equivalent `[subagents.<name>]` (copy-mode) without any user action; no
  `homonto agents adopt` command.
- **Default scope: project** — a subagent with an omitted `scope` installs into
  the project (`<repo>/.claude`, `.opencode`), not `user`.

## 7. Key trade-off C accepts

C **removes the shipped `homonto agents` command surface** (`list`/`add`/`update
[--all]`/`doctor`/`prune`/`gc`) and the `[agents]` config table — an
outward-facing, breaking change. The *capabilities* are preserved inside `apply`
(copy-mode subagents get versioning + three-way merge + base blobs + conflict
sidecars); only the imperative UX and the second declaration table go away. This
is a deliberate re-architecture and must land as a sequence of reviewed comet
changes, not a single edit.

## 8. Option-C implementation plan (incremental, apply-preserving)

Each step is independently shippable and green; the `[agents]` surface is removed
only in the last step, after `[subagents]` reaches full parity.

> **Landed so far (2026-07-11):** step 1a — omitted `[subagents]` scope defaults
> to `project` (`7fba2dc`); and a safety guard rejecting a name declared in both
> `[agents]` and `[subagents]` (`4f28565`, closes design problem #1 for the
> transition). Blob GC (`3529ce7`) is also done. **The coupled core below —
> dedicated `Subagent` type + copy-mode projection + apply-time three-way merge —
> is the remaining work; it delivers value only when steps 1b–3 land together and
> re-plumbs the deterministic `apply` path, so it should be built as a focused
> comet change with per-step TDD, not piecemeal.**

1. **Subagent model gains `mode` + `version`.** Add `Mode` (`copy`|`link`, default
   `link` = today's symlink) and `Version` to the subagent model, validated.
   Purely additive; symlink projection unchanged. Default subagent `scope` becomes
   `project` (additive — subagent scope is *required* today, `config.go:604`, so
   defaulting an omitted value relocates no existing install).
   **Constraint discovered:** `config.Resource` is shared by frameworks, skills,
   commands, AND subagents (`Config.{Frameworks,Skills,Commands,Subagents}` are all
   `map[string]Resource`), and skills/commands also *require* scope. So `Mode`,
   `Version`, and the project scope-default **must not** be added to the shared
   `Resource` blanket — either give subagents their own struct (e.g. `Subagent`
   with the extra fields) or gate the new validation on the subagent kind only.
   The dedicated-struct route is cleaner and mirrors how `Agent` is already
   separate; it is the recommended first move of the re-architecture.
2. **Copy-mode subagent projection in `apply`.** A `mode=copy` subagent is
   materialized as a real file (not a symlink); `apply` records its base hash in
   `state.json` and stores the base content in the existing `agentblob` store.
   Idempotent; a copy-mode subagent with no local edits re-applies to a no-op.
3. **Three-way merge at apply for copy-mode subagents.** When the on-disk copy was
   locally edited (differs from the recorded base) and the source changed, `apply`
   runs `merge.Merge(base, local, source)`: 0 conflicts → write merged, advance the
   base, back up the prior local; conflict → write `<dst>.merged`, leave the live
   file, and report the resource as conflicted in the plan/status (never a silent
   overwrite). Reuses `internal/merge` + `internal/agentblob` verbatim.
4. **`status`/`doctor` surface copy-subagent state.** A `<dst>.merged` sidecar is
   a reported conflict; blob GC (already shipped as `agents gc`) is re-homed as an
   `apply`-time or `homonto gc` reclaim over `state.json`-referenced base hashes.
5. **Auto-supersede `[agents]` → `[subagents]`.** At load, translate every
   `[agents.<name>]` into an equivalent copy-mode `[subagents.<name>]` (carrying
   `version`/`targets`/`scope`), with a one-release deprecation warning. A name in
   both tables resolves to the subagent (the agent table is legacy).
6. **Remove the `[agents]` surface.** Delete the `homonto agents` command group
   (`internal/cli/agents*.go`), the `config.Agent` table, and `internal/agentlock`
   (its role is now `state.json`); keep `internal/agentblob` + `internal/merge`.
   Update specs: fold `agent-lifecycle` into `subagent-projection`; roadmap item 9
   → done.

**Risk to manage:** step 3 adds non-deterministic-looking behavior (merge output
depends on on-disk local edits) to the previously pure `plan`. Mitigation: `plan`
reports a copy-subagent as `~ merge` / `! conflict` without performing the merge;
only `apply` writes. Characterization tests from the `[agents]` suite port over to
lock the merge behavior in its new home.
