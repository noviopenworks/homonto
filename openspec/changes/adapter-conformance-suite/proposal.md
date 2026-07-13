## Why
ROADMAP E3 / finding F55 (first slice): each adapter has ad-hoc per-adapter tests,
but there is no single REUSABLE conformance suite that every adapter must pass, so
a new adapter (or a regression in an existing one) can silently diverge from the
contract. Start a shared, table-driven conformance suite over the `Adapter`
interface (Name/Plan/Apply/ObserveHashes).
## What Changes
- A shared conformance harness asserting, for each registered adapter (claude,
  opencode), the core contract: Plan on a fresh config yields creates; Apply
  writes them; ObserveHashes then reports every applied key clean (unchanged);
  a second Plan is a no-op (idempotent); an unmanaged file in the target tree is
  preserved across apply.
## Impact
- **Code:** a new `internal/adapter/conformance` test harness (test-only).
- **Spec:** `tool-adapters` delta (adapters pass a shared conformance suite).
- **Out of scope (F55 remainder):** adoption, drift-reset, secret redaction,
  malformed-doc, conflict-safety, and the codex adapter's reduced surface — added
  incrementally; and the Claude/OpenCode consolidation onto structproj.
