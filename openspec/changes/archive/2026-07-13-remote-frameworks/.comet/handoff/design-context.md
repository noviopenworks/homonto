# Comet Design Handoff

- Change: remote-frameworks
- Phase: design
- Mode: compact
- Context hash: e6b7121c5b34d691d23dec52345118ab28548ef9039ad0106b108a5be8df6f2b

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/remote-frameworks/proposal.md

- Source: openspec/changes/remote-frameworks/proposal.md
- Lines: 1-53
- SHA256: e038302c54093d77909c195dd5dfd3f88defcf9d7b4cd4c0b2e5399ab9c2c311

```md
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

```

## openspec/changes/remote-frameworks/design.md

- Source: openspec/changes/remote-frameworks/design.md
- Lines: 1-52
- SHA256: 0fdc68d439fb19ca2210e4cc823d173d6400e0cefeb01573c2ad3e36c68de463

```md
# Design — remote framework resolution

## Reuse, don't reimplement

Remote frameworks = the remote trust pipeline (fetch/verify/digest-pin/
revocation/cache — `internal/remote`, already used for subagents) + the
local-framework overlay path (`LoadWithLocal`/`mergeFrameworkRoot`, FS-aware
materialize — just shipped). The only new code is the WIRING.

## Config

- `validateFrameworkResources`: allow `remote.IsRemoteSource(src)` for a
  framework, requiring a valid `digest` (reuse `remote.ParseRemoteSource` +
  `remote.ParseDigest`); mirror the remote-subagent rules. `builtin:`/`local:`
  unchanged; every other source still fails.
- `Config` gains an unexported `remoteFrameworkDirs map[string]string`
  (name→verified cache dir), set by the engine (below). `FrameworkCatalog`
  merges local: paths AND these dirs via `cat.NewWithLocal` (each an
  `os.DirFS(dir)`). The three `Expanded*` treat a `remote:` framework like a
  local one — resolved by its config-key name, resources tagged `builtin:<name>`.

## Engine

Add `resolveRemoteFrameworks()` mirroring `materializeRemotes` (same
`remote.Resolver{Cache, Revocations, Limits}`, lock, revocation quarantine
fail-closed): for each `[frameworks.X] source="remote:<url>" digest=D`, parse the
pin, revocation-check (fail closed), `Resolver.Resolve(src, pin)` → verified Tree
in the digest-addressed cache, and record the cache dir for X. Call it in
`engine.Build` (after `config.Load`) and set `cfg.remoteFrameworkDirs`, so both
`Plan` (expansion) and `materializeCatalog` see the overlays. Content-addressed
cache ⇒ re-resolution is a cache hit (no refetch); a config with no remote
frameworks does nothing (unchanged).

## Security (reused, verified in review)

Digest is verified by `remote.Resolver.Resolve` BEFORE the cache dir is exposed;
a mismatch or a revoked pin returns an error that aborts `Build` (fail-closed, no
catalog use). No new crypto/verify code — the tested pipeline's guarantees hold.

## Acceptance test

Build a framework tar.gz (framework.toml name=myfw + skills/myskill/SKILL.md),
`pin = CanonicalDigest(fetch(...))`; config
`[frameworks.myfw] source="remote:file://<tar>" digest="<pin>" scope="user"` →
`Build`→`Plan`→`Apply` materializes `.homonto/catalog/skills/myskill/SKILL.md`.
A WRONG digest → `Build`/apply aborts, nothing installed.

## Risk

Medium — wiring two tested subsystems. Mitigations: no new security code; the
E2E acceptance (good + wrong digest) drives the real path; builtin/local/
remote-subagent behavior unchanged (full suite green).

```

## openspec/changes/remote-frameworks/tasks.md

- Source: openspec/changes/remote-frameworks/tasks.md
- Lines: 1-18
- SHA256: c446bfc60a756fe82e9327b132910cc225458af2337cd04540bda4e1956a39af

```md
# Tasks — remote-frameworks

## 1. Config: accept remote: frameworks (+ required digest)
- [ ] validateFrameworkResources accepts remote:<url> with a required digest
      (reuse remote source/digest parsing); other sources unchanged. Config
      carries injected remote framework dirs; FrameworkCatalog merges them +
      expansion handles remote: (as builtin:<name>).

## 2. Engine: resolve remote frameworks via the trust pipeline
- [ ] Resolve declared remote frameworks through remote.Resolver (fetch → verify
      digest → cache; revocation fail-closed) into per-framework cache dirs,
      injected into the config before catalog use (Plan + materialize). Reuses
      LoadWithLocal + FS-aware materialize.

## 3. E2E + verify
- [ ] E2E: a digest-pinned remote:file:// framework's skill is materialized by
      apply; a wrong digest aborts fail-closed. `go test ./... -race`, vet, build,
      `openspec validate --all` green.

```

## openspec/changes/remote-frameworks/specs/framework-expansion/spec.md

- Source: openspec/changes/remote-frameworks/specs/framework-expansion/spec.md
- Lines: 1-26
- SHA256: 0f04a4f70eb18e8c9289b50b85d1e1debbdf4dd291eef8312971bfc1cdb2d576

```md
# framework-expansion

## ADDED Requirements

### Requirement: A remote framework installs through the trust pipeline

Config loading SHALL accept a framework whose source is `remote:<url>` with a
required `digest` pin, and homonto SHALL resolve it through the same remote trust
pipeline as remote subagents — fetching, verifying the content against the
pinned digest, honoring revocation, and caching by digest — before merging the
verified content as a framework overlay and installing its resources through the
same validated path as a builtin or local framework. A remote framework without
a digest, or whose fetched content does not match the pin, or whose pin is
revoked, MUST fail closed with no installation. Resolution MUST be
content-addressed and cached so re-resolution needs no refetch.

#### Scenario: A digest-pinned remote framework installs

- **GIVEN** a config with `[frameworks.X] source="remote:<url>" digest="sha256:<hex>"` whose content matches the pin
- **WHEN** the change is applied
- **THEN** the framework is fetched, verified, and its resources are installed exactly as a local framework's would be

#### Scenario: A mismatched digest aborts fail-closed

- **WHEN** a remote framework's fetched content does not match its pinned digest
- **THEN** resolution fails closed and nothing is installed

```
