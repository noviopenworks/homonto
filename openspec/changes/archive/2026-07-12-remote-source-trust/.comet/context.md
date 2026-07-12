# Comet Design Handoff

- Change: remote-source-trust
- Phase: design
- Mode: compact
- Context hash: 30fb2d7dffe8714def303ae7979f6e6e26dc2f3e6ab3caf909fb03e67e8f1b81

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/remote-source-trust/proposal.md

- Source: openspec/changes/remote-source-trust/proposal.md
- Lines: 1-76
- SHA256: 2e7aeed35afdcc398969623c03dfb05eb305740b4756b47a503371303c940a1e

```md
## Why

Today every managed resource resolves from a `builtin:` (compiled-in catalog) or
`local:` (in-repo directory) source — both fully trusted and offline. Roadmap
item 10 requires accepting **remote** resources (frameworks, skills, subagents,
commands) without reusing those local-source trust assumptions against untrusted
input. Fetching remote content introduces redirects, path traversal, symlink
escapes, archive bombs, tampered payloads, compromised registries, and
dependency substitution — none of which the current pipeline defends against.
This change adds a remote-source trust boundary so a remote install is pinned,
verified, cacheable, reproducible, revocable, and removable, and any
malformed or tampered content fails **before** any file on disk is mutated.

## What Changes

- **New `remote:` source type.** Resources may declare `source =
  "remote:<url>"` with a **required** `digest = "sha256:<hex>"` pin. An unpinned
  remote source is a load-time error (fail closed).
- **Fetch transports.** Two verified transports: `https` archive
  (`.tar.gz`/`.tgz`) and `git` (pinned commit). A `file://` transport is
  supported for offline/testing. All transports enforce redirect caps,
  timeouts, and size ceilings.
- **Verify-before-mutate pipeline.** Fetch to a temp area → compute the content
  digest → compare to the pin (mismatch aborts) → validate the archive
  (reject `..` traversal, reject escaping symlinks, cap total size / per-entry
  size / entry count) → check the revocation list → only then extract into a
  content-addressed cache and materialize. No target file is touched until every
  check passes.
- **Content-addressed cache + offline.** Verified content is stored under
  `.homonto/cache/remote/sha256/<digest>/`. A cache hit needs no network, so a
  pinned resource applies offline and reproducibly.
- **Remote lockfile.** `.homonto/remote.lock.json` records, per remote resource,
  the resolved digest, transport, canonical locator, byte size, and fetch
  provenance — auditable and reproducible.
- **Rollback + revocation + removal.** A pin change re-resolves deterministically
  and keeps the prior cache entry until GC (rollback = restore the prior pin).
  A local revocation list (`.homonto/revoked.json`) fails a revoked digest
  closed. De-declaring a remote resource prunes its install like any other
  managed resource; a cache GC reclaims unreferenced content.
- **Threat model doc + ADR.** A recorded threat model enumerating each attack
  class and the enforced control, plus an ADR for the trust boundary and the
  pin-not-auto-update decision.
- **Malicious-fixture test suite.** Fixtures for traversal, escaping symlink,
  size/entry bomb, tampered payload (hash mismatch), redirect swap, and revoked
  digest — each asserted to fail closed with no disk mutation.

Non-goal: **automatic remote updates**. homonto never silently re-resolves a
`remote:` source to a newer version; advancing a pin is a manual config edit.

## Capabilities

### New Capabilities
- `remote-source-trust`: the remote-source declaration syntax, the pin +
  verify-before-mutate pipeline, transports, content-addressed cache/offline,
  the remote lockfile, revocation, rollback, removal/GC, and the fail-closed
  threat-model guarantees.

### Modified Capabilities
- `config-model`: adds the `remote:` source form and the required `digest`
  field, and the load-time rule that a remote source without a valid digest is
  rejected.
- `apply-pipeline`: resolution of a remote resource routes through the remote
  trust pipeline (cache/fetch/verify) before materialization, and remote
  installs/prunes are tracked in the remote lockfile.

## Impact

- New package `internal/remote/` (locator, fetch, verify, extract, cache, lock,
  revocation).
