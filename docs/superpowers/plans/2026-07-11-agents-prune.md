---
change: agents-prune
design-doc: docs/superpowers/specs/2026-07-11-agents-prune-design.md
base-ref: 7b678aaf31e4ad78b8af63998fe1e7b12b91fddf
---

# Plan: agents prune (v2 polish)

`homonto agents prune`: remove orphaned/de-declared agent installs, backup-safe.
See Design Doc D1/D2/D3. TDD.

## Task 1: `homonto agents prune`
- [x] 1.1 (TDD RED first) `agentsPruneCmd` (prune, NoArgs, --dry-run) per D1/D2/D3: per lockfile agent — orphan→prune all target files + drop lock.Agents[name]; de-declared target→prune that file + drop from Installed. pruneFile: skip missing; dry-run→"would remove"; else .bak when on-disk hash != recorded, remove file + .merged sidecar. Report; "nothing to prune"; --dry-run no writes/Save; Save once when changed. Register under agentsCmd.
- [x] 1.2 (TDD RED first) Tests: orphan→file removed+entry gone; de-declared target→that file only+agent stays; local-edit orphan→.bak; .merged removed; nothing-to-prune; --dry-run changes nothing.
- [x] 1.3 GREEN; gofmt/vet clean. Commit: `feat(cli): 'homonto agents prune' removes orphaned/de-declared agent installs`

## Task 2: Regression and docs
- [x] 2.1 Full regression (build/test/-race/vet/gofmt/mod tidy). E2E: add agent; remove from config; doctor→orphan; prune --dry-run lists; prune removes; doctor→healthy.
- [x] 2.2 Update `docs/roadmap.md` v2 + README (agents prune). No over-claim.
- [x] 2.3 Commit all changes.
