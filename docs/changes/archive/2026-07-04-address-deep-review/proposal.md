# Proposal: address-deep-review

## Why

The 2026-07-04 deep review (docs/reviews/2026-07-04-deep-review.md)
found the product broken at its primary job: the Claude MCP projection
writes a schema Claude Code cannot consume (P0), import corrupts working
configs (P0), plan leaks resolved secrets when state is missing
(CRITICAL, reproduced), nothing is ever pruned ("declarative" is false),
sjson path injection corrupts live configs, and the repo is legally
unusable (no LICENSE) and undistributable. The onto workflow needs its
worst ceremony defects fixed. User directive: implement until finished.

## What Changes

- **Claude MCP schema**: emit `{type: "stdio", command: <string>,
  args: [...], env: {...}}`; conformance fixtures taken from real tool
  files. **BREAKING** for configs applied with the old shape (apply
  corrects them).
- **Import**: read real Claude schema (`command` string + `args`),
  tolerate legacy array form; expanded redaction patterns
  (`*_SECRET`, `*_PASSWORD`, `glpat-`, `npm_`, `AIza`, `Bearer`).
- **Secret safety**: unknown-provenance on-disk values (key not in
  state) are redacted in plan output; `writeAtomic` preserves existing
  file mode, defaults 0600, fsyncs before rename; unique temp names.
- **Pruning**: new `delete` action — keys recorded in state but absent
  from desired config are removed from tool files, state entries
  garbage-collected, owned-skill symlinks recorded in state and removed
  when unowned (only if they are homonto's own links).
- **Robustness**: sjson/gjson path escaping for dotted keys; skill
  names validated as single path elements; deterministic (sorted) plan
  output; per-adapter state save (partial apply leaves a record);
  memoized secret resolver (one `pass` call per token); clear error on
  non-object JSON roots.
- **Hygiene**: MIT LICENSE, GitHub Actions CI (go vet + test),
  `var Version` (ldflags-stampable), README honesty pass (JSONC
  comment loss is total; owned content = skills only; declarative
  claims match pruning reality).
- **onto v2.1**: tweak preset covers small features (≤5 files, no new
  capability, no spec change); rtk/graphify preflight warns and
  proceeds instead of halting; onto-close rewrites workspace ADR links
  to final `docs/adr/NNNN` paths before archiving; ADR 0007 skip-
  recording sentence corrected (errata note); guide synced.

## Capability Impact

- **Modified**: `tool-adapters` (schema, pruning, modes, escaping,
  ordering), `apply-pipeline` (per-adapter state save, single
  resolution, unknown-provenance redaction), `cli-commands` (import
  behavior, version), `secret-references` (redaction-when-unknown),
  `onto-workflow` (preset scope, preflight, close link rewrite).
- Untouched: config-model.

## Not split

The review is one endorsed work unit; Go fixes share files (adapters,
utils, engine) and the docs/spec updates must land with the behavior
they describe. onto v2.1 rides along because its spec lives in the same
delta cycle. Pushing to origin is explicitly out of scope.

## Grounding

docs/reviews/2026-07-04-deep-review.md (file:line evidence, two
findings reproduced against the built binary; MCP schema verified
against the live ~/.claude.json).

## Impact

- Modified: internal/adapter/{adapter,claude,opencode}, importer,
  jsonutil, link, plan, engine, secret, cli/root; testdata fixtures
  added; README; content/skills/onto*, docs contracts/guide; ADR 0007.
- New: LICENSE, .github/workflows/ci.yml, conformance test fixtures.
