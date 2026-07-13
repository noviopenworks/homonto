# Homonto & Onto — Product Roadmap

**Horizon:** the path from "strong projector, weak workflow" to a
state-of-the-art spec-driven toolkit.
**Grounded in:** the 2026-07-13 harsh review (`FINDINGS.MD`), re-verified
against the current tree on 2026-07-13. Each item below cites the findings it
closes.
**Relationship to `docs/roadmap.md`:** that file is the release-status ledger
(what shipped, with evidence). This file is forward strategy. Finding
22 (F22) — "the roadmap is a completion diary, not strategy" — is why the two
are now separate. History lives in `openspec/changes/archive/`; strategy lives
here.

## Where we are (honest)

The projection engine is genuinely good: plan/confirm/apply, secrets referenced
never resolved, drift separated from desired-state change, state persisted after
each adapter, remote pinning with fail-closed extraction. The workflow layer
around it is not. The review scored the core engine 8/10 and onto workflow
coherence 2/10, and re-verification backs that split.

The single structural problem, from which most others follow:

> **Onto is two incompatible control planes wearing one name.** The markdown
> skill framework (`catalog/skills/onto*`) stores rich state in `state.yaml`.
> The Go binary (`cmd/onto`, `internal/ontocli`, `internal/ontostate`) stores
> seven fields in `onto-state.yaml`. Neither reads the other's file. They report
> different phase, workflow, and archive state, and neither reconciles.
> (Verified: `internal/ontostate/state.go:126`, `catalog/skills/onto-open/SKILL.md:63`.)

Recent work hardened the **skill** plane (commit discipline, close idempotency,
delta-coverage lint, isolation timing, no-slop safety) and shipped the
remote-trust subsystem, the adapter contract + Codex pilot, and a documentation
consolidation. The **binary** plane was not touched and remains structural-only:
it gates on filenames and checkboxes, not on confirmed design, scenario
coverage, a passing verification, merged specs, or resolved ADRs. A user can
create empty files, check boxes, and archive unverified work
(`internal/ontocli/advance.go:77`, `internal/ontocli/close.go:58`).

So the roadmap's first job is not new features. It is making onto **true and
safe**, then building the spec-driven core the name already claims.

## The strategic fork — RESOLVED (2026-07-13)

Everything in "Now" depended on one decision. It is now made. The decision chain
below is locked; the "Now" items are rewritten to match it.

1. **Which control plane is authoritative? → Binary.** Enforcement is the point
   of onto, not portability. One Go engine owns one versioned state schema and
   deterministic phase transitions; skills *invoke* the binary, they do not edit
   state files. This **ends the markdown-only / "no external CLI" promise** — that
   copy in `onto*/SKILL.md` is now false and must be deleted, and onto now carries
   a **hard dependency on the compiled binary** (no graceful markdown fallback).

2. **What does the binary enforce? → B1: ceremony, not judgment.** The binary can
   cheaply verify *structural* facts (a named artifact exists, checkboxes are
   ticked, a well-formed `verification-result: pass` token is present). It cannot
   read a design doc and decide it is sound. So the agent performs the semantic
   judgment and emits a machine-checkable token; the binary enforces that the
   **token exists and is well-formed** and trusts the judgment behind it. onto's
   guarantee is therefore *"an honest agent cannot skip a step,"* **not** *"no
   agent can ship bad work."* We own the weaker claim on purpose — B2 (the binary
   re-deriving evidence: running tests, mechanically checking scenario coverage)
   is out of scope, much of it isn't mechanically decidable, and it is not funded.

3. **Threat model → T-honest for onto, T-hostile for the engine.** B1 defends
   against a *forgetful/sloppy* agent, not a *malicious* one — nobody's threat
   model is "an attacker forges a spec-driven audit trail on my own repo," and if
   they hand-write the pass token B1 is worthless anyway. So onto's gates
   explicitly assume an honest agent. The **projection engine is different**: it
   consumes remote content and deletes files, so it faces a real adversary
   (tampered state, hostile remotes). This split is load-bearing below: it moves
   the security findings (F7/F25/F29 and remote transactionality) **out of onto's
   blockers and into a separate engine-safety gate** — still P0 *for the engine*,
   but no longer coupled to onto's truth problem.

