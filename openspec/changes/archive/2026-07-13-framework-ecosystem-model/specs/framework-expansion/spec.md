# framework-expansion

## ADDED Requirements

### Requirement: The framework model supports versioned manifests and validated custom-source resolution

The framework ecosystem SHALL support versioned framework manifests and a single
validated resolution path that a builtin, a fourth builtin, or a trusted custom
framework all pass through. A framework manifest MAY declare a manifest schema
version, provided/required capabilities, and compatibility ranges; loading MUST
reject a manifest whose schema version exceeds what the binary supports (fail
closed), and MUST reject an incompatible framework or an unresolved required
capability with a clear error rather than silently installing nothing. The
existing guarantees — transitive dependency resolution, cycle detection, and
duplicate-resource rejection — MUST be preserved.

This requirement is recorded as the design target for roadmap E1; the design is
delivered and reviewed before implementation, which lands in later phased changes.

#### Scenario: A manifest from a newer schema is rejected

- **WHEN** a framework manifest declares a manifest schema version greater than
  the binary supports
- **THEN** loading fails closed with an "upgrade homonto" error and installs
  nothing

#### Scenario: A custom framework resolves through the same validated path

- **WHEN** a trusted custom framework is resolved
- **THEN** it is loaded and validated through the same manifest/dependency/
  path checks as a builtin framework, and an unsupported source or an
  incompatible version fails loudly
