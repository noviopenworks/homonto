# Tasks — adapter-conformance-codex
## 1. Codex in the conformance suite
- [x] Add codex to the shared conformance table; run the checks its MCP-only surface
      supports; explicitly t.Skip (with reason) those it doesn't. No weakening.
## 2. Verify
- [ ] go test ./internal/adapter/... -race, vet, build, openspec validate --all green.
