## Why
ROADMAP E3 / F55 (slice 3, completes the conformance core): extend the shared
adapter conformance suite with the remaining contract properties — secret
non-resolution/redaction and conflict-safe non-clobber of foreign content — so the
suite covers the full behavioral contract uniformly.
## What Changes
- The conformance suite gains, for claude + opencode: (a) secret non-resolution —
  a `${pass:...}`/`${ENV}` secret reference in config is NOT resolved into a hash or
  left as plaintext on disk in a way that leaks (the raw secret never escapes via
  ObserveHashes); (b) foreign-content safety — a managed KEY whose on-disk value was
  written by something else (not owned in state) is not silently clobbered/adopted
  without going through the normal plan (conservative ownership).
## Impact
- **Code:** `internal/adapter/conformance/conformance_test.go` (extend, test-only).
- **Spec:** `tool-adapters` delta (conformance covers secret non-resolution + foreign-content safety).
- **Out of scope:** codex adapter conformance; the Claude/OpenCode structproj consolidation.
