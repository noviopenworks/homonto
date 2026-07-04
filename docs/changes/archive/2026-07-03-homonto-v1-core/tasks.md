# Tasks — homonto-v1-core

TDD throughout: write the failing test first, then minimal implementation, commit
per task. Detailed per-step guidance is refined in the build phase from
`docs/superpowers/plans/2026-06-24-homonto.md`, adjusted for the hashed-state
idempotency model (⚑ marks deltas from that plan).

## 1. Foundation
- [x] 1.1 Scaffold module + `version` command (`go.mod`, `main.go`, `internal/cli/root.go`, `.gitignore`)
- [x] 1.2 Config model + TOML loader (`internal/config`) — MCPs, Skills, Plugins, Settings, `TargetsOrAll`
- [x] 1.3 Secret resolver (`internal/secret`) — `${pass:…}` + `${ENV}`, `Resolve`, `ContainsRef`
- [x] 1.4 ⚑ Hash helper — `sha256` of a resolved value (in `internal/secret` or `internal/state`)

## 2. State + merge primitives
- [x] 2.1 ⚑ State store with hashed entries (`internal/state`) — `Entry{Desired, Applied}`; `Set(tool,key,desired,appliedHash)`, `Get`, atomic `Save`/`Load`
- [x] 2.2 Surgical JSON/JSONC merge (`internal/jsonutil`) — `SetJSON`, `GetJSON`, `Standardize`, `EnsureArrayElem`
- [x] 2.3 Content linker (`internal/link`) — idempotent symlink + conflict detection (never clobber)

## 3. Adapters
- [x] 3.1 Adapter interface + `Change`/`ChangeSet` + plan printer (`internal/adapter`, `internal/plan`) — `+`/`~`, hide noops, never resolve secrets
- [x] 3.2 ⚑ Claude adapter (`internal/adapter/claude`) — MCP/settings/plugins surgical projection; state-aware noop for secret keys; **redact `Change.Old` for secret-bearing keys**; store `{desired, sha256(resolved)}` on apply
- [x] 3.3 ⚑ OpenCode adapter (`internal/adapter/opencode`) + Claude skill linking — JSONC merge, plugin array append, same hashed-state + redaction rules
- [x] 3.4 ⚑ Secret-safety tests — `plan` output **and** `state.json` never contain a resolved secret, including on **drift of a secret-backed key**

## 4. Engine + CLI
- [x] 4.1 Engine + `plan`/`apply` (`internal/engine`, `internal/cli`) — two-phase (resolve all, abort before any write), confirm `[y/N]`/`--yes`, save state last
- [x] 4.2 `status` (drift) + `doctor` (`pass` on PATH, tool dirs, owned-skill presence)
- [x] 4.3 `init` scaffold (never overwrite existing files)
- [x] 4.4 `import` — bootstrap `homonto.toml` from existing setup with **secret redaction** to `${pass:…}`; `--force` guard

## 5. Verification
- [x] 5.1 ⚑ End-to-end test: `init`→edit→`plan`→`apply` projects into both tools + symlinks; **second apply is a no-op including a secret-backed MCP**
- [x] 5.2 Two-phase abort test (internal/engine/engine_test.go) — missing secret ref → no file written, missing ref named
- [x] 5.3 Golden-file surgical-merge tests (adapter unmanaged-key/comment assertions) — unmanaged keys survive in all target files
- [x] 5.4 README (quickstart, secret-reference syntax, JSONC comment caveat, symlinked content)
- [x] 5.5 Full suite green: `go test ./... && go vet ./... && go build ./...`

## Code review outcome (executing-plans review gate)

Reviewer: general-purpose subagent over base 02d9779..HEAD. Verdict: "With fixes".

Fixed before verify:
- [x] CRITICAL: secret resolution now operates on parsed JSON string leaves
  (`secret.ResolveJSON`), so a secret containing `"`/`\`/newline can no longer
  corrupt the tool file or inject sibling keys (regression tests added).
- [x] Adapters fault-isolate: an unparseable tool file is skipped with a warning
  (`engine.Warnings`); other tools still proceed (spec scenario now covered).
- [x] `doctor` checks each tool's config location (`~/.claude`, `~/.config/opencode`).
- [x] Secret→literal config edit redacts `Change.Old` (no on-disk secret in plan).
- [x] Link conflicts are checked before writing tool files (fail fast per adapter).

Accepted for v1 (non-critical), with rationale:
- `import` covers Claude `mcpServers` only (not OpenCode / settings / plugins).
  Rationale: v1 `import` is a best-effort bootstrap; secret redaction + `--force`
  guard are the safety-critical parts and are implemented/tested. Broader import
  is a documented follow-up.
- Cross-adapter partial apply (Claude applied, OpenCode then errors) leaves
  Claude keys on disk but unsaved in state until a fully successful apply.
  Rationale: state is intentionally written last for crash-safety; the next apply
  reconciles (re-plans the unsaved keys). No data loss; matches "fail safe".
- Minor: drift includes pending config edits; deleted managed keys re-plan as
  create; dotted key names aren't path-escaped; `contentDir` is CWD-relative.
  Rationale: edge cases outside v1's common-denominator scope; noted for follow-up.
