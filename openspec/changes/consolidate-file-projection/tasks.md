# Tasks — consolidate-file-projection

## 1. fileproj contract
- [x] Add `internal/adapter/fileproj` (Link, Project, Conflicts, ApplyState,
      ApplyLinks, Observe + recordedDst + " -> " constant). Table-driven unit
      tests; green in isolation.

## 2. claude migration (skills canary → commands → subagents)
- [x] claude skills via fileproj (narrow inline adopt/delete loop to
      command./subagent.). scope/adopt/observehashes/pruning/conformance green.
- [x] claude commands via fileproj. Suites green.
- [x] claude subagents via fileproj; delete now-empty inline loop + dead
      recordedDst. Suites green.

## 3. opencode migration (same sequence)
- [x] opencode skills → commands → subagents via fileproj; drop dead
      recordedDst. opencode + conformance suites green.

## 4. Verify + scope confirm
- [x] Copy-mode (subagentcopy.*) and internal/link untouched; generic delete
      loop unchanged (fileproj plans no deletes).
- [x] `go test ./... -race`, vet, build, `openspec validate --all` green;
      byte/behavior identical (conformance + per-adapter link tests).