- `internal/config/` — parse and validate the `remote:` source + `digest`.
- Apply/materialize wiring in `internal/catalog/` + `internal/engine/`.
- New state artifacts under `.homonto/` (`remote.lock.json`, `cache/remote/`,
  `revoked.json`); documented in the roadmap and guides.
- New ADR (remote trust boundary) and `docs/` threat-model note.
- No change to `builtin:`/`local:` behavior; remote is strictly additive and
  opt-in.

```

## openspec/changes/remote-source-trust/design.md

- Source: openspec/changes/remote-source-trust/design.md
- Lines: 1-116
- SHA256: a58c3363c959753fb8894382e52e3c78069c49d3256a7deb6e112446d7ad04f0

[TRUNCATED]

```md
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

```

Full source: openspec/changes/remote-source-trust/design.md

## openspec/changes/remote-source-trust/tasks.md

- Source: openspec/changes/remote-source-trust/tasks.md
- Lines: 1-74
- SHA256: db87915e879a7f0378f12d2b74e3c2b286d658a14f3331777196643202c94710

```md
## 1. Config: remote source syntax + pinning

- [ ] 1.1 Add `Digest` field to the resource/subagent config types; parse
  `source = "remote:<url>"` and `digest = "sha256:<hex>"`.
- [ ] 1.2 Validate at load: a `remote:` source requires a well-formed
  `sha256:<64-hex>` digest; missing/malformed/other-algorithm → load error.
  Add table-driven tests + a fuzz seed for the digest parser.

## 2. Safe extraction + archive validation (fail-closed core)

- [ ] 2.1 `internal/remote/extract.go`: streaming tar(.gz) validator/extractor
  rejecting absolute paths, `..` traversal, symlinks/hardlinks/devices; caps on
  per-entry size, total uncompressed size, and entry count.
- [ ] 2.2 Canonical-tree serialization + sha256 digest (sorted paths, normalized
  modes, no timestamps) — transport-independent, reproducible.
- [ ] 2.3 Malicious-fixture tests: traversal, escaping symlink, size bomb, entry
  bomb, non-regular entries — each fails closed with no files written outside
  the temp dir.

## 3. Transports

- [ ] 3.1 `Transport` interface + selection by URL scheme.
- [ ] 3.2 `https` tarball transport: redirect cap, timeout, `LimitReader` size
  ceiling; writes only to an isolated temp dir.
- [ ] 3.3 `file://` transport (offline/testing) through the same validation.
- [ ] 3.4 `git` pinned-ref transport (shallow, temp worktree) validated by the
  same archive/canonical pipeline.

## 4. Verify pipeline + pin match + revocation

- [ ] 4.1 `internal/remote/verify.go`: run size → validate → canonicalize →
  digest → pin-match → revocation, aborting on first failure before any cache
  or target write.
- [ ] 4.2 Pin-mismatch (tamper/substitution) and revoked-digest fixtures fail
  closed; redirect-swap fixture (final content differs from pin) fails closed.

## 5. Content-addressed cache + offline

- [ ] 5.1 `internal/remote/cache.go`: atomic store at
  `.homonto/cache/remote/sha256/<digest>/`; resolve checks cache first.
- [ ] 5.2 Offline test: a cached pin resolves with the network transport
  disabled; reproducibility test: same content → same cache path.

## 6. Remote lockfile + provenance

- [ ] 6.1 `internal/remote/lock.go`: read/write `.homonto/remote.lock.json`
  (locator, transport, digest, size, provenance); no wall-clock (diff-stable).
- [ ] 6.2 Lock is written after verify+cache and read on apply to confirm the
  pin is unchanged; tampered-lock/pin-drift tests.

## 7. Resolver integration into apply/materialize

- [ ] 7.1 `remote.Resolver` seam: `remote:` resolves through the pipeline to a
  cache path; `builtin:`/`local:` unchanged.
- [ ] 7.2 Wire into catalog/materialize + engine so a `remote:` skill/command/
  subagent projects like any managed resource; plan/apply/status/doctor honor it.
