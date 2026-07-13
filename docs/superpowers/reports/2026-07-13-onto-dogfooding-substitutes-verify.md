# Verification Report — onto-dogfooding-substitutes

**Date:** 2026-07-13 · ROADMAP N7 (dogfooding substitutes) · Comet tweak · Result: PASS

- **F21 persona doc** — `docs/personas.md` (homonto=product, onto=native binary-enforced
  workflow, Comet/OpenSpec/Superpowers=unenforced alternatives, build-with-Comet/ship-onto,
  who onto is for). Linked from README.
- **onto conformance suite** — `internal/ontocli/conformance_test.go` (6 tests): happy-path
  lifecycle (new→set→advance×4→close/archive) + gate rejections (missing artifact blocks
  advance; invalid --workflow; out-of-shape enum/guides writes nothing; status/doctor classify
  malformed + missing-state per F14). No product-code change; no onto gate weakness found.
- Delta: onto-binary ADDs the conformance-suite requirement.

## Evidence
`go test ./internal/ontocli/... -race` → 70 passed (incl. 6 conformance); vet clean; build OK;
`openspec validate --all` 16/16.

## Note
Closes the N7 dogfooding-substitute obligation from the 2026-07-13 fork. Remaining gate-A
item: N2 (semantic gate content).
