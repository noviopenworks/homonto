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