- [ ] 7.3 End-to-end test: declare a `remote:` (file:// fixture) resource →
  plan → apply → status idempotent → de-declare → prune.

## 8. Rollback, revocation, removal, GC

- [ ] 8.1 Rollback test: change the pin then revert → prior content resolves
  from cache, no network.
- [ ] 8.2 Revocation surface (`.homonto/revoked.json`) honored across resolve.
- [ ] 8.3 Cache GC reclaims only unreferenced digests (`--dry-run` preview);
  removal path prunes install + drops lock entry.

## 9. Threat model, ADR, docs, gate

- [ ] 9.1 ADR: remote trust boundary + pin-not-auto-update decision.
- [ ] 9.2 Threat-model note mapping each attack class → enforced control → test.
- [ ] 9.3 Update README + guide + `docs/roadmap.md` (item 10 status with
  evidence); delta specs for `config-model` and `apply-pipeline`.
- [ ] 9.4 Full gate green: `go test -race ./...`, fuzz seeds, `./scripts/gate.sh`.

```

## openspec/changes/remote-source-trust/specs/apply-pipeline/spec.md

- Source: openspec/changes/remote-source-trust/specs/apply-pipeline/spec.md
- Lines: 1-28
- SHA256: 90cb890426d63222a27db5595e38f7b43cae0f0f8f078c82d213a5c7545568c1

```md
## ADDED Requirements

### Requirement: Remote resolution routes through the trust pipeline

When the apply pipeline resolves a resource whose source is `remote:`, it SHALL
route resolution through the remote trust pipeline (cache lookup → verified
fetch → validate → pin-match → revocation) and materialize only from the
content-addressed cache. `builtin:` and `local:` resolution SHALL be unchanged.
A remote resolution failure SHALL abort the apply before any target mutation,
consistent with the atomic-writes / state-last guarantee.

#### Scenario: Remote resource projects like a managed resource

- **GIVEN** a pinned, cacheable `remote:` subagent/skill/command
- **WHEN** plan then apply runs
- **THEN** it materializes into each target tool exactly like a builtin/local resource, and status/doctor track it

#### Scenario: Remote resolution failure aborts apply cleanly

- **GIVEN** a `remote:` resource whose content fails verification
- **WHEN** apply runs
- **THEN** the apply aborts before any target file is written and existing state is unchanged

#### Scenario: Idempotent remote apply

- **GIVEN** an already-applied pinned remote resource
- **WHEN** apply runs again
- **THEN** it is a no-op (cache hit, no network, no target rewrite)

```

## openspec/changes/remote-source-trust/specs/config-model/spec.md

- Source: openspec/changes/remote-source-trust/specs/config-model/spec.md
- Lines: 1-28
- SHA256: 9a5b3778bcfcda6fda8c46632ba2e7959f415f65a1e0d62b527d91d0db8435c1

```md
## ADDED Requirements

### Requirement: Remote source form with required digest

The config model SHALL accept a `remote:` URL source on any resource that
already accepts `builtin:` or `local:` sources, and SHALL require a sibling
`digest` field holding a sha256 pin. A `remote:` source without a valid sha256
digest SHALL be a load-time error. The `digest` field SHALL NOT affect
non-remote sources, preserving existing `builtin:` and `local:` behavior
unchanged.

#### Scenario: Remote source with valid digest parses

- **GIVEN** `[subagents.x]` with `source = "remote:https://h.test/x.tgz"` and `digest = "sha256:<64 hex>"`
- **WHEN** the config loads
- **THEN** the resource carries a remote source and the recorded pin

#### Scenario: Remote source without digest is rejected

- **GIVEN** a `remote:` source with no `digest`
- **WHEN** the config loads
- **THEN** loading fails with a clear missing-pin error

#### Scenario: Builtin and local sources are unaffected

- **GIVEN** existing `builtin:`/`local:` resources with no `digest`
- **WHEN** the config loads
- **THEN** they load exactly as before

```

## openspec/changes/remote-source-trust/specs/remote-source-trust/spec.md

- Source: openspec/changes/remote-source-trust/specs/remote-source-trust/spec.md
- Lines: 1-140
- SHA256: a1f1b9e7369a8e903b83b96e7e43711afb2c91b655167c3afae963968c1652af

[TRUNCATED]

```md
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


```

Full source: openspec/changes/remote-source-trust/specs/remote-source-trust/spec.md
