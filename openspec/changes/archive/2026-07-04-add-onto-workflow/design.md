# Design (high-level): add-onto-workflow

Deep technical design happens in the design phase (Design Doc + delta spec).
This file records the open-phase architecture decisions and approach selection.

## Approach Selection

Considered:

1. **Fork Comet** (copy skills + scripts, swap paths) — rejected: keeps the
   bash guard/state machinery and openspec CLI dependency we want to remove.
2. **Pure convention, no state file** — rejected by user: phase should be
   explicit and cheap to read; deriving it purely from artifact presence makes
   resume/edge cases ambiguous.
3. **Markdown-only skills + tiny agent-managed state file** — **chosen**: all
   workflow logic lives in SKILL.md prose the agent follows; a small
   `state.yaml` per change records phase and decisions; verifiable file state
   overrides it on conflict. Nothing to install, nothing to execute.

## Architecture

```
content/skills/                     # authored here (homonto-owned)
├── onto/SKILL.md                   # dispatcher: detect phase → route
├── onto-open/SKILL.md              # clarify → proposal/design/tasks
├── onto-design/SKILL.md            # brainstorm → design doc + ADR drafts + spec deltas
├── onto-build/SKILL.md             # plan → TDD tasks → commit per task
├── onto-verify/SKILL.md            # checks vs design/specs → verification.md
├── onto-close/SKILL.md             # merge spec deltas, accept ADRs, write guides, archive
├── onto-fix/SKILL.md               # preset: bugfix (skips design; upgrade rules)
└── onto-tweak/SKILL.md             # preset: small change (skips design+full plan)

homonto.toml [skills] own → homonto apply → symlinks into .claude/skills/

docs/                               # workflow artifact layout (per project)
├── adr/NNNN-<title>.md             # numbered ADRs (proposed → accepted at close)
├── specs/<capability>.md           # living capability specs (deltas merged at close)
├── changes/<name>/                 # active change workspace
│   ├── state.yaml                  # agent-managed: phase, workflow, decisions
│   ├── proposal.md  design.md  tasks.md  verification.md
│   ├── adr/ specs/                 # drafts/deltas staged for close-phase merge
├── changes/archive/YYYY-MM-DD-<name>/
└── guides/<topic>.md               # post-implementation user docs (close phase)
```

## Key Decisions

- **State**: `state.yaml` is a cache of truth, not truth. Every dispatch
  cross-checks it against artifact presence/content; mismatch → correct the
  file, continue from real state.
- **Blocking points preserved**: artifact review (open), approach confirmation
  (design), plan-ready + execution-config (build), fail handling (verify),
  final confirmation (close) — via the platform's question tool.
- **Presets**: `/onto-fix`, `/onto-tweak` skip design; upgrade rules (file
  count, architecture impact, new capability) force the full path.
- **Tooling**: rtk wraps shell ops; graphify (or its codegraph index) is the
  mandated exploration tool in open/design. Both hard-required.
- **GitHub**: resolve-issue / continue-pr are documented entry points that
  start or resume an onto change; PR creation/review remain separate skills.
- **Docs obligation**: close phase cannot complete without updating
  `docs/guides/` (or explicitly recording why no guide change is needed).

## Data Flow

issue/PR/idea → open (proposal, tasks skeleton) → design (design doc, ADR
drafts, spec deltas) → build (plan, TDD commits) → verify (verification.md)
→ close (spec merge, ADR accept, guides, archive) → done.

## Migration (this repo)

`openspec/specs/*` → `docs/specs/`; archived change → `docs/changes/archive/`;
`docs/superpowers/specs|plans|reports` → ADR extraction + archived change
records; retire `openspec/` and `docs/superpowers/`. Comet remains installed
globally but homonto development uses onto from the next change onward.
