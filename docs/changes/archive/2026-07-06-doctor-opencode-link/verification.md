# Verification Report: doctor-opencode-link

- **Date:** 2026-07-06
- **Mode:** light (why: `workflow: fix`, single-function change in one module)
- **Range:** 1217944..HEAD on `fix/20260706/doctor-opencode-link`
- **Result: pass**

## Scenario evidence

| Requirement / Scenario | Verdict | Evidence (fresh command + output) |
|---|---|---|
| cli-commands: Missing OpenCode link is flagged | pass | Real `doctor` (claude link only) → `ok: skill "s" linked (claude)` + `warn: skill "s" content present, not linked for opencode (run apply)`. `TestDoctorChecksOpenCodeSkillLink` PASS |
| cli-commands: both links ok when present | pass | Real `doctor` after linking both → `ok: skill "s" linked (claude)` + `ok: skill "s" linked (opencode)` |
| cli-commands: Missing owned skill is flagged (unchanged) | pass | `TestDoctorFlagsMissingSkillContent` PASS |
| Existing Claude-link reporting preserved | pass | `TestDoctorReportsSkillLinkState` PASS (substring assertions still hold) |

## Design conformance

Preset fix — no design.md. `engine.Doctor` now loops both tool links
(`~/.claude/skills/<name>`, `~/.config/opencode/skills/<name>`) per owned skill,
reporting `ok`/`warn` per tool, matching the proposal and the MODIFIED
`cli-commands` doctor requirement.

## Adversarial pass

Skipped (light mode, optional): the change is a single localized loop over two
known link paths, fully exercised by the new test (missing → warn, present →
ok, per tool) plus a real-binary smoke, and all pre-existing doctor tests still
pass. Recorded skip per `onto-verify/references/adversarial.md`.

## Regression

- `go build ./...` → Success
- `go vet ./...` → No issues found
- `go test ./...` → 129 passed in 15 packages
- `gofmt -l internal/` → (empty)

## Deviations

None.
