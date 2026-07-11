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

## 5. Recommendation

**Adopt Option B now; keep Option A (full unification) as the north star.**
B removes the real hazards (collision, scope gap, unrecoverable conflicts) with
a small, safe, additive change that discards nothing and stays reversible toward
A once the unified model is designed against a stable, released baseline. A is
the "right" long-term shape but is a breaking rewrite that should not gate the
first release; C throws away shipped value.

## 6. If B is approved — increment slicing

1. `scope` on `config.Agent` + validation + thread through the `agents` commands
   (default `user`, matching today). Tests: project-scope install/doctor/prune.
2. Load-time collision guard (name-in-both + same-resolved-path) with a focused
   rejection message. Tests: both collision kinds.
3. `agents adopt <name>` (subagent → lifecycle) + `agents resolve <name>`
   (accept `.merged`). Tests: adopt records the right base; resolve advances base
   and clears the sidecar.
4. Docs: the boundary paragraph in the agents guide + `agent-lifecycle` /
   `subagent-projection` spec notes; roadmap item 9 → done.

## 7. Open questions for you

- **Direction:** B (bounded, recommended) or A (full unification now)?
- **Promotion:** explicit `agents adopt` (B as written), or should declaring the
  same name in `[agents]` auto-supersede a `[subagents]` entry?
- **Default agent scope:** keep `user` (today's behavior) as the default when
  `scope` is omitted, or default to `project` for parity with how skills are
  commonly declared in this repo?
