# Remote framework resolution via the trust pipeline

## Why

Roadmap E1 (F36), the last framework-source kind. Local frameworks install
end-to-end (`local-frameworks`); the catalog can merge framework overlays
(`catalog-local-overlays`); and the `internal/remote` trust pipeline already
fetches/verifies/pins/revokes remote SUBAGENT content. This change combines them
so a `[frameworks.X] source="remote:<url>" digest="sha256:<hex>"` installs like a
local framework — fetched and verified through the SAME trust pipeline, then
merged as an overlay — completing "install a framework from anywhere: builtin,
local, or remote."

## Decision (the Plan-time gate, resolved)

A remote framework's resources are only known from its manifest (remote content),
so expansion needs it. Decision: **the engine resolves remote frameworks through
the trust pipeline at engine build (fetch → verify digest → cache), keyed by the
pinned digest**, and injects the verified cache directory as a framework overlay.
Resolution is content-addressed and cached, so `Plan`/`status` reuse the cache
(network only on a first or changed pin) and a dry run is accurate. Fail-closed:
a missing/mismatched digest or a revoked pin aborts before any catalog use — the
existing pipeline's guarantees, reused unchanged.

## What Changes

- **Config**: accept `[frameworks.X] source="remote:<url>"` with a required
  `digest`; a remote framework without a digest fails at load (as remote
  subagents do). Other non-builtin/non-local/non-remote sources still fail.
- **Engine**: resolve declared remote frameworks via the `remote.Resolver`
  (digest-pin, revocation, cache) into per-framework verified cache dirs, and
  inject them (name→dir) into the config so the framework catalog merges them as
  overlays — reusing `LoadWithLocal` and the FS-aware materialize.
- **Config catalog**: `FrameworkCatalog` merges local: paths AND the injected
  remote cache dirs as overlays; expansion handles `remote:` frameworks (their
  resources project as `builtin:<name>`, like local/builtin).

## Impact

- **Specs:** `framework-expansion` gains a requirement that a remote framework
  installs through the trust pipeline and the same validated overlay path.
- **Behavior:** builtin/local/remote-subagent behavior unchanged; new: a remote
  framework's resources install after digest-verified fetch.
- **Risk:** medium — reuses the tested trust pipeline (no new security code) and
  the local-framework overlay path; guarded by an E2E acceptance test (a
  digest-pinned remote framework's skill is materialized by apply; a wrong digest
  aborts) plus the full suite.

## Non-goals

- `[compat].homonto`, capabilities (later/decision-gated). Lockfile-cached
  manifest optimization (the content-addressed cache already makes re-resolution
  cheap).
