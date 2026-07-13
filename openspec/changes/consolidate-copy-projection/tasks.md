# Tasks — consolidate-copy-projection

## 1. copyproj core
- [x] Add `internal/adapter/copyproj` (Name, Plan, Apply + internal
      recordedCopyHashes + keyPrefix "subagentcopy."). Table-driven tests
      (create/update/prune/local-edit/conflict/prune-root-refusal). Green.

## 2. claude migration
- [ ] Route claude copy-mode through copyproj; keep copySubagentDesired +
      copyPruneRoots; Plan emit uses copyproj.Name. claude + conformance green.

## 3. opencode migration
- [ ] Route opencode copy-mode through copyproj (same). opencode + conformance
      green.

## 4. Verify
- [ ] internal/copyfile untouched; F7 prune-root guard + local-edit backup
      preserved. `go test ./... -race`, vet, build, `openspec validate --all`.
