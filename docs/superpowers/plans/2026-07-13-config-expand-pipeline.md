---
change: config-expand-pipeline
design-doc: docs/superpowers/specs/2026-07-13-config-expand-pipeline-design.md
base-ref: 7fd9a3aaec5349263843ee559b6145db1a6f4942
---
# Plan
1. Extract expandEntriesForTool generic; 3 wrappers (behavior-preserving, suite is the gate).
2. Verify: go test ./... -race, vet, build, openspec validate --all.
