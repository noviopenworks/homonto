# Tasks — local-frameworks

## 1. Catalog: FS-aware index + single-framework merge
- [x] Track per-resource source FS (skillFS/commandFS/subagentFS); Materialize*
      + SubagentContent resolve from it (base-only identical). Add
      mergeFrameworkRoot + LoadWithLocal(base, locals). Tests: base identity;
      a local single-framework merges + materializes from its FS.

## 2. Config: local: acceptance + overlay catalog + expansion
- [x] Accept local:<path> frameworks (keep F35 for other non-builtin); build the
      catalog with the config's local overlays (thread baseDir, replace the
      loadedCatalog singleton); expand local frameworks as builtin:<name>.

## 3. Engine wiring + E2E
- [x] materializeCatalog builds the catalog with the config's local overlays.
      E2E: a local: framework's skill is materialized by apply; builtin path
      unchanged.

## 4. Verify
- [x] `go test ./... -race`, vet, build, `openspec validate --all` green.