4. **Product hierarchy (F21) → homonto is the product; onto is its native,
   binary-enforced workflow.** Because onto now requires the homonto toolchain, it
   is no longer a standalone framework sitting beside Comet/OpenSpec/Superpowers —
   those remain *unenforced alternative* workflows you can drive projection with;
   only onto is enforced by the binary.

5. **Dogfooding → we build with Comet, we ship onto.** This repo continues to
   develop with Comet; onto is a shipped-but-not-self-used product. This is an
   honest choice with a **standing tax**: onto never gets the dogfooding feedback
   loop that made the projector 8/10, which is exactly why onto scored 2/10. Two
   obligations replace that loop and are now *non-optional* (see N7): a
   full-lifecycle onto E2E/conformance suite, and an F21 persona/selection doc.

Consequence for sequencing: onto's truth problem (N1/N2/N3) and the engine's
safety problem (N4/N5/N6) are now **two independent release gates**, not one
monolith.

---

## Now — onto Truth (P0, gate A)

Blockers for onto. Independent of the engine-safety gate below. Nothing in
Next/Later is worth starting while these stand.

### N1. Unify onto's control plane onto the binary — ✅ DONE (2026-07-13)
*Shipped as two changes, both archived on `main`: `onto-binary-authoritative-state`
(binary owns one versioned schema, on-read migration, `onto set …`/`onto state
--json`, status/doctor enumerate+classify per F14) and `onto-skills-shell-out`
(the 8 `onto*` skills invoke the binary — grep-gate enforced — markdown-only copy
deleted; binary gained `onto new --workflow`, `onto set base-ref/deps/guides`).
Closes F1/F3/F14. Residual `abandon`/workflow-upgrade transitions deferred to N2.*

- **Problem:** two divergent state models, no reconciliation.
- **Decision applied:** binary-authoritative (fork decision 1). Skills stop
  writing state and **shell out** to `onto open|advance|close`; the exit gate is
  not "both planes read one file" but "skills contain **zero direct state
  writes**."
- **Closes:** F1, F3, F14 (state deleted or skill-only → invisible to binary
  status/doctor), and unblocks F2/F4/F9/F10.
- **Exit gate:** one authoritative state schema with a version field; every skill
  transition is a binary invocation, no skill writes `state.yaml`/`onto-state.yaml`
  directly; the "markdown-only / no external CLI" copy is deleted from the skills;
  `status`/`doctor` enumerate change directories first and classify (valid /
  malformed / missing-state), so a workspace never silently disappears.

### N2. Make the binary's gates semantic — B1-scoped — ✅ CORE DONE (2026-07-13)
*Archived on `main` (`onto-semantic-gates`): `onto close` now gates on close-phase
evidence workflow-aware (full: verify.result==pass + close.merged + guides resolved;
fix/tweak: verify.result==pass + close.merged); `onto advance` gates leaving-verify
on verify.result==pass and entering-build on isolation set. 135 tests -race.
**Deferred N2 follow-ups** (separable, recorded): comet verification-scale-by-risk +
non-waivable finding classes + skeptic/reviewer subagents (F11/F12); skill-plane
recovery/atomic bookkeeping (F16/F17); full dep resolver with cycle detection (F10).*

- **Problem:** advance/close check existence + checkboxes, not validity; empty
  unverified work can archive.
- **Decision applied:** B1 (fork decision 2). The binary enforces that a
  well-formed, agent-emitted evidence **token** exists and is structurally valid;
  it does **not** re-derive the judgment (no running tests, no mechanical scenario
  proof). Guarantee = "an honest agent cannot skip a step." T-honest (decision 3):
  no defense against a forged token is in scope here.
- **Closes:** F2, F9 (workflow-aware transitions + `onto new --workflow
  full|fix|tweak`), F10 (one shared dep resolver that blocks entering build,
  date-anchored exact match, cycle detection).
