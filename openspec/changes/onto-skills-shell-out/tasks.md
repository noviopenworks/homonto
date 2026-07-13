# Tasks — onto-skills-shell-out

Open-phase outline. The design phase resolves the deferred decisions (observational
drop-vs-keep, per-skill field→command audit, flag shapes); the build phase turns
these into a detailed plan.

## 1. Binary: workflow at creation
- [x] `onto new --workflow full|fix|tweak` (validated); stop hardcoding `full`.
- [x] Tests: each workflow accepted; bad value rejected.

## 2. Binary: creation-field setters
- [x] `onto set base-ref <change> <ref>`.
- [x] `onto set deps <change> <names…>` (flag shape decided in design).
- [x] Tests: happy path + reads back.

## 3. Binary: guides field + setter
- [x] Add `guides` gated field to `ontostate.State` (shape `pending|updated|
      waived: <reason>`) with validation; confirm whether a schema_version bump
      is needed.
- [x] `onto set guides <change> <value>`.
- [x] Round-trip + shape-reject tests.

## 4. Binary: observational decision
- [x] Design-confirmed drop OR keep+setters. If drop: remove
      metrics/tasks_total/verify_rounds/preset_escalated from the model after
      confirming no skill/doctor path depends on them; update tests.

## 5. Per-skill field→command audit (design gate)
- [x] Enumerate every state write in each of onto, onto-open, onto-design,
      onto-build, onto-verify, onto-close, onto-fix, onto-tweak and map it to a
      command. Confirm no residual field lacks one after tasks 1–4.

## 6. Rewrite skills to shell out
- [x] Replace every direct state-write instruction in the 8 skills with the mapped
      `onto` command; reads via `onto state --json` / `onto status`.
- [x] Confirm `onto-no-slop` needs no change.

## 7. Delete the markdown-only / no-external-CLI copy
- [x] Remove the "markdown-only" / "no external CLI" claims from `onto/SKILL.md`
      and any sibling; state the hard binary dependency.

## 8. Enforcement gate + verification
- [x] Grep-based CI gate: no `onto*` skill contains a direct state-file write and
      none contains the markdown-only/no-CLI copy.
- [x] `openspec/specs/onto-binary/spec.md` delta for the added commands + guides.
- [x] `go test ./internal/ontostate/... ./internal/ontocli/... -race`, `go vet`,
      `go build`, `openspec validate --all` green.

## 9. Out of scope (recorded)
- [x] (note only) workflow-aware transition *rules*, semantic gates, dep resolver
      → N2; full-lifecycle skill dry-run → N7 onto E2E suite.
