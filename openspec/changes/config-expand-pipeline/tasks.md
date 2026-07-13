# Tasks — config-expand-pipeline

## 1. Extract the generic pipeline
- [ ] Extract expandEntriesForTool(tool, kind, base, expand) from the three
      Expanded* functions; they become thin wrappers (base + kind + catalog
      adapter). No behavior change. Config suite green unchanged.

## 2. Verify
- [ ] `go test ./... -race`, vet, build, `openspec validate --all` green.
