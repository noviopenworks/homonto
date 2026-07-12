# Remote source trust

homonto can install resources from **remote** sources, not only `builtin:`
(compiled-in) and `local:` (in-repo) ones. Because remote content is untrusted,
every remote install is **pinned, verified, and fail-closed**: nothing is
written to your tools until the fetched content matches its declared pin and
passes every safety check.

## Declaring a remote source

```toml
[subagents.reviewer]
source = "remote:https://example.com/reviewer.tar.gz"   # remote:<url>
digest = "sha256:<64 hex>"                               # REQUIRED content pin
scope  = "project"
targets = ["claude", "opencode"]
```

- The `digest` is **required**. A `remote:` source without a valid
  `sha256:<hex>` pin is a load-time error.
- Supported transports: `https://…(.tar.gz|.tgz)`, `git+https://…#<commit>`,
  and `file://…` (for local mirrors / offline). Plain `http://` is rejected.
- A remote **subagent** archive must contain `<name>.md` at its root.

## What homonto guarantees

Resolution runs a fixed **verify-before-mutate** pipeline. Each step aborts
before any cache or target write:

1. **cache lookup** — a pin already in `.homonto/cache/remote/` resolves with no
   network (offline, reproducible);
2. **fetch** — bounded by a redirect cap, timeout, and size ceiling;
3. **archive validation** — reject absolute paths, `..` traversal, symlinks /
   hardlinks / devices, and enforce per-entry / total / entry-count caps;
4. **canonical digest** — a transport-independent sha256 over a deterministic
   tree serialization;
5. **pin match** — the computed digest must equal the declared pin;
6. **revocation** — the digest must not be in `.homonto/revoked.json`.

Provenance is recorded in `.homonto/remote.lock.json` (locator, transport,
digest, size — no timestamps, so it is diff-stable).

## Threat model → control → test

| Attack class | Enforced control | Test |
|---|---|---|
| Path traversal (`../escape`, absolute) | rejected during extraction | `TestValidateTarFailsClosed` |
| Symlink / hardlink / device escape | non-regular members rejected | `TestValidateTarFailsClosed`, `TestFetchFileRejectsSymlinkInDir` |
| Archive / decompression bomb | per-entry, total, entry-count caps; gzip bounded while streaming | `TestValidateTarFailsClosed`, `TestValidateTarGzBombBoundedByTotal`, `TestFetchHTTPSSizeCapped` |
| Tampered payload / dependency substitution | canonical digest must equal the pin | `TestResolvePinMismatchFailsClosed`, `TestRemoteSubagentPinMismatchAbortsApply` |
| Compromised registry serving different bytes | same pin match (host is not the trust root) | `TestResolvePinMismatchFailsClosed` |
| Redirect swap / redirect loop | redirect cap + final-content pin match | `TestFetchHTTPSRedirectCapped` |
| Revoked content (even if cached) | revocation checked on both fetch and cache-hit paths | `TestResolveRevokedEvenWhenCached`, `TestRemoteSubagentRollbackAndRevocation` |
| Moved git tag/branch | pin governs trust, not the ref | (git checkout re-validated) `TestFetchGit` |

## Lifecycle

- **Offline / reproducible** — a cached pin applies with no network.
- **Rollback** — revert the `digest` in config; the prior content resolves from
  cache (kept until an explicit GC).
- **Revocation** — add a `sha256:…` to `.homonto/revoked.json`; the next resolve
  of that digest fails closed, even from a warm cache.
- **Removal** — de-declare the resource; `apply` prunes its install and drops
  its lockfile entry.
- **Cache GC** — an explicit maintenance step reclaims cache entries no lock
  entry references (kept out of `apply` so a revert can still roll back).

## Known boundary

The first pin is trust-on-first-use unless you obtain the digest out of band. A
signing/provenance-attestation layer is future work; the content digest is the
trust root today. Automatic remote updates are a deliberate **non-goal** —
homonto never re-resolves a `remote:` source to a newer digest than your pin.
