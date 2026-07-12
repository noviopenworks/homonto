---
change: remote-source-trust
design-doc: docs/superpowers/specs/2026-07-12-remote-source-trust-design.md
base-ref: f1c8b4273b20c39970873b0583099fa4df69b6ea
---

# Implementation Plan — Remote Source Trust

Executes the Design Doc. TDD: for each task, add a failing test first (the
malicious-fixture and boundary tests are the spec), then implement to green. One
commit per task group; run `go test ./internal/remote/...` (and `./...` at
integration steps) before each check-off.

## Task 1 — Digest type + config parsing (tasks.md §1)

- `internal/remote/digest.go`: `Digest{Algo,Hex}`, `ParseDigest("sha256:<hex>")`
  (reject bad algo, non-64, non-hex), `String()`, `Equal`.
- `internal/remote/locator.go`: `ParseRemoteSource("remote:<url>")` → URL +
  transport kind; reject non-`remote:` here (caller guards).
- `internal/config`: add `Digest string` to the resource/subagent types; in
  `validate*`, when a source has the `remote:` prefix require a parseable digest,
  else error. Keep `builtin:`/`local:` unaffected.
- Tests: `digest_test.go` table (valid/blank/short/long/upper/non-hex/other-algo)
  + a fuzz seed `FuzzParseDigest`; config test for remote-without-digest error.

## Task 2 — Safe extraction + canonical digest (tasks.md §2)

- `internal/remote/extract.go`: `ValidateTarGz(r io.Reader) (Tree, error)` and
  `ValidateTar`. Stream entries; reject absolute/`..`/non-regular; enforce
  `maxEntries`, `maxEntryBytes`, `maxTotalBytes`; gzip via LimitReader.
- `internal/remote/canonical.go`: `Tree` (map path→bytes+mode), `Digest(Tree)`
  → sha256 over canonical stream (sorted paths, mode&0o755, len, bytes).
- Tests: `extract_test.go` malicious table (traversal, symlink, hardlink,
  device, per-entry over, total over, entry-count over, dup path) each →
  fail-closed, nothing written; `canonical_test.go` determinism + empty tree.

## Task 3 — Transports (tasks.md §3)

- `internal/remote/fetch.go`: `Transport` iface `Fetch(ctx, url) (Tree, size,
  error)`; `selectTransport(url)` by scheme.
- `https`: `http.Client{Timeout, CheckRedirect cap 5}`, body via LimitReader,
  https-only; pipe through `ValidateTarGz`.
- `file://`: read local `.tar.gz` or tar a directory in-memory → validate.
- `git`: `git -c protocol.file.allow=user clone --depth 1` pinned ref to temp,
  canonicalize worktree minus `.git`.
- Tests: file:// happy path; https via `httptest` (redirect cap, size cap);
  unknown scheme error. (git test guarded by `git` presence.)

## Task 4 — Verify pipeline + pin + revocation (tasks.md §4)

- `internal/remote/revoke.go`: load `.homonto/revoked.json` (array), `Contains`.
- `internal/remote/verify.go`: `Resolve(src, cache, revoke)` implementing the
  step order (cache→fetch→validate→canonical→pin→revoke), abort-first.
- Tests: pin-mismatch (tamper), revoked digest, redirect-swap fixture → all
  fail-closed with no cache write.

## Task 5 — Content-addressed cache + offline (tasks.md §5)

- `internal/remote/cache.go`: `Cache{Root}`, `Has(Digest)`, `Dir(Digest)`,
  `Put(Digest, Tree)` atomic (temp+rename), `GC(referenced, dryRun)`.
- Tests: put/has/dir; offline resolve (transport that panics if called) on a
  warm cache; same content → same path; GC reclaims only unreferenced.

## Task 6 — Remote lockfile (tasks.md §6)

- `internal/remote/lock.go`: `Lock` map keyed `kind/name` →
  `{Locator,Transport,Digest,Size}`; `Load`/`Save` (sorted, no timestamps,
  atomic via fsutil.WriteAtomic).
- Tests: round-trip; diff-stable (two saves byte-identical); pin-drift detect.

## Task 7 — Resolver integration (tasks.md §7)

- `internal/remote/resolver.go`: `Resolver{Root}` with
  `ResolveDir(source, digest) (dir, error)` — remote:→pipeline→cache dir;
  others → `("", ErrNotRemote)` so callers fall through.
- Wire into `internal/catalog/materialize` (or the entries resolution) + engine
  so a `remote:` subagent/skill/command materializes from the cache dir. Write
  lock entries after verify. Plan/apply/status/doctor honor it.
- Tests: unit resolver; E2E in Task 7.3.

## Task 8 — Rollback/revocation/removal/GC end-to-end (tasks.md §8)

- E2E (`internal/engine` or `internal/remote` integration test) using a
  `file://` fixture as `[subagents.x] source="remote:..." digest=...`:
  plan → apply → status idempotent (offline) → revert pin (rollback from cache)
  → de-declare → prune + lock entry dropped → GC reclaims.

## Task 9 — Threat model, ADR, docs, gate (tasks.md §9)

- ADR `docs/adr/0013-remote-source-trust-boundary.md` (pin-not-auto-update).
- `docs/guides/` threat-model note: attack class → control → test name.
- Update README (remote source form), `docs/roadmap.md` item 10 → done w/
  evidence. Delta specs already written; sync happens at archive.
- Gate: `go test -race ./...`, fuzz seeds, `./scripts/gate.sh` (docker may be
  skipped locally if unavailable — record any gap).

## Verification

Each task: `go test ./internal/remote/... -race`. Integration tasks (7,8):
`go test ./... -race`. Final: full gate + fuzz seeds. Fail-closed assertions
must check BOTH an error is returned AND no file/cache entry was created.