- **Exit gate:** a phase advances only when the phase's evidence token is present
  and well-formed — a confirmed-design marker, a scenario-coverage token, a
  `Result: pass` record, a merged-deltas marker, resolved ADRs/guides — with
  fix/tweak presets gated on their own reduced token set, not the full one. The
  binary trusts the token's *contents*; it enforces its *presence and shape*.
- **Workflow safety (also here):** isolation (branch/worktree) is chosen
  **before the first workspace commit**, so open/design work never lands on
  `main` (F15 — verified: skills commit at open/design exit but the branch is
  created only at build); build recovery distinguishes user, concurrent-agent,
  and failed-task work and preserves dirty paths into a patch/WIP branch before
  any destructive reset (F16); parallel task code and its checkoff land in one
  coordinator-owned commit so a crash never redispatches completed work (F17).



### N3. Fix stale canonical specs (validation must mean truth) — ✅ DONE (2026-07-13)
*Archived on `main` (`fix-stale-canonical-specs`): agent-lifecycle retired to a
tombstone, cli-commands/config-model corrected, and `scripts/spec-command-check.sh`
(a spec↔code correspondence gate) wired into `scripts/gate.sh`. Closes F5 + F20
residue. `docs/superpowers/*` historical residue left to F19.*

- **Problem:** `openspec/specs/agent-lifecycle` and `cli-commands` still mandate
  the removed `homonto agents` command group; `openspec validate` reports 15/15
  because it checks form, not correspondence to reality
  (verified: `openspec/specs/agent-lifecycle/spec.md:6`, no `agents` command in
  `internal/cli`).
