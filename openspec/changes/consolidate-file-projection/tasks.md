# Tasks — consolidate-file-projection

## 1. fileproj contract
- [x] Add `internal/adapter/fileproj` (Link, Project, Conflicts, ApplyState,
      ApplyLinks, Observe + recordedDst + " -> " constant). Table-driven unit
      tests; green in isolation.

## 2. claude migration (skills canary → commands → subagents)
- [ ] claude skills via fileproj (narrow inline adopt/delete loop to
      command./subagent.). scope/adopt/observehashes/pruning/conformance green.
- [ ] claude commands via fileproj. Suites green.
- [ ] claude subagents via fileproj; delete now-empty inline loop + dead
      recordedDst. Suites green.

## 3. opencode migration (same sequence)
- [ ] opencode skills → commands → subagents via fileproj; drop dead
      recordedDst. opencode + conformance suites green.

## 4. Verify + scope confirm
- [ ] Copy-mode (subagentcopy.*) and internal/link untouched; generic delete
      loop unchanged (fileproj plans no deletes).
- [ ] `go test ./... -race`, vet, build, `openspec validate --all` green;
      byte/behavior identical (conformance + per-adapter link tests).
