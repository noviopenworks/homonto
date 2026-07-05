# Design: state-source-of-truth

Status: Confirmed
Confirmed: 2026-07-05 (Approach 1 — first-class `adopt` action + drift decoupled via secret-safe adapter observation)

## Summary

`state.json`'s `Entry.Applied` (sha256 of the last-applied resolved value) is
already the right authority for "does disk still match what we last applied?"
but the non-secret code path never consults it. This change makes it
authoritative for both the managed set and drift.

Two mechanisms:

1. **Adoption.** A new invisible `adopt` action. When Plan sees a declared,
   non-secret key that is present on disk, equals desired, and is **absent
   from state**, it emits `adopt` instead of `noop`. `adopt` renders no diff
   line. On apply the adapter records state (`st.Set`) *without writing tool
   files*. `apply` performs this reconciliation even when adoption is the
   only pending work (state-only, no confirmation prompt, one-line summary).
   Adopted keys thereby become visible to pruning and drift.
2. **Drift from disk-vs-state.** A new secret-safe adapter capability returns
   per-recorded-key disk **hashes** (only hashes leave the adapter).
   `engine.Status()` compares each hash to `Entry.Applied` — a key is drifted
   iff its on-disk value differs from what was last applied (or is missing).
   Un-applied `homonto.toml` edits are reported **separately** as pending, not
   as drift.

Rejected alternatives: **Approach 2** (overload `noop`, thread `Change.DiskHash`,
keep drift Plan-derived) — muddier semantics, "No changes" path secretly writes
state, drift stays coupled to Plan. **Approach 3** (adapter-owned StatusReport)
— largest interface change, duplicates pending logic per adapter, over-engineered
for two tools.

## Goals / Non-Goals

**Goals:** make `Entry.Applied` authoritative for the managed set (adoption →
pruning/drift visibility) and for drift (disk-vs-state, pending reported
separately); identical behavior across the claude and opencode adapters.

