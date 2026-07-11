# The Comet Development Workflow

Comet is Homonto's development workflow. OpenSpec owns WHAT: proposals,
requirements, delta specs, and archive semantics. Superpowers owns HOW: deep
technical design, implementation plans, execution discipline, verification, and
branch finishing. Comet state and scripts bind the two.

## Quick Start

- New work: `/comet <what you want to build>`
- Resume work: `/comet`
- Bug fix: `/comet-hotfix <symptom>` when it is an existing behavior bug
- Small tweak: `/comet-tweak <change>` when it is copy/config/docs/prompt-scale

## Layout

```text
.comet/config.yaml
openspec/changes/<name>/.comet.yaml
openspec/changes/<name>/{proposal.md,design.md,tasks.md}
openspec/specs/<capability>/spec.md
docs/superpowers/specs/YYYY-MM-DD-<topic>-design.md
docs/superpowers/plans/YYYY-MM-DD-<feature>.md
docs/superpowers/reports/YYYY-MM-DD-<change>-verify.md
```

## Phase Model

1. Open: clarify goals, non-goals, scope, scenarios, and create OpenSpec artifacts.
2. Design: use Superpowers brainstorming to produce the deep technical design doc.
3. Build: write an implementation plan, choose isolation/execution/TDD/review mode, then execute.
4. Verify: run evidence-based verification and finish branch handling.
5. Archive: merge OpenSpec delta specs into main specs and archive the change.

## Gates

Comet has blocking user decisions for requirements confirmation, change name,
design approach, plan-ready workflow configuration, verify failures, branch
handling, and archive confirmation. Agents must not infer these choices from
history or defaults.

## Change history

Completed change history lives in `openspec/changes/archive/`. New work must use
`openspec/changes/` (active) — start it with `/comet`.
