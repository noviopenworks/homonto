# onto-binary (delta)

## ADDED Requirements

### Requirement: onto ships a full-lifecycle conformance suite

onto SHALL carry a full-lifecycle conformance test suite that drives the real CLI
through `new → set → advance → close` and asserts the binary gates REJECT bad
work, not only that the happy path succeeds. The suite SHALL cover at least:
advancing without a required phase artifact is refused; an invalid `--workflow`
is refused with no change created; a gated-field setter with an out-of-shape value
writes nothing; and `status`/`doctor` classify a malformed or missing state rather
than silently dropping it. This substitutes for the dogfooding feedback loop onto
forgoes (the project builds with Comet and ships onto — see `docs/personas.md`).

#### Scenario: the conformance suite proves gates reject bad work

- **GIVEN** the onto conformance suite
- **WHEN** it runs against the onto CLI
- **THEN** it passes only if each gate rejects its bad-work case (missing artifact, invalid workflow, out-of-shape enum, malformed/missing state) and the happy-path lifecycle still succeeds
