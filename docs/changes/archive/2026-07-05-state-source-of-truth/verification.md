# Verification Report: state-source-of-truth

- **Date:** 2026-07-05
- **Mode:** full (why: `workflow: full` and the diff touches far more than 5 files in `base_ref..HEAD`)
- **Range:** a72e535..HEAD on `feature/20260705/state-source-of-truth`
- **Result: pass**

## Scenario evidence

All evidence is fresh from this verify round: named tests run verbose, plus a
real-binary manual smoke in a temp `$HOME`. Full suite: `go test ./...` → 125
passed in 15 packages.

| Requirement / Scenario | Verdict | Evidence (fresh command + output) |
|---|---|---|
| apply-pipeline: Pre-existing matching key adopted on apply | pass | `go test -run TestAdoptRecordsStateWithoutWritingFile -v` → PASS (state recorded, `.claude.json` byte-identical via `bytes.Equal`). Smoke: `apply` → `Reconciled 1 pre-existing resource(s) into state.`, `diff` → identical |
| apply-pipeline: Adoption-only apply runs without a prompt | pass | `go test -run TestApplyAdoptionOnlyReconcilesWithoutPrompt -v` → PASS. Smoke: `apply </dev/null` (no `--yes`) → `Reconciled 1...`, no `[y/N]` prompt, no diff |
| apply-pipeline: Stale applied hash refreshed, phantom drift clears | pass | `go test -run TestStaleAppliedRefreshedViaAdopt` + `TestOpenCodeStaleAppliedRefreshedViaAdopt` → PASS. Smoke: disk+config both `sonnet`, state `hash(opus)` → `status` "drifted", `apply` byte-identical, `status` after → `No drift.` |
| apply-pipeline: Adopted key becomes pruneable | pass | `go test -run TestAdoptedKeyIsPruneable` / `TestOpenCodeAdoptedKeysArePruneable` → PASS. Smoke: blank the config → plan shows `-` delete |
| apply-pipeline: Out-of-band change surfaces (drift) | pass | `go test -run TestStatusDetectsDriftAfterOutOfBandChange -v` → PASS. Smoke: edit disk → `claude setting.model drifted (will reset on apply)` |
| apply-pipeline: No drift after clean apply | pass | `go test -run TestStatusCleanAfterApply -v` → PASS. Smoke: `status` → `No drift.` |
| apply-pipeline: Config edit is pending, not drift | pass | `go test -run TestStatusConfigEditIsPendingNotDrift -v` → PASS (`len(drift)==0 && pending==1`). Smoke: edit only `homonto.toml` → `1 config change(s) awaiting apply`, zero drift lines |
| apply-pipeline: Deleted managed key reported missing | pass | `go test -run TestStatusReportsMissingManagedKey -v` → PASS. Smoke: empty `mcpServers` → `claude mcp.cg missing (deleted out of band)` |
| tool-adapters: Claude adopts a pre-existing MCP | pass | `go test -run TestAdoptRecordsStateWithoutWritingFile` + `TestClaudeAdoptOnlyApplyLeavesFileByteIdentical` → PASS. Skeptic drove a hand-written comment-bearing `.claude.json` → byte-identical, pruneable |
| tool-adapters: OpenCode adopts a pre-existing setting AND plugin | pass | `go test -run 'TestOpenCodeAdopt(Setting|Plugin)RecordsState'` + `TestOpenCodeAdoptOnlyApplyLeavesFileByteIdentical` → PASS. Skeptic drove hand-authored `opencode.jsonc` with `//` comments → `Reconciled 2`, comments intact, both pruneable |

## Design conformance

Walked each `design.md` key decision against the implementation:

- **`adopt` as a first-class silent action** — present in `Change.Action`; Plan
  emits it for non-secret `disk==desired` keys where `!(inState && Applied==
  hash(disk))`; `plan.Render` renders no line; `plan.HasChanges` is visible-only
  (create/update/delete). `apply.go` three-way flow reconciles adoption-only
  runs without a prompt. Conforms.
- **adapter Apply records adopt state without a file write** — `st.Set` on the
  adopt branch, `continue` before writes; conditional `WriteAtomic`
  (`mjChanged`/`sjChanged`/`docChanged`) means adopt/noop-only apply writes no
  tool file. Conforms (byte-identical proven on comment-bearing files).
- **Drift decoupled from Plan via `ObserveHashes`** — `engine.Status` derives
  drift solely from `ObserveHashes`-vs-`Entry.Applied`; pending = Plan visible
  changes not in the drifted set. Only hashes leave the adapter. Conforms.
- **Secret keys never adopted** — `!secret.ContainsRef(want)` gates the branch;
  both skeptics confirmed a secret-matching key goes through `update`, no
  plaintext leak. Conforms.
- **Adapter parity** — claude and opencode mirrored; the one intentional
  difference (opencode plugins are array membership) is documented and matches
  how plugin `Applied` is stored.

## Adversarial pass

Two parallel fresh-context skeptics (full mode), prompted to refute. Round 1.

- **Conformance skeptic:** could not refute any of the 10 scenarios. Re-read
  every test's assertions and drove the real binary with hand-authored,
  comment-bearing, oddly-formatted tool files the unit tests never exercise.
  Confirmed adopt stores `hash(canonical(disk))` (not `hash(desired)`), no
  stray `!= "noop"` miscount, no secret adoption/leak. One **minor/by-design**:
  opencode plugin drift is a blind spot (present plugin can't be reported
  drifted) — explicitly documented in `design.md`; no scenario claims plugin
  drift. Not a violation. No CRITICAL/major.
- **Robustness skeptic:** could not break any category across 9 scratch
  environments — churn/convergence (unicode, floats, big ints > 2^53, reordered
  nested objects, empty containers, dotted keys), secret safety, mixed
  create+adopt+delete, partial apply (adopt state persisted, exit non-zero),
  skill.* (skills-only apply writes no JSON, repoint→drift, delete→missing),
  drift/pending accounting (no double-count, no cross-adapter contamination),
  error recovery (unparseable files warn + isolate, apply exits non-zero),
  idempotency. One **minor/cosmetic**: when BOTH tool files are unparseable,
  `status` prints warnings then `No drift.` and exits 0 — consistent with the
  documented "status keeps exit 0 with warnings" contract. No CRITICAL/major.

No scenario refuted → no failure gate. `metrics.verify_rounds` = 1.

## Regression

Fresh, this round:

- `go build ./...` → Success
- `go vet ./...` → No issues found
- `go test ./...` → 125 passed in 15 packages
- `go test -race ./...` → 125 passed in 15 packages
- `gofmt -l internal/` → (empty — all formatted)

## Deviations

Two accepted known-limitations (both minor, neither a scenario violation; each
raised by a skeptic and accepted here as out of this change's scope):

1. **OpenCode plugin drift blind spot** — a plugin still present in the
   `plugin` array can never be reported as drifted (plugins are array
   membership with no scalar to re-hash). Documented in `design.md`; removal is
   still detected as `missing`. Accepted: no scenario claims plugin value
   drift, and opencode has no per-plugin value to drift.
2. **`status` exit code when every adapter file is unparseable** — prints the
   warnings then `No drift.` and exits 0, matching the pre-existing
   "status/plan keep exit 0 with warnings" contract (only `apply` exits
   non-zero on skipped adapters). Accepted: consistent with existing behavior;
   changing exit-code semantics is out of scope for this change.
