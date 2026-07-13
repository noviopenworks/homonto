## Why

The 2026-07-13 fork chose "build with Comet, ship onto," which means onto never
gets the dogfooding feedback loop that made the projector 8/10 — exactly why onto
scored 2/10. Two substitutes are non-optional (ROADMAP N7): a full-lifecycle onto
E2E/conformance suite (so gate regressions are caught without a human living in
onto), and an F21 persona/selection doc (so users aren't confused that the
maintainers don't use what they ship). With N1 now shipped (binary authoritative,
skills shell out), both are buildable against a real command surface.

## What Changes

- **F21 persona/selection doc** (`docs/personas.md` or similar): homonto is the
  product; onto is its native binary-enforced workflow; Comet/OpenSpec/Superpowers
  are unenforced alternatives; we build with Comet and ship onto — who onto is for
  and why. Referenced from the README/roadmap.
- **onto lifecycle E2E/conformance suite**: drive `onto new → advance → set … →
  close` (and the doctor/status classification) end-to-end and assert the binary
  gates actually reject bad work — advancing without required artifacts, an invalid
  workflow, a malformed/missing state, a bad enum — not just the happy path.

## Impact

- **Docs:** a new persona doc + README/roadmap pointer.
- **Test:** a new onto lifecycle E2E test (Go test or a scripted `onto` driver).
- **No product-code behavior change** (docs + tests only).
- **Out of scope:** N2 (semantic gate content — the E2E asserts the *structural*
  B1 gates that exist today, not future semantic gates).
