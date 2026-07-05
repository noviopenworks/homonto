# Design: address-deep-review

Status: Confirmed
Confirmed: 2026-07-04 (review-prescribed fixes; gate pre-answered by
user directive — see notes.md)

## Summary

Implement the deep review's fix list (items 1–4, 6, 7 fully; item 5
except pushing to origin). The review document contains the
alternatives analysis; this design records the binding technical
decisions per fix. Rejected alternative: rewriting the adapter layer
around a schema-registry abstraction — over-engineering for two tools;
fix in place, add conformance fixtures.

## Goals / Non-Goals

**Goals:** correct Claude MCP schema; non-destructive import; no secret
leaks in any state condition; true pruning; injection-safe key
handling; deterministic plans; distributable repo (LICENSE/CI/version);
honest README; onto v2.1 ceremony fixes. **Non-goals:** pushing to
origin (user's standing choice); new adapters; multi-machine state;
project-scoped config; chezmoi parity debates.

## Architecture

### Claude MCP schema (P0)

`claude.desired()` emits per MCP:
`{"type":"stdio","command":cmd[0],"args":cmd[1:],"env":{...}}` (args
omitted when empty, env omitted when empty — match `claude mcp add`
output exactly). `current()` unchanged (reads whatever is on disk;
comparison is canonical JSON so shape mismatch = update → old configs
self-heal on next apply). Conformance test: fixture copied from a real
`~/.claude.json` shape asserting byte-level key structure, plus a
fixture-driven test that `desired()` output round-trips into a file
Claude Code would accept (`type` present, `command` is a string).

### Import (P0)

Read `command` as string + `args` array (real schema); tolerate the
legacy all-in-`command` array by splitting head/tail; preserve into
`MCP.Command = [command, args...]`. Import `env` with expanded
redaction: prefixes `sk-`, `ghp_`, `github_pat_`, `xox`, `glpat-`,
`npm_`, `AIza`, `Bearer `; key patterns `*_KEY`, `*_TOKEN`, `*_SECRET`,
`*_PASSWORD`, `*_CREDENTIALS`, `DATABASE_URL`. Read failures other than
absence produce a warning, not silence.

### Secret safety

- `planKey`/claude Plan: when a key is **not in state** and disk
  differs from desired, `Old = adapter.SecretRedaction` — unknown
  provenance is treated as secret (safe default; the diff still shows
  the incoming value's token form).
- `writeAtomic` (deduplicated into `internal/jsonutil` or a new
  `internal/fsutil`): stat existing file for mode; default `0600` for
  new files; write temp with that mode; `f.Sync()` before close;
  `os.CreateTemp` in the target dir for unique names; rename.
- Resolver memoizes by token (map guarded — single-threaded CLI, plain
  map fine): one `pass` invocation per distinct token per run.

### Pruning (CRITICAL)

- New action `"delete"` in `adapter.Change`.
- Plan: after desired-key loop, iterate `state.Keys(tool)` — any state
  key with prefix `mcp.`/`setting.`/`plugin.`/`skill.` absent from
  desired → `Change{Action: "delete", Key: k, Old: <redacted-or-disk>}`.
- Apply: `delete` → sjson delete path (escaped) / plugin array removal /
  `os.Remove` of the symlink **only if** it is a symlink pointing into
  our content dir; then `state.Delete(tool, key)`.
- Skill links recorded in state at apply (`skill.<name>` → desired =
  target path, applied = hash of target) so pruning and drift both see
  them.
- Render: `- <key>` lines; `HasChanges` includes deletes; sorted
  output.

### Injection/traversal robustness

- `jsonutil.EscapePath(key)` → escapes `.`, `*`, `?`, `\` per sjson
  rules; used for every dynamic path segment in both adapters
  (read and write sides — gjson accepts the same escaping).
- `config.Load` validates skill names: must equal `filepath.Base(name)`,
  no separators, not `.`/`..`; violation = config error at load time.
- `readStandardized`: root must be a JSON object; anything else →
  explicit error naming the file.

### Determinism

Both adapters sort desired keys before emitting changes; delete keys
sorted; Render output therefore stable. Test: two consecutive Plans
render identically.

### Hygiene

MIT `LICENSE` (flagged for user override at close);
`.github/workflows/ci.yml` (go vet, go test on push/PR, Go 1.23);
`cli/root.go` `var Version = "0.1.0-dev"`; README: comment-loss
sentence rewritten to the truth (all comments in the file are
normalized away on write), owned content claim reduced to skills,
"declarative" sentence now backed by pruning, install section notes
repo status honestly.

### onto v2.1

- `onto-tweak`: scope extended to "small features — ≤5 files (tests
  excluded), no new capability, no existing-spec change"; upgrade rules
  unchanged otherwise. Dispatcher routing table + guide updated to
  match ("small feature → onto-tweak").
- Dispatcher preflight: rtk/graphify missing → **warn, record in
  notes.md Grounding, proceed** (halt text removed); graphify staleness
  wording kept.
- `onto-close` step 2 addition: after numbering ADRs, rewrite workspace
  references to `adr/<slug>.md` → `docs/adr/NNNN-<slug>.md` (design.md,
  notes.md) before the archive move, so archives ship no dangling refs.
- ADR 0007: correct the Decision sentence (skips recorded in the
  report's Adversarial section) with an errata line noting the
  2026-07-04 correction.
- Living spec onto-workflow.md + guide updated via delta (preflight
  requirement, preset scope, close rewrite).

## Key decisions

1. **Fix-in-place over schema-registry abstraction** — two adapters
   don't justify a registry; conformance fixtures carry the correctness
   burden. (No ADR — reversible implementation choice.)
2. **Unknown-provenance = redacted** — plan output loses old-value
   detail for unknown keys; safety beats diff aesthetics. (In delta
   spec, no ADR.)
3. **Preflight warns, never halts** (→ adr/preflight-warns-not-halts.md)
   — reverses part of ADR 0005's hard-requirement stance; deserves its
   own accepted record.

## Error handling

Partial apply: state saved per adapter (adapter A's writes recorded even
if B fails); apply exits non-zero with the failing adapter named.
Delete of a non-homonto symlink: conflict error, never removed. Import
read errors: warned, not swallowed.

## Testing strategy

TDD per bug: failing test reproducing the review finding first
(schema shape, import args-drop, missing-state leak, mode loosening,
dotted-key corruption, traversal, non-convergent plan, ordering
instability, orphaned key after config removal), then fix, then green.
Conformance fixtures from real tool output. Full suite + `go vet` in CI
and locally. onto v2.1 changes: lint-conformant delta + guide/table
sync checks (grep) + fresh-context skeptics at verify.

## Grounding

docs/reviews/2026-07-04-deep-review.md (all findings with file:line;
empirical reproductions; live ~/.claude.json schema check).
