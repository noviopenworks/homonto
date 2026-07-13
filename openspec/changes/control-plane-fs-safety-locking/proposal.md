## Why

Three control-plane safety gaps (ROADMAP N6, gate B; T-hostile engine):
- **F25:** `fsutil.WriteAtomic` resolves symlinks (`fsutil.go:19` `EvalSymlinks`)
  and writes THROUGH them — unsafe for `.homonto` state/cache/lock/catalog, where
  a planted symlink could redirect a control-plane write outside the project.
- **F29:** no cross-process lock — two concurrent `apply`s plan from the same
  snapshot and last-writer-win the state/tool files; apply also races cache GC.
- **F31:** remote locators with embedded credentials (userinfo / query tokens)
  appear verbatim in errors and `remote.lock.json`, leaking secrets to logs/files.

## What Changes

- A **no-follow, root-confined** write path for control-plane files under
  `.homonto`: refuse to write through a symlink; confine the resolved path under
  the project's `.homonto` root. Tool-config writes (which legitimately follow
  symlinks) keep the existing `WriteAtomic`.
- A **project-scoped apply lock** (`.homonto/apply.lock`, `O_EXCL`/flock) so a
  second concurrent `apply` fails fast rather than racing; released on exit.
- **Redact credentials** from remote locators before they reach errors or
  `remote.lock.json`: reject embedded userinfo/token, or store a redacted
  canonical URL.

## Impact

- **Code:** `internal/fsutil/` (no-follow writer), the `.homonto` write callers
  (state/cache/lock/catalog), `internal/cli/apply.go` (lock), `internal/remote/`
  (locator redaction) + tests.
- **Spec:** `apply-pipeline` (lock + no-follow control-plane writes),
  `remote-source-trust` (locator redaction) deltas.
- **Out of scope:** N5 (remote transactionality/staging).