- **Closes:** F5, and the F20 residue (public claims still implying agents/limits).
- **Exit gate:** every living spec matches shipped behavior; CI carries a
  spec↔code correspondence check (even a coarse "spec names a command the CLI
  doesn't register" grep) so form-valid-but-false specs fail.
- **Note:** this is the one-day fix that is a hard blocker for the RC tag (see
  Open decisions §2), independent of the rest of gate A.

### N7. Substitutes for dogfooding (because we ship onto but build with Comet) — ✅ DONE (2026-07-13)
*Archived on `main` (`onto-dogfooding-substitutes`): `docs/personas.md` (F21 —
homonto=product / onto=native binary-enforced workflow / Comet et al.=unenforced
alternatives / build-with-Comet-ship-onto, linked from README) + a full-lifecycle
onto conformance suite (`internal/ontocli/conformance_test.go`, 6 tests) asserting
the gates reject bad work (missing artifact, invalid workflow, out-of-shape
enum/guides, malformed/missing state). No onto gate weakness found.*

- **Problem:** fork decision 5 means onto never gets the feedback loop that made
  the projector 8/10 — the two-plane drift, fake gates, and archive rot the review
  found are exactly what daily use would have surfaced. Nobody on the team lives
  in onto, so its correctness cannot come from use.
- **Closes:** F21 (product hierarchy has no persona/selection matrix), and it is
  the safety net for N1/N2 landing without a human catching regressions in use.
- **Exit gate (both, non-optional):**
  1. a **full-lifecycle onto E2E/conformance suite** that drives
     `open → design → build → verify → archive` end-to-end and asserts the B1
     gates actually *reject* bad work (empty artifacts, missing/malformed evidence
     token, unmerged deltas) — this replaces the feedback loop we declined;
  2. an **F21 persona/selection doc**: "homonto is the product; onto is its native
     binary-enforced workflow; we build with Comet and ship onto; here is who onto
     is for and why we don't use it ourselves." Comet/OpenSpec/Superpowers are
     documented as unenforced alternatives.

## Now — Engine Safety (P0, gate B)

Separated from gate A by the threat-model decision: the projection engine faces a
real adversary (T-hostile) — it consumes remote content and deletes files — so
these stay P0 **for the engine**, but they are **not onto blockers** and gate the
RC scope rather than onto's truth. They can proceed in parallel with gate A.

### N4. Close the arbitrary-deletion and traversal holes — ✅ DONE (2026-07-13)
*Archived on `main` (`close-deletion-traversal-holes`): F28 — `validateResources`
now applies the `local:` plain-name check via a shared helper (skills/commands
can't drift from subagents); F7 — copy-mode prune confined at the `copyfile.Apply`
choke point to the managed provider roots (fail-closed, `..`-safe), so a tampered
state path can't delete an arbitrary file. 195 tests -race.*

- **Problem:** copy-mode prune takes its destination from unvalidated
  `Entry.Desired` in state; a tampered state.json with a matching hash deletes
  an arbitrary file (verified: `internal/adapter/claude/claude.go:248`,
  `internal/copyfile/copyfile.go:145`). `local:` skill/command sources join
  their raw suffix into provider paths with no plain-name check, unlike
  subagents (verified: `internal/config/config.go:693` vs `:740`).
- **Closes:** F7, F28.
- **Exit gate:** prune destinations are reconstructed from validated resource
  identities and confined under a managed root; `local:` skills/commands get the
  same plain-name validation subagents already have; a cleaned path escaping the
  provider root is a load error.

### N5. Make remote application transactional and drift-active — ✅ DONE (2026-07-13)
*Archived on `main` (`transactional-remote-apply`): F8 — `materializeRemotes`
restructured to quarantine → stage-verify-ALL → activate (a mid-run failure leaves
active content + lock untouched); F6 — a digest-only repin shows in `plan` and needs
confirmation; F27 — git fetch under a deadline with size guards before checkout;
F30 — doctor verifies materialized digests vs lock, revoked content deactivated;
F26 — cache-race winner re-hashed. 583 tests -race. **Closes gate B.**

- **Problem:** `materializeRemotes` prunes de-declared content, then fetches and
  materializes each remote in a loop; a later failure leaves earlier content
  changed and the lock stale (verified: `internal/engine/remote.go:62`). Revoked
  but still-declared content stays linked after a failed apply. A digest-only
  repin is invisible in `plan` and applies without a confirm
  (`internal/cli/apply.go:47` — re-verifies but shows no diff). Git fetch runs on
  `context.Background()` with caps applied only after checkout
  (`internal/remote/fetch.go:126`).
- **Closes:** F8, F30, F6 (fully), F27, and the F26 cache re-verify gap.
- **Exit gate:** all remotes verify into staging before any active content or
  lock mutation; a digest change shows in `plan` and needs confirmation; revoked
  content is quarantined and doctor verifies materialized digests against the
  lock; git runs under a deadline with size guards, and a cache-race winner is
  re-hashed before acceptance.

### N6. Control-plane filesystem safety and locking — ✅ DONE (2026-07-13)
*Archived on `main` (`control-plane-fs-safety-locking`): F25 — `fsutil.WriteControlPlane`
no-follow writer for `.homonto` state/cache/lock/catalog (refuses a symlinked
destination); F29 — `internal/applylock` O_EXCL project lock so a second `apply`
fails fast; F31 — `remote.RedactLocator` strips userinfo/secret query tokens from
the lockfile and every URL-bearing error. 577 tests -race. (No-follow ships
destination-symlink refusal; full intermediate root-confinement noted as optional
future tightening.)*

- **Problem:** `WriteAtomic` follows symlinks (correct for tool configs, unsafe
  for `.homonto` state/cache/lock/catalog); no cross-process lock, so two
  applies last-writer-win and apply races GC (verified: no flock/O_EXCL in
  source).
- **Closes:** F25, F29, F31 (redact credentials from locator errors and lock
  entries).
- **Exit gate:** a no-follow, root-confined writer for control-plane paths; a
  project-scoped lock plus a generation/fingerprint check after confirmation;
  locators with embedded credentials are rejected or redacted before they reach
  logs or `remote.lock.json`.

---

## Next — The real spec-driven core (P1)

The features that would make "spec-driven" true rather than aspirational. Start
only after Now.

### X1. Stable IDs and a typed traceability graph
- **Problem:** requirements and scenarios are mutable headings; verification maps
  names to tests by hand (verified: `openspec/specs/*/spec.md` use
  `### Requirement: <name>`). The toolkit cannot answer "which code, tests,
  decisions, commits, and release prove scenario X."
- **Closes:** F13.
- **Exit gate:** immutable IDs for capabilities, requirements, scenarios,
  decisions, tasks, and evidence; typed edges (`implements`, `tests`,
  `supersedes`, `deviates-from`, `released-in`) validated in CI.

### X2. Immutable typed plans and transaction journals
- **Typed-operations slice DONE (2026-07-13, `typed-plan-operations` archived):**
  `adapter.Action` is now a defined type with constants + `Valid()`;
  `ChangeSet.Validate(knownTools)` and a fail-closed check at the top of
  `engine.Apply` reject an unknown tool (previously **silently skipped** by the
  `byName` lookup) or an undefined action (previously a silent no-op) before any
  secret resolution, materialization, or write. Low-churn/non-breaking (constants
  keep the historical string values). This closes the F41 "plan actions are
  unrestricted strings" gap.
- **Stateless-Apply slice DONE (2026-07-13, `stateless-adapter-apply` archived):**
  `Adapter.Apply` now takes `cfg *config.Config` and re-derives its
  skill/command/subagent entries via a shared `expand(cfg)` helper called at the
  top of both `Plan` and `Apply` — removing Apply's hidden precondition (it
  previously read `a.skills`/`a.commands`/`a.subagents` populated only by a prior
  `Plan` on the same instance, silently under-applying otherwise). Codex ignores
  cfg; `engine.Apply` passes `e.Cfg`. Behavior-preserving (same cfg → identical
  entries), ~135 test call sites updated. Closes the "`Apply` reads mutable
  adapter fields set by a prior `Plan`" X2 concern.
- **Remaining X2:** only transaction journals (F42) — plus, optionally, driving
  `Apply` purely from the `ChangeSet` (a larger rethink). F41/F47/F4/
  stateless-Apply are done.
- **F42 re-assessment (2026-07-13):** the journal's motivating concern —
  "adapter writes are sequential with no journal" → unsafe partial apply — is
  **already mitigated by construction**. `engine.Apply` saves state after each
  adapter; every managed write is atomic per key (`WriteControlPlane`/
  `WriteAtomic`/`link`, and skill dirs now stage-swap); and the **adopt path
  self-heals** a file written-but-not-yet-recorded (a crash between the write and
  the state save is re-adopted on the next run, since disk-matches-desired-but-
  absent-from-state → adopt). So apply is already crash-resilient/resumable
  without data loss. A journal (F42) would add explicit crash *detection* and an
  audit trail — not correctness — making it a lower-value, design-latitude
  greenfield item best scoped with a maintainer decision on desired semantics
  (roll-forward/resume already works; rollback of applied changes is likely
  undesirable). De-prioritized accordingly.
- **Catalog-materialization slice DONE (2026-07-13,
  `crash-safe-catalog-materialize` archived):** `catalog.Materialize` now
  stage-then-swaps each builtin skill dir (`<skill>.staging` → `RemoveAll` old +
  `Rename`), so a crash/error mid-walk leaves the prior complete dir or none —
  never a partial dir that `allSkillDirsExist` (Stat-only) would mistake for
  complete and never repair. Closes the skill-dir half of F47 (commands/subagents
  already write atomically).
- **Close archive-ordering slice DONE (2026-07-13, `close-archive-rollback`
  archived):** `onto close` set `archived: true` and saved before the archive
  move; a `MkdirAll`/`Rename` failure left the change marked-archived-but-not-moved,
  contradicting the spec's "archives NOTHING on failure". Now a failed move rolls
  the flag back to `false`, so a failed close leaves the change fully un-archived.
  Closes the deterministic error-path half of F4 (a process kill between the save
  and the rename still has a window — full crash-safety needs location-derived
  archived state, a larger redesign).
- **Problem (remaining):** `Apply` reads mutable adapter fields set by a prior
  `Plan`, not the plan alone; adapter and close writes are sequential with no
  journal (F42). Catalog materialization and the close error-path are now
  crash/consistency-safe; the deeper immutability + journaling (F42) and full
  crash-safety remain.
- **Closes:** F41, F42, F4, F47, F18 (archive must move all historical
  artifacts, rewrite references, and validate every referenced path and hash
  before marking the change archived — the item-10/11 archives already do this
  by hand; make it the enforced path).
- **Exit gate:** typed immutable operations rejected on unknown action/tool;
  per-operation journals for apply, close, and catalog materialization; a
  versioned staging tree swapped atomically; every close/archive step validated
  before the change is marked done, with no dangling reference or stale hash in
  the archived record.

### X3. Workflow profiles and a capability registry
- **F37 DONE (2026-07-13):** both planes are now versioned + fail-closed on a
  future version — `state-schema-version` (state.json) and `config-schema-version`
  (homonto.toml, archived).
- **F33 adapter-registry DONE (2026-07-13, `adapter-registry` archived):**
  `internal/adapter/registry` (Deps/Factory/Registry + `Builtins()`) replaces the
  hardcoded adapter slice in `engine.Build` — the engine no longer imports the
  concrete adapters, and adding a built-in is one `Register` line in `Builtins()`.
  Behavior-identical (same three adapters, same order, same options). Remaining X3
  (F34 interface-type generalization — the `Adapter` contract still binds concrete
  `config.Config`/`secret.Resolver`/`state.State`; config-loading phase split F43;
  non-waivable finding classes F11/F12 in the onto/comet workflow) is larger and
  design-first.
- **Problem:** verification scale keys on task/file counts, not risk or changed
  requirements, so a one-file security change can get less scrutiny than a large
  refactor (F11); escape hatches are too broad and the skeptic/reviewer subagents
  the ideal process needs are not bundled (F12); adapters are hardcoded and the
  contract binds concrete config/global/secret types (F33, F34).
- **Closes:** F11, F12, F33, F34, F37.
- **Exit gate:** any changed delta spec forces requirement-level evidence;
  non-waivable finding classes for security/data-loss/failed-core-acceptance;
  the reviewer/skeptic subagents ship in the catalog; a `ToolID`-keyed capability
  registry so a new adapter is a registration, not a repo-wide edit; explicit
  config and state schema versions with ordered idempotent migrations. Config
  loading splits into explicit phases (decode → migrate → normalize → validate →
  expand) with a generic per-kind expansion pipeline, ending the monolith (F43).

---

## Later — Ecosystem-grade (P2)

What turns an opinionated internal toolkit into something others build on.

### E1. A real framework ecosystem model
- **Started (2026-07-13):** F35 done (`reject-non-builtin-frameworks`) — a non-builtin
  `[frameworks.X]` source now fails loudly at load instead of silently installing
  nothing. Remaining E1 (F36 versioned manifests + dependencies/capabilities/
  compatibility ranges + local/custom framework resolution; F38 plugin lifecycle)
  is the large ecosystem model — design-first.
- **Closes:** F35 (a `local:` framework must fail at load, never silently install
  nothing — verified: `internal/config/config.go:256` skips non-builtin), F36
  (versioned manifests with dependencies, provided/required capabilities,
  compatibility ranges, overrides, migrations, conflict policy, local/custom
  resolution), F38 (a true plugin capability or an honest rename).
- **Exit gate:** a fourth framework or a local framework installs through the same
  validated, versioned path; unsupported source/kind combinations fail loudly.

### E2. Machine-readable CLI and a stable automation contract — DONE (2026-07-13)
- **DONE (2026-07-13):** F49, F45, F51, F48, F52, F46, F50 all archived on `main`.
  `--output json` COMPLETE (status, doctor, plan). The opt-in exit-code taxonomy
  shipped via `--exit-code` (`exit-code-taxonomy`, archived): `plan --exit-code`
  → 2 when changes/repins pending else 0; `status --exit-code` → 3 drift / 2
  pending / 0 clean; default behavior (no flag) unchanged so existing automation
  never breaks. Plumbed through a testable `cli.Execute(args) int` sink; only
  homonto's `main.go` changed, onto's is untouched. **E2 fully closed.**
- **Closes:** F45 (never print "up to date" after skipping an adapter), F46
  (catalog upgrades appear in the plan; doctor reports version mismatch), F49
  (`cobra.NoArgs` everywhere; a stray positional never silently ignored — verified
  missing on apply/plan/status/doctor/import), F50 (`--output text|json` and a
  detailed exit-code taxonomy), F51 (a `cache gc` command; bounded git; multi-
  remote progress), F52, F48 (import parse failure fatal before mutation; atomic,
  backed-up writes).
- **Exit gate:** a versioned JSON output and exit codes for drift / pending /
  degraded / warning / abort / doctor-findings that a CI pipeline can depend on.

### E3. Adapter conformance and the Claude/OpenCode consolidation
- **Started (2026-07-13):** `adapter-conformance-suite` archived — a shared
  `internal/adapter/conformance` suite; claude + opencode pass the core contract
  (Plan->create, Apply, ObserveHashes-clean, idempotent re-Plan, unmanaged
  preservation). F55 conformance COMPLETE for all three adapters (claude, opencode, codex) —
  core, drift-reset, malformed-doc, secret non-resolution, foreign-content (4
  slices archived). **F40 structured-doc slice DONE (2026-07-13,
  `consolidate-structured-doc-projection` archived):** claude + opencode now
  project their structured-document managed keys through the shared `structproj`
  core via one shared `internal/adapter/jsoncodec` (Codec over `jsonutil`),
  mirroring Codex — the duplicated diff/write/observe control-flow was removed
  (claude 1037→948, opencode 999→954). Behavior-preserving, pinned by the
  conformance suite; 645 tests green under -race. **F40 file-projection slice
  DONE (2026-07-13, `consolidate-file-projection` archived):** new
  `internal/adapter/fileproj` (symlink analogue of structproj) now owns both
  adapters' `skill.*`/`command.*`/`subagent.*` symlink projection via a
  type-agnostic `[]fileproj.Link`; six near-identical inline blocks removed, Apply
  fail-fast conflict ordering preserved verbatim, generic delete loop keeps pruning
  (fileproj plans no deletes). **F40 is now COMPLETE** across both slices: the two
  adapters dropped claude 1037→762, opencode 999→776, behind the shared
  `structproj`+`jsoncodec`+`fileproj` cores. Only follow-on left is copy-mode
  (`subagentcopy.*`) consolidation — a small optional change wrapping the already-
  shared `internal/copyfile` (opencode's array-based `plugin.*` stays adapter-owned
  by design). **E3 exit gate met** (conformance suite every adapter passes; the two
  big adapters reduced to shared cores + thin per-tool builders). **Copy-mode
  follow-on DONE (2026-07-13, `consolidate-copy-projection` archived):** new
  `internal/adapter/copyproj` (wrapping the shared `internal/copyfile`) now owns
  both adapters' `subagentcopy.*` reconciler; F7 prune-root guard + local-edit
  `.bak` backup preserved. **The adapter-consolidation story is now complete across
  all three surfaces** (structured-doc / file-projection / copy-mode): the two big
  adapters dropped **claude 1037→714, opencode 999→731**, behind the shared
  `structproj`+`jsoncodec`+`fileproj`+`copyproj` cores.
- **Closes:** F40 (both adapters are ~1000 lines duplicating security-sensitive
  planning/link/prune/copy/adopt/drift logic; the Codex `structproj` design is the
  better direction — migrate the two onto the contract), F55 (a reusable
  conformance suite: create/update/delete, adopt, drift, unmanaged preservation,
  secret redaction, malformed docs, conflict safety, byte stability), F39 (Codex
  coverage and docs generated from an executable capability matrix), F44 (typed
  observation results: clean/changed/degraded/unreadable/failed, not a warning
  side channel).
- **Exit gate:** one conformance suite every adapter passes; the two big adapters
  reduced to contract + per-tool codec.

### E4. Supply chain, CI, and release integrity
- **Partial (2026-07-13):** F58 (`perf-benchmarks` — BenchmarkLoad + BenchmarkMerge
  foundation) archived; F59 (build doc aligned to the pinned `go1.26.5` toolchain,
  `17c9c85`). Remaining E4 (CI real-tool E2E, native multi-OS CI, coverage/fuzz
  gates, signed provenance, comet-runtime provenance F23/F24) needs CI/external
  infrastructure not available in this environment.
- **Closes:** F32/F60 (pin action SHAs and tool versions; enforce SemVer,
  protected ancestry, approval, signed provenance, SBOMs, attestations), F53
  (CI runs the real-tool E2E matrix, no `|| true` swallowing exit status), F54
  (native macOS/Windows CI for symlink-sensitive behavior), F56/F57/F58 (coverage
  threshold, scheduled fuzz campaigns with preserved corpus, performance/allocation
  budgets), F24 (the ~13k-line generated Comet runtime gets authored sources, a
  pinned upstream, a generated-file header, and a regeneration-diff CI check),
  F23 (Comet's build guard recognizes Go so the repo stops needing a hidden
  `COMET_SKIP_BUILD`), F59 (align the documented Go version with the pinned
  `go1.26.5` toolchain), F61 (a thin archive format: canonical artifacts, hashes,
  and evidence, not full `.comet` runtime residue).
- **Exit gate:** a tag publishes only signed, provenance-attested, SemVer-valid
  artifacts, proven by native multi-platform real-tool E2E.

---

## Already delivered (ledger, not diary)

Pointers, so this roadmap doesn't re-list closed work. Full history is in
`openspec/changes/archive/` and `docs/roadmap.md`. Each carries the honest
caveat the review surfaced.

- **Remote-source trust** (`remote-source-trust`, ADR 0013): `remote:` sources,
  pinning, verify-before-mutate, cache/offline, revocation, lockfile.
  *Caveat:* N5/N6 above — not yet transactional across remotes, git ctx
  unbounded, revoked-but-declared content stays live, locators unredacted.
- **Adapter contract + Codex pilot** (`adapter-contract-codex-pilot`, ADR 0014):
  `structproj` + `tomlutil` + Codex MCP. *Caveat:* X3 — Claude/OpenCode not yet
  migrated onto it; the interface is not yet a real plugin boundary (F33).
- **Onto skill-plane hardening** (`4dabe8a`, `6520c14`): commit discipline, close
  idempotency + gate-before-mutation, delta-coverage lint, no-slop marker safety,
  preset resume maps. *Caveat:* the binary plane (N1/N2) is untouched.
- **Loose builtin skills/commands** (`0ef0485`): `handoff`, `grilling`.
- **Documentation consolidation**: single-source roadmap, README/guides aligned
  for Codex + remote. *Caveat:* F19 — `docs/superpowers/` still holds three
  retained historical designs while its README says "active only."

## Non-goals / explicitly deferred

- Automatic remote updates (a pin advances only by a manual config edit — a
  standing decision, not a gap).
- A hosted registry or signing PKI beyond content-digest pinning, until E4.
- Parallel new adapters: the contract and conformance suite (X3) come first, then
  one pilot at a time.

## Maintainer decisions — RESOLVED (2026-07-13)

1. **The strategic fork** — RESOLVED: binary-authoritative + B1 + T-honest-for-onto
   / T-hostile-for-engine. See "The strategic fork — RESOLVED" above.
2. **`v0.1.0-rc.1`** — HOLD conditions **now CLEARED (2026-07-13).** The RC was held
   on N3 (stale specs) + gate B (engine safety). All are done and archived on `main`:
   N3 (`fix-stale-canonical-specs`), N4 (`close-deletion-traversal-holes`), N6
   (`control-plane-fs-safety-locking`), N5 (`transactional-remote-apply`). The tag
   is now cuttable — remaining step is a **maintainer push + `v0.1.0-rc.1` tag** plus
   the post-tag smoke (`docs/roadmap.md` item 7); the agent cannot push/tag.
3. **Product hierarchy** (F21) — RESOLVED: homonto is the product; onto is its
   native, binary-enforced workflow; Comet/OpenSpec/Superpowers are unenforced
   alternatives. We build with Comet and ship onto. The persona/selection doc that
   makes this honest to users is tracked as N7.
