# Brainstorm Summary

- Change: onto-binary-authoritative-state (N1 change A)
- Date: 2026-07-13

## Confirmed Technical Approach (user-endorsed lean + recon refinements)

- **Schema:** core+typed struct for gated fields + a carried observational bag
  (never gated) + `schema_version` (start at 1). Gated fields: change, workflow,
  phase, created, base_ref, deps, isolation, build_mode, tdd_mode,
  decisions.directive, verify.scale, verify.result, close.merged (+progress).
  Observational (carried, never gated): metrics (per-phase dates), task counts,
  verify rounds, preset_escalated.
- **State file location — RESOLVED by recon:** both planes already use
  `docs/changes/<name>/`. They differ only in filename: binary `onto-state.yaml`
  vs skill `state.yaml`. Canonical = **`docs/changes/<name>/onto-state.yaml`**
  (keep the binary's name; binary is authoritative). onto lives in `docs/changes/`,
  distinct from comet/openspec's `openspec/changes/` — no conflict.
- **Phase vocab:** keep `open|design|build|verify|close`; `archived` is a terminal
  boolean (both planes already model it this way).
- **Migration:** on-read auto-migrate. Legacy binary `onto-state.yaml` (7-field,
  no version) and legacy skill `state.yaml` (rich, no version) both up-migrate to
  the current `schema_version`; writes always emit current version; ordered +
  idempotent. **Reality check:** `docs/changes/` is empty in this repo — nothing
  to migrate here; migration is forward-compat for existing user workspaces.
- **Both-legacy-files conflict policy:** if a dir has BOTH `onto-state.yaml` and
  `state.yaml`, merge observational fields, but if the gated core (phase /
  workflow / archived) disagrees, report **malformed / fail-loud** (this IS the
  divergence bug — do not silently pick a winner). T-honest.
- **CLI surface:** keep `init/new/advance/close/status/doctor`; ADD gated-field
  transition commands (set isolation / build_mode / tdd_mode / verify.scale /
  verify.result / close progress / directive) + a structured `--json` read so
  change B's skills can drive the whole lifecycle without touching a state file.
- **status/doctor:** enumerate `docs/changes/*` directories FIRST, then classify
  each `valid` / `malformed` / `missing-state`. A deleted state file → a reported
  `missing-state` row, never a silent drop (F14).
- **B1 validation:** each gated field gets a presence/shape rule (enum/format).
  The binary validates presence + shape only — never semantic judgment (that's N2).

## Key Trade-offs and Risks

- core+typed vs full-rich: chose core+typed so observational drift can't break
  gating; cost = two field groups to maintain.
- Migration data loss risk mitigated by round-trip tests over real rich
  `state.yaml` fixtures before any write path; low real risk since repo has no
  existing state.
- Schema/command shape must let N2 EXTEND, not rewrite (hence schema_version now).

## Testing Strategy

- Round-trip (marshal→parse) over every gated field + a full rich fixture.
- Migration: both legacy shapes → v1; both-present conflict → malformed.
- Classify: valid / malformed / deleted-state→missing-state (F14 regression).
- Per new command: happy path + validation rejection.
- `-race`, `go vet`, `openspec validate --all` green.

## Spec Patches (to onto-binary delta)

- MODIFY the onto-state.yaml model requirement → versioned core+typed schema.
- MODIFY status/doctor requirements → enumerate-then-classify.
- ADD gated-field transition + structured read commands to the command surface.
- (Confirm exact requirement boundaries when writing the delta.)

## Open (refine in the plan, not blocking design confirm)

- Exact command naming/granularity (semantic per-field setters vs one guarded
  `onto state set`). Leaning semantic setters for gated fields.
