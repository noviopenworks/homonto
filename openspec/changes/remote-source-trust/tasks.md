## 1. Config: remote source syntax + pinning

- [x] 1.1 Add `Digest` field to the resource/subagent config types; parse
  `source = "remote:<url>"` and `digest = "sha256:<hex>"`.
- [x] 1.2 Validate at load: a `remote:` source requires a well-formed
  `sha256:<64-hex>` digest; missing/malformed/other-algorithm → load error.
  Add table-driven tests + a fuzz seed for the digest parser.

## 2. Safe extraction + archive validation (fail-closed core)

- [x] 2.1 `internal/remote/extract.go`: streaming tar(.gz) validator/extractor
  rejecting absolute paths, `..` traversal, symlinks/hardlinks/devices; caps on
  per-entry size, total uncompressed size, and entry count.
- [x] 2.2 Canonical-tree serialization + sha256 digest (sorted paths, normalized
  modes, no timestamps) — transport-independent, reproducible.
- [x] 2.3 Malicious-fixture tests: traversal, escaping symlink, size bomb, entry
  bomb, non-regular entries — each fails closed with no files written outside
  the temp dir.

## 3. Transports

- [x] 3.1 `Transport` interface + selection by URL scheme.
- [x] 3.2 `https` tarball transport: redirect cap, timeout, `LimitReader` size
  ceiling; writes only to an isolated temp dir.
- [x] 3.3 `file://` transport (offline/testing) through the same validation.
- [x] 3.4 `git` pinned-ref transport (shallow, temp worktree) validated by the
  same archive/canonical pipeline.

## 4. Verify pipeline + pin match + revocation

- [x] 4.1 `internal/remote/verify.go`: run size → validate → canonicalize →
  digest → pin-match → revocation, aborting on first failure before any cache
  or target write.
- [x] 4.2 Pin-mismatch (tamper/substitution) and revoked-digest fixtures fail
  closed; redirect-swap fixture (final content differs from pin) fails closed.

## 5. Content-addressed cache + offline

- [x] 5.1 `internal/remote/cache.go`: atomic store at
  `.homonto/cache/remote/sha256/<digest>/`; resolve checks cache first.
- [x] 5.2 Offline test: a cached pin resolves with the network transport
  disabled; reproducibility test: same content → same cache path.

## 6. Remote lockfile + provenance

- [x] 6.1 `internal/remote/lock.go`: read/write `.homonto/remote.lock.json`
  (locator, transport, digest, size, provenance); no wall-clock (diff-stable).
- [x] 6.2 Lock is written after verify+cache and read on apply to confirm the
  pin is unchanged; tampered-lock/pin-drift tests.

## 7. Resolver integration into apply/materialize

- [x] 7.1 `remote.Resolver` seam: `remote:` resolves through the pipeline to a
  cache path; `builtin:`/`local:` unchanged.
- [x] 7.2 Wire into catalog/materialize + engine so a `remote:` skill/command/
  subagent projects like any managed resource; plan/apply/status/doctor honor it.
- [x] 7.3 End-to-end test: declare a `remote:` (file:// fixture) resource →
  plan → apply → status idempotent → de-declare → prune.

## 8. Rollback, revocation, removal, GC

- [x] 8.1 Rollback test: change the pin then revert → prior content resolves
  from cache, no network.
- [x] 8.2 Revocation surface (`.homonto/revoked.json`) honored across resolve.
- [x] 8.3 Cache GC reclaims only unreferenced digests (`--dry-run` preview);
  removal path prunes install + drops lock entry.

## 9. Threat model, ADR, docs, gate

- [ ] 9.1 ADR: remote trust boundary + pin-not-auto-update decision.
- [ ] 9.2 Threat-model note mapping each attack class → enforced control → test.
- [ ] 9.3 Update README + guide + `docs/roadmap.md` (item 10 status with
  evidence); delta specs for `config-model` and `apply-pipeline`.
- [ ] 9.4 Full gate green: `go test -race ./...`, fuzz seeds, `./scripts/gate.sh`.
