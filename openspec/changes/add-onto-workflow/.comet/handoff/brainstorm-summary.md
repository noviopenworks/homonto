# Brainstorm Summary

- Change: add-onto-workflow
- Date: 2026-07-04

## Confirmed Technical Approach

CONFIRMED by user 2026-07-04 ("Confirm all"): design as below, plus build
config isolation=branch, execution=direct, tdd_mode=direct, verify_mode=full,
autonomous run-through to publish-ready.

- **onto** skill set: 8 markdown-only skills under `content/skills/`
  (`onto`, `onto-open`, `onto-design`, `onto-build`, `onto-verify`,
  `onto-close`, `onto-fix`, `onto-tweak`). No scripts, no external CLI.
- Dogfood discovery: `homonto apply` symlinks `content/skills/<name>` →
  `~/.claude/skills/<name>` (user-global), so onto becomes available in all
  projects — repo needs a root `homonto.toml` (`[skills] own`) which does not
  exist yet and must be created.
- Layout contract per project: `docs/adr/`, `docs/specs/`, `docs/changes/`
  (+ `archive/`), `docs/guides/`.
- State: `docs/changes/<name>/state.yaml`, agent-managed; file state wins on
  conflict; dispatcher re-derives phase every invocation.

## Candidate design decisions (pending confirmation)

1. **ADR staging**: draft ADRs live unnumbered in the change workspace
   (`docs/changes/<name>/adr/<slug>.md`, status: Proposed); at close they get
   the next global number and move to `docs/adr/NNNN-<slug>.md` (status:
   Accepted). Keeps docs/adr/ free of abandoned-change pollution.
   (Alternative rejected: direct numbering at draft time — collision +
   abandonment noise.)
2. **Spec format**: keep OpenSpec-style requirement blocks (SHALL +
   `#### Scenario:` given/when/then) in both living specs and deltas
   (ADDED/MODIFIED/REMOVED sections); verify phase checks scenarios.
   (Alternative rejected: freeform specs — unverifiable.)
3. **Plan location**: `docs/changes/<name>/plan.md` inside the workspace
   (not a separate plans tree).
4. **state.yaml schema**: change, workflow (full|fix|tweak), phase,
   created, base_ref, decisions{isolation, execution, tdd}, verify{mode,
   result, report}, guides (pending|updated|waived:<reason>), archived.
5. **rtk/graphify preflight** in dispatcher: `rtk --version` +
   graphify skill or `graphify-out/`/`.codegraph/` index; missing → halt
   with install instructions.
6. **GitHub entry points**: resolve-issue → starts `/onto` change in
   worktree; continue-pr → resumes change (or opens fix change) from PR
   feedback. Documented contract in dispatcher; those global skills keep
   working with comet until user migrates them (out of scope here).
7. **Migration mapping**: openspec/specs/* → docs/specs/<name>.md;
   archived v1-core change + superpowers specs/plans/reports → merged
   archived-change dirs under docs/changes/archive/; roadmap →
   docs/roadmap.md; extract 4 ADRs (plan/confirm/apply+adapters,
   secrets-referenced-never-stored, symlinked-owned-content/surgical-merge,
   atomic-writes-state-last) + 1 new (adopt-onto-workflow); retire
   openspec/ + docs/superpowers/ at the very end (this change's own
   workspace moves post-archive).

## Key Trade-offs and Risks

- Agent-managed state can drift → mitigated by file-state-wins cross-check
  table in dispatcher.
- No guard scripts means no hard enforcement → mitigated by explicit exit
  checklists in each skill + verification-before-completion style evidence.
- Bootstrapping recursion: this change is built with comet while creating
  onto; final migration retires comet's directories after comet-archive runs.
- `homonto apply` writes to real `~/.claude/skills/` — intended (dogfood).

## Testing Strategy

- Markdown deliverables: validation = dry-run lifecycle walkthroughs (full
  path + both presets + upgrade trigger + resume-after-drift), symlink load
  check, and grep-based self-containment check (no openspec CLI / comet
  script references).
- `go test ./...` must stay green (no Go changes expected).

## Spec Patches

- Create delta spec `specs/onto-workflow/spec.md` (new capability) with
  requirements: phase model & dispatch, artifact layout, state & recovery,
  design rigor gates, presets & upgrade rules, tooling preflight, GitHub
  entry points, close-phase docs obligation, dogfood wiring.
