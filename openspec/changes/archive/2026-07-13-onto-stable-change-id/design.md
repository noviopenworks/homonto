# Design — onto stable change id

## Approach

`ontostate.State` gains `ID string` (`yaml:"id,omitempty" json:"id,omitempty"`).
`onto new` (`internal/ontocli/new.go` `runNew`) generates it via a `newID()` that
reads crypto/rand and hex-encodes 4 bytes (an 8-char hex id), setting it on the
State before Save. No other command writes `ID`; `Save` round-trips it, so
`set`/`advance`/`close` (which Load → mutate → Save) preserve it verbatim. `Load`
leaves an absent id empty and never mints one, so an id never changes meaning
across reads (backward-compatible with pre-feature states).

`onto state --json` already marshals the whole State, so the id surfaces via the
json tag; `onto status` prints it in its per-change summary.

## Why crypto/rand is fine here

The "no Date/random" rule constrains the comet workflow *scripts* (Math.random/
Date.now break resume). This is the onto Go binary — a normal program — where
crypto/rand for a one-time id is correct. Tests assert the id is present,
8 hex chars, unique across two changes, and unchanged by transitions — not a
fixed value.

## Risk

Low — additive immutable field + generation at creation. onto* command tests
pin generation/uniqueness/immutability.

## Alternatives
- A content/timestamp-derived id — rejected; not stable across a rename and not
  guaranteed unique. A random id assigned once is both.
