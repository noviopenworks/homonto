# Tasks — consolidate-structured-doc-projection

## 1. Shared JSON codec
- [x] Add a `structproj.Codec` backed by `internal/jsonutil` (EnsureRoot→
      ObjectRoot, Get→GetJSON, Set→SetJSON, Delete→DeleteJSON, Canonical→
      Canonical), shared by claude + opencode. TDD: codec unit test round-trips
      get/set/delete/canonical and normalizes an empty doc.

## 2. claude structured-doc migration
- [x] Route `setting.*` (settings.json) through structproj.Project/Apply/
      Observe; delete the bespoke branch. claude + conformance suites green.
- [x] Route `.claude.json` prefixes (mcp/plugin/pluginconfig/marketplace)
      through structproj; delete those branches. Suites green.

## 3. opencode structured-doc migration
- [x] Route `opencode.json` prefixes (mcp/setting) through structproj; delete
      the bespoke loop. opencode + conformance suites green.

## 4. Confirm scope + verify
- [x] File-projection paths (skills/commands/subagents symlinks, inactive
      dirs, copy-subagents) untouched in both adapters.
- [x] `go test ./... -race`, `go vet`, `go build`, `openspec validate --all`
      green; plan/apply/observe output byte-identical (conformance suite).
