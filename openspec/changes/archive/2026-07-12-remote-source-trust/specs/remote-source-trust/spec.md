## ADDED Requirements

### Requirement: Remote source declaration and mandatory pinning

A remote source SHALL be declared as a `remote:` URL and SHALL carry a `digest`
pin of the form `sha256:` followed by 64 hex characters. The config loader SHALL
reject a `remote:` source whose digest is absent, malformed, or names an
algorithm other than sha256, failing closed at load before any resolution.

#### Scenario: Pinned remote source loads

- **GIVEN** a resource with `source = "remote:https://example.test/x.tar.gz"` and `digest = "sha256:<64 hex>"`
- **WHEN** the config loads
- **THEN** the resource is accepted with its pinned digest recorded

#### Scenario: Unpinned remote source is rejected

- **GIVEN** a resource with `source = "remote:https://example.test/x.tar.gz"` and no `digest`
- **WHEN** the config loads
- **THEN** loading fails with a missing-digest error and no resolution occurs

#### Scenario: Malformed digest is rejected

- **GIVEN** a remote source with `digest = "sha256:not-hex"` or a non-sha256 algorithm
- **WHEN** the config loads
- **THEN** loading fails closed with a digest-format error

### Requirement: Verify-before-mutate resolution

Resolving a remote source SHALL fetch content into an isolated temporary area,
validate the archive, canonicalize it, compute its sha256, compare that digest
to the declared pin, and check the digest against the revocation list — in that
order — aborting on the first failure. No file outside the temporary area and no
target file SHALL be created or modified until every check passes.

#### Scenario: Digest mismatch aborts before any mutation

- **GIVEN** fetched content whose canonical digest differs from the declared pin
- **WHEN** resolution runs
- **THEN** resolution fails closed and no cache entry or target file is written

#### Scenario: Revoked digest fails closed

- **GIVEN** a pin whose digest appears in `.homonto/revoked.json`
- **WHEN** resolution runs
- **THEN** resolution fails closed with a revoked-digest error

#### Scenario: Successful verification materializes from cache

- **GIVEN** fetched content whose canonical digest equals the pin and is not revoked
- **WHEN** resolution runs
- **THEN** the content is stored in the content-addressed cache and materialized from there

### Requirement: Fail-closed archive validation

Archive extraction SHALL reject absolute member paths, any `..` traversal
component, and non-regular members (symlinks, hardlinks, devices), and SHALL
enforce a per-entry size cap, a total-uncompressed-size cap, and an entry-count
cap. Any violation SHALL abort extraction before writing the offending member.

#### Scenario: Path traversal is rejected

- **GIVEN** an archive member named `../escape`
- **WHEN** extraction runs
- **THEN** extraction aborts and nothing is written outside the temp dir

#### Scenario: Symlink member is rejected

- **GIVEN** an archive containing a symlink member
- **WHEN** extraction runs
- **THEN** extraction aborts fail-closed

#### Scenario: Oversized or too-many-entry archive is rejected

- **GIVEN** an archive exceeding the total-size or entry-count cap (a bomb)
- **WHEN** extraction runs
- **THEN** extraction aborts before exhausting resources

### Requirement: Content-addressed cache and offline resolution

Verified content SHALL be stored under
`.homonto/cache/remote/sha256/<digest>/` written atomically. Resolution SHALL
check the cache before fetching; a cache hit SHALL resolve with no network
access, so a pinned resource applies offline and reproducibly.

#### Scenario: Cached pin resolves offline

- **GIVEN** a pin whose content is already in the cache
- **WHEN** resolution runs with the network transport unavailable
- **THEN** resolution succeeds from the cache

#### Scenario: Same content yields the same cache path

- **GIVEN** two resolutions of identical content
- **WHEN** each is cached
- **THEN** both map to the same `sha256/<digest>` cache path

### Requirement: Remote lockfile records provenance

A remote install SHALL be recorded in `.homonto/remote.lock.json` with its
canonical locator, transport, digest, and byte size. The lockfile SHALL contain
no wall-clock timestamps so consecutive writes of unchanged state are
byte-stable. Apply SHALL read the lockfile to confirm a pin is unchanged.

#### Scenario: Lock entry written after verification

- **GIVEN** a remote resource resolved and verified
- **WHEN** apply completes
- **THEN** `remote.lock.json` has an entry with the locator, transport, digest, and size

#### Scenario: Lockfile is diff-stable

- **GIVEN** an unchanged remote configuration
- **WHEN** apply runs twice
- **THEN** `remote.lock.json` is byte-identical between runs

### Requirement: Rollback, removal, and cache GC

Reverting a resource's `digest` SHALL re-resolve the prior content from cache
with no network. De-declaring a remote resource SHALL prune its materialized
install and drop its lockfile entry. A cache GC SHALL reclaim only cache entries
no lockfile entry references and SHALL support a `--dry-run` preview.

#### Scenario: Reverting a pin rolls back from cache

- **GIVEN** a resource updated to a new pin, then reverted to a previously cached pin
- **WHEN** apply runs
- **THEN** the prior content resolves from cache with no network access

#### Scenario: De-declared remote resource is pruned

- **GIVEN** a previously applied remote resource removed from config
- **WHEN** apply runs
- **THEN** its materialized install is pruned and its lockfile entry removed

#### Scenario: GC reclaims only unreferenced content

- **GIVEN** a cache containing a digest no lockfile entry references
- **WHEN** cache GC runs
- **THEN** only the unreferenced digest is reclaimed; referenced digests remain
