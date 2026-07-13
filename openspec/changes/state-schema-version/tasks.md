# Tasks — state-schema-version
## 1. State schema version
- [ ] Add SchemaVersion + CurrentStateSchemaVersion; Save stamps it; Load rejects a
      future version (>current) and tolerates absent/0. Round-trip + future-reject tests.
## 2. Verify
- [ ] go test ./internal/state/... ./internal/engine/... -race, vet, build, validate --all green.
