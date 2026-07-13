# Brainstorm Summary

- Change: onto-skills-shell-out (N1 change B)
- Date: 2026-07-13
- Mode: /goal autonomous — decisions taken at recommended defaults

## Confirmed Technical Approach

Two layers: close binary gaps, then rewrite skills.

### Binary extensions (additive; minimal)
- `onto new --workflow full|fix|tweak` — stop hardcoding `full` (`new.go:81`).
- `onto set base-ref <change> <ref>`; `onto set deps <change> <names…>` (deps as a
  repeatable `--dep` flag or comma list — decide in plan; lean repeatable flag).
- Add **`guides`** to `ontostate.State` as a gated core field, shape
  `pending | updated | waived: <reason>` (validate: `pending`/`updated`/prefix
  `waived:`); `onto set guides <change> <value>`. Additive + legacy-tolerant
  (empty allowed) → **no schema_version bump**.

### Observational — DROP (skills stop writing metrics)
Audit: ~8 skills WRITE metrics/tasks_total/verify_rounds/upgraded, but the review
confirms metrics is observational-only, never a gate. Decision: the skill rewrite
**removes** the metric-writing instructions; the binary keeps the carried
`Observed` fields (now always empty) — **no schema change, no observational
setters**. Metric history is lost; it never gated, so acceptable. (If a later
audit finds a real read-for-decision, revisit — none found.)

### Skill rewrite (8 state-writing skills)
onto, onto-open, onto-design, onto-build, onto-verify, onto-close, onto-fix,
onto-tweak → every direct state write becomes an `onto` command
(`new --workflow`/`advance`/`close`/`set …`); reads via `onto state --json` /
`onto status`. `onto-no-slop` is prose-only (0 state refs) → no change.
Delete the "markdown-only / no external CLI" copy; state the hard binary dep.

## Key Trade-offs and Risks
- Dropping observational loses metric history (accepted; never gated).
- A skill field with no command after the binary extensions → mitigated by the
  field→command audit: after `--workflow`/base-ref/deps/guides, every gated field
  in state-yaml.md maps to a command.
- `--workflow` sets the field only; workflow-aware transition RULES stay N2.

## Testing Strategy
- Binary: TDD per new command (happy + shape-reject); `guides` round-trip + shape.
- Skills: grep-gate — no `onto*` skill contains a direct state-file write, and none
  contains the markdown-only/no-CLI copy. (Full lifecycle dry-run = N7.)

## Spec Patches (onto-binary delta)
- ADD to the command surface: `onto new --workflow`, `onto set base-ref|deps|guides`.
- MODIFY the state-model requirement: add the `guides` gated field.
