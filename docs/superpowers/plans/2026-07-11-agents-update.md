---
change: agents-update
design-doc: docs/superpowers/specs/2026-07-11-agents-update-design.md
base-ref: fa11571692a6a772278e0274ff77d752d809ad03
---

# Plan: agents update (v2 #4)

`homonto agents update`: re-materialize an installed local: agent from source,
backup-safe, idempotent. See Design Doc D1/D2. TDD.

## Task 1: `homonto agents update` (`internal/cli`)

- [ ] 1.1 (TDD RED first) `agentsUpdateCmd` per Design Doc: undeclared→err; non-local→"not yet supported"; not-installed→err→`agents add`; resolve source (missing→err); per declared target (sorted) — copy: up-to-date if on-disk hash==source hash, else BACKUP dst→dst.bak ONLY when on-disk != prev.Hash AND != source hash then WriteAtomic source; link: up-to-date if isSymlinkTo else link.Link; record Install{path,source-hash}; Save lock; print. Register `update`.
- [ ] 1.2 (TDD RED first) Tests (build via `agents add`, perturb): source changed→rewrites+hash refreshed, no .bak; locally-modified+source-changed→.bak has old content + new source written; idempotent→"up to date", no .bak/rewrite; not-installed→err→add; builtin→"not yet supported"; undeclared→err; link-mode→"up to date"/valid symlink.
- [ ] 1.3 GREEN; gofmt/vet clean. Commit: `feat(cli): 'homonto agents update' re-materializes an installed agent (backup-safe)`

## Task 2: Regression and docs

- [ ] 2.1 Full regression (build/test/-race/vet/gofmt/mod tidy). E2E (real `homonto`): add copy agent; edit source→update refreshes (doctor healthy); edit install+source→update makes .bak; re-run→"up to date".
- [ ] 2.2 Update `docs/roadmap.md` v2 status + README (mention `agents update`). No over-claim (backup not merge).
- [ ] 2.3 Commit all changes.
