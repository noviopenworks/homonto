---
comet_change: onto-binary-authoritative-state
role: technical-design
canonical_spec: openspec
---

# onto-binary-authoritative-state — Technical Design

Deep design for N1 change A: make the onto Go binary the single authority for
onto workflow state, so the markdown skills (change B) can stop writing state by
hand. Binary-authoritative + B1 (enforce presence/shape of state, trust the
agent's judgment) under a T-honest threat model.

## Context (verified 2026-07-13)

- Binary state: `internal/ontostate.State` — 7 fields (change, workflow, phase,
  created, base_ref, deps, archived), **no version**; file
  `docs/changes/<name>/onto-state.yaml` (`internal/ontocli/new.go:70,85`).
- Skill state: `docs/changes/<name>/state.yaml` — ~20 rich fields
  (`catalog/skills/onto/references/state-yaml.md`).
- **Both planes already use `docs/changes/<name>/`.** The only path divergence is
  the filename (`onto-state.yaml` vs `state.yaml`).
- `docs/changes/` is **empty** in this repo — no state to migrate here; migration
  is forward-compatibility for existing user workspaces.
- Commands today: `init/new/advance/close/status/doctor`
  (`internal/ontocli/root.go`).

## Architecture

Four units, each independently testable.

### 1. Versioned state schema (`internal/ontostate`)

```
type State struct {
    SchemaVersion int          // gated: must be a known version
    Core          Core         // gated fields
    Observed      Observed     // carried, never gated
}
```

- **Core (gated — validated for presence/shape, never for judgment):**
  change, workflow (full|fix|tweak), phase (open|design|build|verify|close),
  created, base_ref, deps, isolation (branch|worktree|""),
  build_mode (direct|subagent|""), tdd_mode (tdd|direct|""),
  verify.scale (light|full|""), verify.result (pending|pass|fail),
  close.merged (bool), archived (bool), decisions.directive (free string).
- **Observed (carried, never gated):** metrics (map phase→date), task counts,
  verify rounds, preset_escalated.

`schema_version` starts at 1. Validation is presence/shape only: e.g.
`isolation ∈ {branch,worktree,""}`. B1 means the binary rejects a *malformed*
value, never a *substantively unconvincing* one.

**Why core+typed, not one flat struct:** an unknown/garbage observational field
can never break a gate; the two groups marshal to one YAML doc but validate
separately.

### 2. Migration (`internal/ontostate`, on-read)

`Load(path)` recognizes three inputs and up-migrates to the current version:
1. **Legacy binary** `onto-state.yaml` (7 fields, no version) → v1 (unknown
   gated fields default empty; observational empty).
2. **Legacy skill** `state.yaml` (rich, no version) → v1 (map its fields onto
   Core/Observed).
3. **Current** versioned schema → used as-is.

Writes always emit the current version. Migration is ordered (v0→v1→…) and
idempotent (migrating a current doc is a no-op).

**Both-legacy-files-present conflict policy:** if a dir holds BOTH
`onto-state.yaml` and `state.yaml`:
- merge Observed (union; skill's richer set wins per-field);
- if any **gated core** field disagrees (phase / workflow / archived), the state
  is **malformed** — surface it as a fail-loud error / a `doctor` finding, never
  silently pick a winner. This disagreement IS the divergence bug (F1/F3); the
  human resolves it.

Canonical file is `onto-state.yaml`; a migrated `state.yaml` is folded into it
and the stale `state.yaml` removed on the next write (or left for change B /
manual cleanup — decided in the plan; not gating).

### 3. CLI transition + read surface (`internal/ontocli`)

Keep `init/new/advance/close/status/doctor`. Add:
- **Gated-field transition commands** — one semantic command per gated mutation
  the skills do today by hand (set isolation, build_mode, tdd_mode, verify scale,
  verify result, close progress, directive). Each validates presence/shape and
  writes through the schema. Semantic-per-field (not a raw `state set k v`) so the
  binary owns the rule for each field.
- **Structured read** — `onto state <name> --json` emits the full validated state
  as JSON so a skill queries state without parsing YAML.

Exact command names/grouping are refined in the implementation plan; the design
commitment is: *every gated mutation and a full read are reachable via the CLI.*

### 4. status/doctor: enumerate → classify

Invert today's "enumerate `docs/changes/*/onto-state.yaml`". First enumerate
change **directories** under `docs/changes/` (excluding `archive/`), then classify
each:
- `valid` — state file present, parses, validates;
- `malformed` — present but unparseable/invalid (incl. the both-files conflict);
- `missing-state` — a change directory with no state file.

A deleted state file becomes a reported `missing-state` row (F14) instead of a
silent disappearance.

## Testing strategy

- **Schema round-trip:** marshal→parse preserves every gated field; a full rich
  fixture (mirroring `state-yaml.md`) survives with no gated field dropped.
- **Migration:** legacy 7-field → v1; legacy rich → v1; current → no-op;
  both-present with agreeing core → merged; both-present with disagreeing core →
  malformed error.
- **CLI:** each transition command — happy path + a shape-rejection case;
  `--json` read emits valid JSON matching the state.
- **status/doctor:** valid / malformed / deleted-state→`missing-state`
  (F14 regression) across a `docs/changes/` fixture tree.
- Gate: `go test ./internal/ontostate/... ./internal/ontocli/... -race`,
  `go vet`, `go build ./...`, `openspec validate --all`.

## Risks

- **Migration data loss** — a wrong field map drops skill-only state. Mitigation:
  round-trip + migration tests over a real rich fixture before any write path
  ships. Real risk low (repo has no existing state).
- **Schema churn vs N2** — N2 adds gate *semantics*. Keep Core/commands shaped so
  N2 extends (new validation on existing fields, new commands), not rewrites;
  `schema_version` makes future migration ordered.
- **Filename cleanup** — removing a folded `state.yaml` touches user files; keep
  it conservative (fold-then-remove-on-write, or defer to change B). Decided in
  plan.

## Non-goals

Rewriting `onto*` skills / deleting the markdown-only copy (change B); semantic
gate content, workflow-aware transitions, dep resolver (N2); homonto-engine work.
