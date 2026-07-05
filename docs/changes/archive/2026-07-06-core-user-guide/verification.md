# Verification Report: core-user-guide

- **Date:** 2026-07-06
- **Mode:** light (why: `workflow: tweak`, docs-only change)
- **Range:** 01626ad..HEAD on `tweak/20260706/core-user-guide`
- **Result: pass**

## Scenario evidence

The deliverable is a guide; verification is that every claim matches the specs
and the real binary.

| Claim | Verdict | Evidence |
|---|---|---|
| `init` scaffolds `homonto.toml`, `.gitignore`, `.env.example`, `content/skills/` | pass | real `homonto init <dir>` → `.env.example .gitignore homonto.toml content/skills/` |
| `homonto version` prints `homonto <version>` | pass | real run → `homonto 0.1.0-dev` |
| Command surface (`init/import/plan/apply/status/doctor/version`, `--config`, `import --force`) | pass | matches `cli-commands` spec |
| Secret forms `${pass:…}` / `${VAR}` | pass | `internal/secret/resolver.go` (pass-prefix vs env, error if unset) |
| Validation rules (unknown target, empty command, reserved keys, index-like names) | pass | `config-model` spec + `internal/config/config.go` (this session's #3) |
| status drift-vs-pending wording; adoption "Reconciled N…" | pass | matches observed CLI output + `apply-pipeline` spec |
| doctor checks both tool links | pass | `cli-commands` doctor requirement (this session's #6) |
| Output on stderr caveat | pass | verified this session (`homonto version` → stderr) |
| Internal link `status-and-adoption.md` resolves | pass | file present on branch |

## Design conformance

Tweak — no design.md. Docs-only; no source or spec changed.

## Adversarial pass

Skipped (light mode, optional): a documentation deliverable whose every factual
claim was cross-checked against the living specs and the real binary. Recorded
skip.

## Regression

`go test ./...` → 129 passed (docs-only change; nothing broke). `gofmt`/`vet`
unaffected.

## Deviations

None.
