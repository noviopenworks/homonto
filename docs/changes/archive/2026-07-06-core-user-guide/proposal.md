Preset: tweak

# Proposal: core-user-guide

## Why

Guide coverage is mostly the onto *workflow* (`docs/guides/onto-workflow.md`)
plus the narrow `status-and-adoption.md` from an earlier change. There is no
core "how to use homonto" guide — install, the `homonto.toml` model, the
command surface, secrets, and how projection works (NEXT_AGENT gap #8).

## What Changes

Add `docs/guides/using-homonto.md`: what homonto is, install/build, a
quickstart (`init` → edit → `plan` → `apply`), the `homonto.toml` model (MCPs,
skills, plugins, settings, targets, secret references), the command surface
(`init`/`import`/`plan`/`apply`/`status`/`doctor`/`version`), how projection
works (surgical merge, symlinked skills, pruning, adoption, `state.json`), and
known limitations. Docs only — no source or spec change.

## Capability Impact

- Untouched: no living spec requirement changes. The guide describes behavior
  already specified in `cli-commands`, `config-model`, `secret-references`,
  `apply-pipeline`, and `tool-adapters`.

## Grounding

Specs `docs/specs/{cli-commands,config-model,secret-references,apply-pipeline,
tool-adapters}.md`; secret forms `internal/secret/resolver.go` (`${pass:…}`,
`${VAR}`); command behavior verified against the real binary this session.

## Impact

- Files: `docs/guides/using-homonto.md` (new). Docs only.
- Risk: guide drifting from behavior. Mitigated by grounding every claim in the
  living specs and observed CLI behavior, and keeping known limitations honest.
