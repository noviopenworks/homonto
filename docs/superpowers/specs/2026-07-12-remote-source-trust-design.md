---
comet_change: remote-source-trust
role: technical-design
canonical_spec: openspec
archived-with: 2026-07-12-remote-source-trust
status: final
---

# Remote Source Trust — Technical Design

Deep technical refinement of `openspec/changes/remote-source-trust/design.md`.
The OpenSpec delta specs (`remote-source-trust`, `config-model`,
`apply-pipeline`) are the canonical behavior; this doc is the implementation
plan, test strategy, and edge-case ledger.

## 1. Package layout

```
internal/remote/
  locator.go     RemoteSource parse (remote:<url> + sha256 pin), transport detect
  digest.go      Digest type: parse "sha256:<hex>", format, equality
  extract.go     safe streaming tar(.gz) validator+extractor (fail-closed)
  canonical.go   canonical tree serialization + sha256 over canonical form
  fetch.go       Transport interface + https / file / git implementations
  verify.go      Resolve(): cache → fetch → validate → canonical digest → pin → revoke
  cache.go       content-addressed store under .homonto/cache/remote/sha256/<d>/
  lock.go        remote.lock.json read/write (diff-stable, no timestamps)
  revoke.go      .homonto/revoked.json load + membership
  resolver.go    Resolver seam used by catalog/materialize
  testdata/      malicious fixtures (traversal, symlink, bombs, tampered)
```

All new code lives in one package with narrow, individually testable units. No
existing package's public API changes except an additive resolver hook.

## 2. Data shapes

- `Digest{ Algo: "sha256", Hex: string }` — `Parse("sha256:<64hex>")`; anything
  else errors. `String()` → `"sha256:<hex>"`.
- `RemoteSource{ Raw, URL string, Transport TransportKind, Pin Digest }`.
- Lock entry: `{ "name": ..., "kind": "skill|command|subagent", "locator":
  "<url>", "transport": "https|git|file", "digest": "sha256:...", "size": N }`.
  Lock file is a stable-sorted JSON object keyed by `kind/name`. No timestamps.

## 3. Verify pipeline ordering (the invariant)

`Resolve(src RemoteSource) (cacheDir string, err error)`:
1. If `cache.Has(src.Pin)` → return cache dir (offline, no fetch). Still checks
   revocation (a pin can be revoked after caching).
2. `raw := transport.Fetch(ctx, src.URL)` into a fresh temp dir, bounded by
   `LimitReader(maxTotalBytes)` and an overall context timeout; https caps
   redirects at 5.
3. `tree := extract.Validate(raw)` — streams entries, enforcing every structural
   rule below; writes members only into the temp staging dir.
4. `got := canonical.Digest(tree)`.
5. `if got != src.Pin { return errPinMismatch }` — tamper / substitution.
6. `if revoke.Contains(got) { return errRevoked }`.
7. `cache.Put(got, tree)` atomically (temp dir + rename); return cache dir.

The **only** place a target/cache write happens is step 7, strictly after 1-6.
Steps 2-4 write solely inside a per-resolve temp dir that is removed on any
error. This is what makes malformed/tampered/bomb content fail closed.

## 4. Extraction rules (extract.go)

Reject, before writing the member:
- absolute path (`filepath.IsAbs` after cleaning) or any `..` component;
- `tar.TypeSymlink`/`TypeLink`/`TypeChar`/`TypeBlock`/`TypeFifo` (non-regular);
- member whose running total would exceed `maxEntryBytes` or `maxTotalBytes`;
- entry count beyond `maxEntries`.
gzip is read through `gzip.NewReader` wrapped in the same `LimitReader` so a
decompression bomb trips `maxTotalBytes` during streaming, not after.

Caps (constants, documented): `maxEntries = 10_000`, `maxEntryBytes = 64 MiB`,
`maxTotalBytes = 256 MiB`. Chosen well above real skill/agent bundles, well
below resource-exhaustion. Overridable later; hard-coded for increment 1.

## 5. Canonical digest (canonical.go)

Serialize the validated tree deterministically: walk paths in lexical order,
for each regular file emit `path\0mode&0o755\0len\0bytes`. sha256 over that
stream. This makes the pin independent of archive framing (tar vs git checkout)
and of mtimes/uids. The canonical form is documented so a third party can
reproduce a pin from the extracted tree.

## 6. Transports (fetch.go)

- `https`: `http.Client{ Timeout, CheckRedirect: cap 5 }`; body via
  `LimitReader`. Only `https` (not `http`) accepted for network fetch.
- `file://`: read a local `.tar.gz` (or a directory → tar it in-memory) through
  the identical validate/canonical path. Used by tests and offline mirrors.
- `git`: `git -c protocol.file.allow=user clone --depth 1` of a pinned ref into
  a temp worktree, then canonicalize the worktree tree (minus `.git`). The ref
  in the URL fragment must be a full commit sha or tag; the pin still governs
  trust, so a moved tag is caught by the digest.

Transport is selected by scheme; unknown scheme → error.

## 7. Resolver seam (resolver.go)

`catalog/materialize` currently maps `builtin:`/`local:` to a source dir.
Introduce `remote.Resolver{ Root string }` with
`ResolveDir(src string, pin Digest) (dir string, err error)`. The materialize
path calls the resolver for `remote:` sources and uses the returned cache dir as
the content root; `builtin:`/`local:` bypass it entirely. This keeps adapters
transport-agnostic and preserves plan/apply/state semantics — a remote resource
is just a managed resource whose content root is a verified cache dir.

## 8. Rollback / revocation / removal / GC

- Rollback: pins are content-addresses; reverting `digest` in config re-resolves
  from cache (step 1). No dedicated command needed.
- Revocation: `revoke.go` loads `.homonto/revoked.json` (array of
  `sha256:...`); checked at steps 1 and 6.
- Removal: de-declaring flows through the existing managed prune; the resolver's
  lock entry is dropped when apply rewrites the lock without it.
- GC: `cache.GC(referenced []Digest, dryRun bool)` removes
  `sha256/<d>` dirs not in `referenced`; surfaced behind the existing
  maintenance path with `--dry-run`.

## 9. Test strategy

- **Unit, per file**: digest parse (+ fuzz seed), extract rules (table of
  malicious tars), canonical determinism, cache put/has/GC, lock round-trip
  (diff-stable), revoke membership.
- **Malicious-fixture suite** (`testdata/`): traversal tar, symlink tar, size
  bomb (gzip), entry bomb, tampered payload (digest ≠ pin), revoked digest,
  redirect-swap (file fixture whose content ≠ pin) — each asserts a fail-closed
  error AND that no file exists outside the temp/staging dir and no cache entry
  was created.
- **Integration/E2E**: a `file://` fixture wired as a `[subagents.x]
  source="remote:..."` resource → plan → apply → status idempotent (offline) →
  revert pin (rollback) → de-declare → prune → GC. Reuses the engine test
  harness.
- **Gate**: `go test -race ./...`, fuzz seeds, `./scripts/gate.sh`.

## 10. Edge cases / decisions

- Empty archive → valid empty tree with a well-defined canonical digest (not an
  error); a pin can legitimately point at empty content.
- Duplicate member paths in a tar → reject (ambiguous canonicalization).
- gzip vs plain tar: sniff magic; both supported for `https`/`file`.
- A revoked-after-cache pin still fails (revocation checked on the cache-hit
  path), so revocation cannot be bypassed by a warm cache.
- Non-goal reaffirmed: no auto-update — the resolver never re-resolves a URL to
  a different digest than the config pin.
