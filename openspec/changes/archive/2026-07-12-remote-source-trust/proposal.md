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
