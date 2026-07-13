# framework-expansion

## ADDED Requirements

### Requirement: Framework capability requirements are resolved fail-loud

Catalog loading SHALL resolve every framework capability requirement against the
capabilities provided across all frameworks, where a capability is a `name@major`
string (a name plus a non-negative integer major). A framework MAY declare the
capabilities it provides and the capabilities it requires; a required capability
with no provider (exact `name@major` match) among the loaded frameworks MUST fail
the load with an error naming the framework and the capability. A malformed
capability string MUST fail loud. Multiple providers of one capability are
permitted (a capability is an interface, not a uniquely-owned resource), and a
framework with no capability declarations MUST behave exactly as before.

#### Scenario: An unresolved capability requirement fails to load

- **WHEN** a framework requires a capability that no loaded framework provides
- **THEN** loading fails with an error naming the framework and the capability

#### Scenario: A satisfied capability requirement loads

- **WHEN** a required capability is provided by some loaded framework (including
  one merged from an overlay)
- **THEN** the catalog loads and expansion behaves as before