**Non-goals:** secret-key adoption (secrets that are unrecorded re-apply as
`update`, as today); any `state.json` on-disk format change (`Applied` already
exists); other NEXT_AGENT backlog items (#3–#8).

## Architecture

### New action: `adopt`

`adapter.Change.Action` gains a fourth value `"adopt"` (still a string
literal; the enum comment updates). It flows through the four action-literal
sites:

- **Plan** (`claude.go` inline non-secret branch; `opencode.go` `planKey` and
  the plugin branch): the non-secret "disk == desired" case splits — true
  `noop` **only when** `inState && Entry.Applied == secret.Hash(canonical(disk))`
  (state already records this exact on-disk value); otherwise `adopt` (the key
  is unrecorded, or recorded with an absent/stale `Applied` hash — e.g. its
  disk was reconciled out of band to a value that now equals desired). This
  mirrors the secret branch's existing noop condition exactly, so both branches
  consult `Entry.Applied`. `adopt` carries `New = want` (unresolved desired) so
  apply can record `Entry.Desired`. Secret keys are never adopted (only
  `!secret.ContainsRef(want)` reaches this branch). Refreshing a stale `Applied`
  is what clears otherwise non-actionable disk-vs-state drift on the next apply.
- **`plan.Render`**: `adopt` produces no line (like `noop`) — plan stays
  silent about adoption.
- **`plan.HasChanges`**: means "visible change" (create/update/delete) — it
  must **exclude** `adopt` (its `!= noop` form silently began counting `adopt`
  when the action was added; restore the contract). A new helper
  `plan.HasAdoptions(sets)` reports whether any `adopt` is pending.
- **`engine.Apply` resolve loop**: `adopt` is skipped alongside `noop`/`delete`
  (non-secret by construction — nothing to resolve).
- **adapter `Apply`**: before the file-write switch, `adopt` does
  `st.Set(tool, key, c.New, secret.Hash(canonical(resolve(c.New))))` and
  `continue` — **no file write**. Because `adopt` fires only when
  `canonical(disk) == canonical(want)` for a non-secret key,
  `resolve(want) == want` and the stored hash equals `hash(canonical(disk))`,
  matching what a real write would have recorded (and what drift compares
  against). No disk read needed at apply.

### Conditional tool-file writes (adopt writes only `state.json`)

Today each adapter's `Apply` rewrites its tool file(s) unconditionally at the
end, even when no managed key changed — which for opencode strips JSONC
comments and for claude reformats. For adoption to literally "not modify the
tool file" (as the delta spec requires), each adapter tracks whether a
managed key **in a given file** actually changed (create/update/delete) and
calls `WriteAtomic` for that file only when it did. `adopt` and `noop` never
mark a file changed. Mapping: claude `mcp.*` → `.claude.json`; claude
`setting.*`/`plugin.*` → `settings.json`; opencode `mcp.*`/`setting.*`/
`plugin.*` → `opencode.jsonc`; `skill.*` is symlink work (separate).
Consequence: an adopt-only (or noop-only) apply leaves every tool file
byte-identical and writes only `state.json`; a skills-only apply no longer
reformats the JSON files either (a welcome side effect, not the goal).

### apply.go flow

Replace the single short-circuit with a three-way branch:

- `!HasChanges && !HasAdoptions` → print `No changes. Everything up to date.`
- `!HasChanges && HasAdoptions` → run `e.Apply(sets)` directly (adoption
  touches only `state.json`, so no `[y/N]` prompt), then print
  `Reconciled N pre-existing resource(s) into state.`
- otherwise → render + prompt + apply as today (adoptions ride along silently
  within a normal apply).

### Drift decoupled from Plan

New adapter method on the `Adapter` interface:

```
ObserveHashes(st *state.State) (map[string]string, error)
```

Returns `key -> observed hash` for every key recorded in state for that tool
**that is still present on disk**; recorded keys absent from disk are omitted
(the engine infers "missing"). All disk reads and hashing happen inside the
adapter, so only hashes escape — secret-safe. The observed hash is computed
the same way the key's `Applied` was stored, per prefix:
- `mcp.*`/`setting.*`: `secret.Hash(canonical(on-disk JSON value))` — claude
  reads via its existing `current()`; opencode reads its file once and
  extracts each recorded key.
- `plugin.*`: present in the array/object → `secret.Hash(canonical(mustJSON(
  name)))` (claude stores plugins as `enabledPlugins` object → value `true`;
  match its stored form); absent → omit.
- `skill.*`: `readlink` the destination; present → `secret.Hash(dst + " -> " +
  target)` (matching how a link's `Applied` is stored at apply); absent or not
  a symlink → omit. This keeps the existing "status reports drift if a link is
  changed out-of-band" requirement working under the new drift computation.

`engine.Drift` is rewritten (and wrapped by a new `engine.Status()`):

```
Status() (drift []string, pending int, err error)
```

- For each tool, `observed = adapter.ObserveHashes(state)`. For each recorded
  key: absent from `observed` → `"<tool> <key> missing (deleted out of band)"`;
  else `observed[key] != Entry.Applied` → `"<tool> <key> drifted"`. Collect
  drifted keys.
- `pending` = count of `Plan()` visible changes (create/update/delete) whose
  `(tool,key)` is **not** in the drifted set — i.e. config edits whose disk
  still matches the last apply, plus genuinely new keys.

`status.go` CLI calls `Status()`: prints warnings, drift lines, and — when
`pending > 0` — `N config change(s) awaiting apply (run \`homonto apply\`)`;
`No drift.` when both are empty.

### Files touched

- `internal/adapter/adapter.go` — `Adapter` interface gains `ObserveHashes`;
  `Change.Action` enum comment adds `adopt`.
- `internal/adapter/claude/claude.go` — Plan adopt branch; Apply adopt branch;
  `ObserveHashes`.
- `internal/adapter/opencode/opencode.go` — `planKey` + plugin adopt branch;
  Apply adopt branch; `ObserveHashes`.
- `internal/plan/plan.go` — `HasAdoptions`; Render leaves `adopt` unrendered
  (already the default — no `adopt` case).
- `internal/engine/status.go` — rewrite `Drift`; add `Status`.
- `internal/cli/status.go` — call `Status`, print pending line.
- `internal/cli/apply.go` — three-way flow.

## Key decisions

1. **Adopt as a first-class silent apply-time action** (ADR draft
   `docs/adr/0009-adopt-preexisting-resources-into-state.md`). Why not overload `noop`:
   apply short-circuits on `!HasChanges`, so a plain-`noop` adoption would
   never run when it is the only work — the primary adoption scenario. A
   distinct action lets `HasAdoptions` drive a state-only reconcile while the
   plan diff and the `[y/N]` prompt stay reserved for tool-file changes.
2. **Compute drift from disk-vs-state, not from the desired plan** (ADR draft
   `docs/adr/0010-drift-from-disk-vs-state.md`). Why a new observation method rather
   than reusing `Plan()`: reusing Plan is the root cause of gap #2 (Plan is
   desired-centric). A narrow hash-only method decouples drift, keeps disk
   values (incl. resolved secrets) inside the adapter, and lets the engine own
   the drift-vs-pending policy.

## Error handling

- An adapter whose file is unparseable is already skipped by `Plan()` with a
  warning; `ObserveHashes` mirrors that — it returns the same error and
  `Status` records a warning and continues with the other tool (never halts,
  never reports false "No drift").
- Adoption never resolves secrets or writes tool files, so it cannot introduce
  a partial-apply hazard; state is still saved per-adapter (unchanged).
- If a key is adopted and its disk value is later found to be a secret-shaped
  string, that is impossible by construction — `adopt` only fires for
  `!secret.ContainsRef(want)` — so no plaintext secret is ever hashed into an
  adopted record differently than a normal apply would.

## Testing strategy

Table-driven adapter + engine tests (colocated), plus a CLI smoke:

- **Adoption:** declared MCP present on disk == desired, not in state → after
  apply, `state.json` records it, tool file byte-unchanged, plan showed no
  diff line. Then remove from config → plan shows a delete (pruneable).
- **Adoption-only apply:** config all-matching with one unrecorded key →
  `apply` reconciles without a prompt and prints the reconcile summary; a
  second apply is a true no-op ("No changes").
- **Drift true positive:** applied key, disk edited out of band → `status`
  reports drifted.
- **Drift true negative (the gap):** applied key, `homonto.toml` desired
  edited, disk unchanged → `status` does **not** report drift; reports
  `1 config change awaiting apply`.
- **Missing:** recorded key deleted from disk → `status` reports missing.
- **Parity:** the adoption + drift cases run for both claude and opencode
  (incl. opencode plugin array membership).
- **Secrets unchanged:** existing secret noop/update/drift behavior preserved
  (secretsafety tests still pass; a secret never enters the adopt path).
- Gate: `go test ./...`, `go vet ./...`, `go build`, `go test -race ./...`,
  and a manual `status`/`apply` smoke on a scratch config demonstrating the
  pending-vs-drift distinction and silent adoption.

## Grounding

Direct reads (2026-07-05): `claude.go:74-140,170-246` (Plan/Apply/current),
`opencode.go:52-117,130-221` (Plan/planKey/Apply), `plan.go:11-44`
(HasChanges/Render), `state.go:17-88` (Entry/Set/Get/Keys), `engine.go:64-116`
(Plan/Apply), `status.go:14-36` (Drift), `cli/apply.go:42-58` (short-circuit),
`cli/status.go:21-35` (Drift call). Full anchor list in notes.md.
