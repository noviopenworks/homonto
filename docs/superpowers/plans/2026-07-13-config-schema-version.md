---
change: config-schema-version
design-doc: docs/superpowers/specs/2026-07-13-config-schema-version-design.md
base-ref: d184369dfe0b61b5909e43e40231539c9fcd1c25
---
# Plan
1. Config.SchemaVersion + CurrentConfigSchemaVersion + Load fail-closed (TDD). Commit.
2. Verify: go test ./... -race, vet, build, openspec validate --all. Commit.
