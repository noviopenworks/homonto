## 1. `homonto agents update` (`internal/cli`)

- [x] 1.1 (TDD RED first) `agentsUpdateCmd` (`update <name>`, ExactArgs(1)) per Design Doc D1/D2: setup like `add`; undeclared→err; non-local→"not yet supported"; not-installed (absent from lockfile)→err pointing to `agents add`; resolve source (missing→err); per declared target (sorted) re-materialize by mode — copy: up-to-date if on-disk hash==source hash else back up to `<dst>.bak` ONLY when on-disk hash != recorded prev.Hash AND != source hash (genuine local edit) then WriteAtomic source; link: up-to-date if isSymlinkTo(dst,src) else link.Link; record Installed{path, source-hash}; Save lock; print per-target status. Register `update` under `agentsCmd()`.
- [x] 1.2 (TDD RED first) Tests (build state via `agents add`, then perturb): source changed → update rewrites each target to new content + lockfile hash refreshed, no .bak (install was untouched); locally-modified install (edit dst) then source also changed → update backs up dst to dst.bak (old content) and writes new source; idempotent (no perturbation) → "up to date", no .bak, no rewrite; not-installed agent → err → `agents add`; builtin → "not yet supported"; undeclared → err; link-mode update after source change → symlink still valid, "up to date" (link points at source).
- [x] 1.3 GREEN; gofmt/vet clean. Commit: `feat(cli): 'homonto agents update' re-materializes an installed agent (backup-safe)`

## 2. Regression and docs

- [x] 2.1 Full regression (build/test/-race/vet/gofmt/mod tidy). E2E (real `homonto`): add copy agent; edit source → `agents update` refreshes install (doctor then healthy); edit the installed file + source → update creates `.bak` with the local edit; re-run update → "up to date".
- [x] 2.2 Update `docs/roadmap.md` v2 status + README (mention `homonto agents update`). No over-claim (backup, not merge).
- [x] 2.3 Commit all changes.
