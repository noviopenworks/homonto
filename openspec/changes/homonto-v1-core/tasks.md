# Tasks — homonto-v1-core

TDD throughout: write the failing test first, then minimal implementation, commit
per task. Detailed per-step guidance is refined in the build phase from
`docs/superpowers/plans/2026-06-24-homonto.md`, adjusted for the hashed-state
idempotency model (⚑ marks deltas from that plan).

## 1. Foundation
- [x] 1.1 Scaffold module + `version` command (`go.mod`, `main.go`, `internal/cli/root.go`, `.gitignore`)
- [ ] 1.2 Config model + TOML loader (`internal/config`) — MCPs, Skills, Plugins, Settings, `TargetsOrAll`
- [ ] 1.3 Secret resolver (`internal/secret`) — `${pass:…}` + `${ENV}`, `Resolve`, `ContainsRef`
- [ ] 1.4 ⚑ Hash helper — `sha256` of a resolved value (in `internal/secret` or `internal/state`)

## 2. State + merge primitives
- [ ] 2.1 ⚑ State store with hashed entries (`internal/state`) — `Entry{Desired, Applied}`; `Set(tool,key,desired,appliedHash)`, `Get`, atomic `Save`/`Load`
- [ ] 2.2 Surgical JSON/JSONC merge (`internal/jsonutil`) — `SetJSON`, `GetJSON`, `Standardize`, `EnsureArrayElem`
- [ ] 2.3 Content linker (`internal/link`) — idempotent symlink + conflict detection (never clobber)

## 3. Adapters
- [ ] 3.1 Adapter interface + `Change`/`ChangeSet` + plan printer (`internal/adapter`, `internal/plan`) — `+`/`~`, hide noops, never resolve secrets
- [ ] 3.2 ⚑ Claude adapter (`internal/adapter/claude`) — MCP/settings/plugins surgical projection; state-aware noop for secret keys; **redact `Change.Old` for secret-bearing keys**; store `{desired, sha256(resolved)}` on apply
- [ ] 3.3 ⚑ OpenCode adapter (`internal/adapter/opencode`) + Claude skill linking — JSONC merge, plugin array append, same hashed-state + redaction rules
- [ ] 3.4 ⚑ Secret-safety tests — `plan` output **and** `state.json` never contain a resolved secret, including on **drift of a secret-backed key**

## 4. Engine + CLI
- [ ] 4.1 Engine + `plan`/`apply` (`internal/engine`, `internal/cli`) — two-phase (resolve all, abort before any write), confirm `[y/N]`/`--yes`, save state last
- [ ] 4.2 `status` (drift) + `doctor` (`pass` on PATH, tool dirs, owned-skill presence)
- [ ] 4.3 `init` scaffold (never overwrite existing files)
- [ ] 4.4 `import` — bootstrap `homonto.toml` from existing setup with **secret redaction** to `${pass:…}`; `--force` guard

## 5. Verification
- [ ] 5.1 ⚑ End-to-end test: `init`→edit→`plan`→`apply` projects into both tools + symlinks; **second apply is a no-op including a secret-backed MCP**
- [ ] 5.2 Two-phase abort test — missing secret ref → no file written, missing ref named
- [ ] 5.3 Golden-file surgical-merge tests — unmanaged keys survive in all target files
- [ ] 5.4 README (quickstart, secret-reference syntax, JSONC comment caveat, symlinked content)
- [ ] 5.5 Full suite green: `go test ./... && go vet ./... && go build ./...`
