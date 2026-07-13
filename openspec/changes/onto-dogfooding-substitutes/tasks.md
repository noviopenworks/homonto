# Tasks — onto-dogfooding-substitutes
## 1. F21 persona / selection doc
- [ ] Write docs/personas.md: homonto=product, onto=native binary-enforced workflow,
      Comet/OpenSpec/Superpowers=unenforced alternatives, build-with-Comet/ship-onto,
      who onto is for and why we don't use it ourselves. Link from README.
## 2. onto lifecycle E2E / conformance suite
- [ ] A test that drives the real onto binary through new→advance→set→close and
      asserts gates REJECT bad work: advance without required artifacts fails;
      invalid --workflow rejected; malformed/missing state classified; bad enum
      rejected; happy-path lifecycle succeeds.
## 3. Verification
- [ ] `go test ./...` (incl. the new E2E) -race, vet, build, `openspec validate --all` green.
