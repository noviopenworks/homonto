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

### N1. Unify onto's control plane onto the binary
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

### N2. Make the binary's gates semantic — B1-scoped
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



### N3. Fix stale canonical specs (validation must mean truth)
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

### N7. Substitutes for dogfooding (because we ship onto but build with Comet)
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

### N4. Close the arbitrary-deletion and traversal holes
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

### N5. Make remote application transactional and drift-active
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

### N6. Control-plane filesystem safety and locking
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
- **Problem:** plan actions/payloads are unrestricted strings and `Apply` reads
  mutable adapter fields set by a prior `Plan`, not the plan alone; adapter and
  close writes are sequential with no journal (F41, F42). Catalog and close both
  mutate destructively before completion (F47, and the binary's F4 archive
  ordering).
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
- **Closes:** F35 (a `local:` framework must fail at load, never silently install
  nothing — verified: `internal/config/config.go:256` skips non-builtin), F36
  (versioned manifests with dependencies, provided/required capabilities,
  compatibility ranges, overrides, migrations, conflict policy, local/custom
  resolution), F38 (a true plugin capability or an honest rename).
- **Exit gate:** a fourth framework or a local framework installs through the same
  validated, versioned path; unsupported source/kind combinations fail loudly.

### E2. Machine-readable CLI and a stable automation contract
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
2. **`v0.1.0-rc.1`** — RESOLVED: **HOLD the tag.** The release-integrity gates are
   green (`docs/roadmap.md` item 7), but the RC ships the projector, which is
   T-hostile, and gate B (N4/N5/N6 — arbitrary deletion, non-transactional remote
   apply, symlink-following writer, no locking) is not yet closed. We do not
   publish a tag with live engine-safety holes. N3 (stale specs) is also a hard
   blocker. Sequence: fix N3, close gate B, then cut the RC. (Considered and
   rejected: cut now with `remote:` flag-gated experimental — rejected in favour of
   a clean engine-safety story before any public tag.)
3. **Product hierarchy** (F21) — RESOLVED: homonto is the product; onto is its
   native, binary-enforced workflow; Comet/OpenSpec/Superpowers are unenforced
   alternatives. We build with Comet and ship onto. The persona/selection doc that
   makes this honest to users is tracked as N7.
