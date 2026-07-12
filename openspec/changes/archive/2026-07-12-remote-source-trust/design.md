## Context

homonto resolves managed resources from `builtin:<name>` (embedded catalog) and
`local:<name>` (in-repo directory) sources. Both are trusted and offline. Adding
remote sources means resolving content that an attacker may control or tamper
with in transit or at rest on a registry. The current apply pipeline
(`config.Load` → adapters → `fsutil.WriteAtomic`) has no fetch, verification, or
provenance layer. Item 9 removed the old `agentlock`/`agentblob` content store,
so there is no existing content-addressed store to reuse. This design adds a
self-contained `internal/remote/` trust boundary whose single guarantee is:
**no target file is mutated until the fetched content is pinned-verified and
passes every structural safety check.**

## Goals / Non-Goals

**Goals:**
- One remote-source model usable by every resource kind (skill, command,
  subagent, framework) with the same pin + verify contract.
- Content-hash pinning recorded in a lockfile; reproducible and auditable.
- Fail closed: malformed, oversized, tampered, traversing, symlinked, or revoked
  content aborts before any disk mutation.
- Offline + cacheable: a pinned, cached resource applies with no network.
- Rollback, revocation, and removal paths defined and tested.
- A malicious-fixture suite that proves each control.

**Non-Goals:**
- Automatic remote updates (pin advancement is a manual config edit).
- A hosted registry or signing/keyserver PKI (digest pinning is the trust root
  for the first increment; provenance is recorded, not cryptographically
  attested beyond the content digest).
- Preserving remote archive file modes beyond what materialization needs.

## Decisions

### 1. Source syntax and pinning
`source = "remote:<url>"` with a sibling **required** `digest = "sha256:<hex>"`
on the resource. Rationale: a separate field keeps the locator readable, maps
cleanly to the lockfile, and makes "missing pin" a distinct, catchable
validation error. `config.Load` rejects a `remote:` source whose `digest` is
absent, malformed, or a non-sha256 algorithm. The digest is over the **canonical
extracted tree** (a deterministic tar re-serialization: sorted paths, normalized
modes, no timestamps) so the same content pins identically regardless of
transport or archive framing.

### 2. Transports
`internal/remote/fetch.go` exposes `Transport` implementations selected by URL
scheme:
- `https://…(.tar.gz|.tgz)` → download with `http.Client` (redirect cap = 5,
  overall timeout, `io.LimitReader` ceiling).
- `git+https://…` or `git://…#<ref>` → shallow fetch of a pinned ref into a temp
  worktree.
- `file://…` → local path (offline/testing; still runs full verification).
Every transport writes to an isolated temp dir and returns a raw byte stream or
tree; none touches the cache or target.

### 3. Verify-before-mutate pipeline
`internal/remote/verify.go` runs, in order, and aborts on the first failure:
1. **Size ceiling** during download (`LimitReader`); exceed → abort.
2. **Archive validation** (`extract.go`) streaming entries: reject absolute
   paths, reject any `..` component, reject symlinks/hardlinks/devices, cap
   per-entry size, cap total uncompressed size, cap entry count (tar/zip bomb).
3. **Canonicalize** the validated tree and **compute sha256**.
4. **Pin match**: computed digest must equal the declared pin → else abort
   (tamper / substitution defense).
5. **Revocation check**: digest not in `.homonto/revoked.json` → else abort.
Only after (5) does the content move into the cache. This ordering means a
tampered or bomb payload is rejected during validation before hashing cost, and
a substituted-but-well-formed payload is rejected at the pin match.

### 4. Content-addressed cache + offline
`internal/remote/cache.go` stores verified trees at
`.homonto/cache/remote/sha256/<digest>/`. Resolution checks the cache first; a
hit skips fetch entirely (offline + reproducible). Writes are atomic (temp dir +
rename). The cache is the single materialization source — adapters never read
from the temp fetch area.

### 5. Remote lockfile
`internal/remote/lock.go` maintains `.homonto/remote.lock.json`: a map of
resource identity → `{locator, transport, digest, size, fetchedAt-less
provenance}`. It is written after a successful verify+cache and read on apply to
confirm the pin is unchanged. (No wall-clock is stored to keep it reproducible
and diff-stable; provenance records transport + canonical locator + size.)

### 6. Rollback, revocation, removal, GC
- **Rollback**: because installs are digest-addressed and cached, reverting the
  `digest` in config re-resolves the prior content from cache with no network.
- **Revocation**: `.homonto/revoked.json` is a user/operator-maintained list of
  banned digests; any resolve of a revoked digest fails closed.
- **Removal**: de-declaring a remote resource prunes its materialized install
  through the existing managed-prune path and drops its lock entry.
- **GC**: a cache GC reclaims `cache/remote/sha256/<d>` entries no lock entry
  references (content-addressed, `--dry-run` preview), mirroring the item-9 GC
  shape.

### 7. Integration seam
A `remote.Resolver` is injected where `builtin:`/`local:` resolution happens in
the catalog/materialize path. For `remote:` sources it returns a local cache
path (after the full pipeline); for others it is a no-op. This keeps adapters
unaware of transport and preserves the existing plan/apply/state contract.

## Risks / Trade-offs

- **Digest over canonical tree vs raw archive.** Canonicalizing lets the same
  content pin identically across `.tar.gz` vs git, but adds a deterministic
  re-serialization step; we accept the cost for stable, transport-independent
  pins and document the canonical form so pins are reproducible by third parties.
- **No signature/PKI in increment 1.** Digest pinning defeats substitution once
  a pin exists, but the *first* pin is trust-on-first-use unless the user
  obtains the digest out of band. Documented as a known boundary; a signing
  layer is future work (item 11 provenance).
- **git transport surface.** Shelling to `git` inherits git's own fetch
  behavior; we constrain it to a pinned ref, shallow depth, and a temp worktree,
  and still run the same archive-validation + pin-match on the checked-out tree.
- **Redirect/DNS rebinding.** Redirect cap + final-content digest match bound
  the exposure; we do not pin IPs. Acceptable because the digest, not the host,
  is the trust root.
