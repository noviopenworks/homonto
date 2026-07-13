# Tasks — remote-frameworks

## 1. Config: accept remote: frameworks (+ required digest)
- [ ] validateFrameworkResources accepts remote:<url> with a required digest
      (reuse remote source/digest parsing); other sources unchanged. Config
      carries injected remote framework dirs; FrameworkCatalog merges them +
      expansion handles remote: (as builtin:<name>).

## 2. Engine: resolve remote frameworks via the trust pipeline
- [ ] Resolve declared remote frameworks through remote.Resolver (fetch → verify
      digest → cache; revocation fail-closed) into per-framework cache dirs,
      injected into the config before catalog use (Plan + materialize). Reuses
      LoadWithLocal + FS-aware materialize.

## 3. E2E + verify
- [ ] E2E: a digest-pinned remote:file:// framework's skill is materialized by
      apply; a wrong digest aborts fail-closed. `go test ./... -race`, vet, build,
      `openspec validate --all` green.
