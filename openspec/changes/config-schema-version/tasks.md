# Tasks — config-schema-version

## 1. Versioned config with fail-closed load
- [ ] Add Config.SchemaVersion (`toml:"schema_version,omitempty"`) +
      CurrentConfigSchemaVersion=1; config.Load rejects a future version
      fail-closed (absent/0 = current). TDD: a future version errors; absent
      and current load fine.

## 2. Verify
- [ ] `go test ./... -race`, vet, build, `openspec validate --all` green.
