---
comet_change: remote-frameworks
role: technical-design
canonical_spec: openspec
status: draft
archived-with: 2026-07-13-remote-frameworks
status: final
---

# remote-frameworks — Technical Design (E1)

OpenSpec is canonical; full model in the change's `design.md`. Remote frameworks
= the shipped `internal/remote` trust pipeline (fetch/verify/digest-pin/
revocation/cache) + the local-framework overlay path (`LoadWithLocal`, FS-aware
materialize). Only the WIRING is new.

## Decision (Plan-time gate resolved)

The engine resolves remote frameworks through the trust pipeline at `Build`
(fetch→verify digest→cache, revocation fail-closed), injecting each verified
cache dir into the config as a framework overlay. Content-addressed cache ⇒
`Plan`/`status` reuse it (network only on a first/changed pin); fail-closed on
missing/mismatched/revoked pin before any catalog use.

## Risk posture

Medium — wiring two TESTED subsystems, no new security code (digest verify +
revocation are the reused pipeline's). E2E acceptance (good + wrong digest)
drives the path; builtin/local/remote-subagent behavior unchanged.

## Out of scope

`[compat].homonto`, capabilities; lockfile-cached-manifest optimization.
