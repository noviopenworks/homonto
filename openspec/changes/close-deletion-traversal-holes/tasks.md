# Tasks — close-deletion-traversal-holes

## 1. F28 — local: plain-name validation for skills/commands
- [ ] `validateResources` rejects a `local:` source that is not a plain name
      (reuse the `validateSubagents` check: reject ``, `.`, `..`, `/`, `\`,
      non-`filepath.Base`). Test: a `local:../x` skill and command are rejected;
      a plain `local:x` passes.

## 2. F7 — confine copy-mode prune to the managed root
- [ ] Prune SHALL refuse to delete a destination that resolves outside the
      managed provider root (reconstruct/validate the destination from resource
      identity, not trust the recorded path blindly). Test: a tampered state
      entry pointing outside the root is NOT deleted (prune skips/errors).

## 3. Verification
- [ ] `go test ./internal/config/... ./internal/copyfile/... ./internal/adapter/... -race`, vet, build, `openspec validate --all` green.
