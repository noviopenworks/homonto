# framework-expansion

## ADDED Requirements

### Requirement: A remote framework installs through the trust pipeline

Config loading SHALL accept a framework whose source is `remote:<url>` with a
required `digest` pin, and homonto SHALL resolve it through the same remote trust
pipeline as remote subagents — fetching, verifying the content against the
pinned digest, honoring revocation, and caching by digest — before merging the
verified content as a framework overlay and installing its resources through the
same validated path as a builtin or local framework. A remote framework without
a digest, or whose fetched content does not match the pin, or whose pin is
revoked, MUST fail closed with no installation. Resolution MUST be
content-addressed and cached so re-resolution needs no refetch.

#### Scenario: A digest-pinned remote framework installs

- **GIVEN** a config with `[frameworks.X] source="remote:<url>" digest="sha256:<hex>"` whose content matches the pin
- **WHEN** the change is applied
- **THEN** the framework is fetched, verified, and its resources are installed exactly as a local framework's would be

#### Scenario: A mismatched digest aborts fail-closed

- **WHEN** a remote framework's fetched content does not match its pinned digest
- **THEN** resolution fails closed and nothing is installed
