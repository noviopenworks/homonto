## Why

AI coding-tool configuration (MCP servers, skills, plugins, settings) is scattered
across per-tool files (`~/.claude.json`, `~/.claude/settings.json`,
`~/.config/opencode/opencode.jsonc`), edited by hand, and impossible to reproduce
or review. `homonto` v1 makes one declarative `homonto.toml` the single source of
truth and projects it into Claude Code and OpenCode through a terraform-style
plan/confirm/apply pipeline, with secrets referenced (never stored) and resolved
only at apply time. This is the foundation every post-v1 roadmap phase builds on,
so it must first prove safety, idempotency, drift detection, and surgical merge.

## What Changes

- New Go CLI `homonto` (module `github.com/noviopenworks/homonto`, Go 1.22+) with
  commands `init`, `import`, `plan`, `apply`, `status`, `doctor`.
- Parse `homonto.toml` into one tool-agnostic desired-state model (MCPs, owned
  skills, per-tool plugins, per-tool settings).
- Per-tool **adapters** (Claude Code, OpenCode) that `Read`/`Plan`/`Apply` via
  **surgical merge**: homonto writes only the keys it manages and preserves all
  unmanaged keys (and, where possible, JSONC comments) in each tool's file.
- **Reference-only secrets** (`${pass:…}`, `${ENV}`) resolved **after** confirm,
  **all-at-once before any write** (two-phase); an interrupted or under-resolved
  apply never leaves a half-written file (atomic temp+rename; state written last).
- Owned content (skills) linked into each tool via **symlinks**, with conflict
  detection (never clobber a non-managed file).
- **Secret-idempotency fix** (roadmap-required pre-implementation adjustment):
  state stores each managed key's *unresolved* desired value plus a *non-secret
  hash* (sha256) of the applied resolved value. A second `plan` on a secret-backed
  value is a no-op, drift of a secret value is still detected, and neither `plan`
  output nor `state.json` ever contains a plaintext secret. **BREAKING** vs. the
  original plan's naive unresolved-only state comparison.
- Local state at `<repo>/.homonto/state.json` (gitignored) for drift detection.

## Capabilities

### New Capabilities
- `config-model`: parsing `homonto.toml` into the tool-agnostic desired-state
  model, including target defaulting (an MCP with no `targets` applies to all).
- `apply-pipeline`: the six-stage plan → confirm → resolve → apply engine —
  two-phase secret handling, atomic writes, idempotent re-apply, and drift.
- `secret-references`: `${pass:…}`/`${ENV}` resolution timing and the hashed-state
  idempotency model that keeps `plan` output and `state.json` free of plaintext.
- `tool-adapters`: Claude Code and OpenCode projection of MCPs/settings/plugins via
  surgical JSON/JSONC merge, plus symlinked owned content with conflict detection.
- `cli-commands`: `init`, `import`, `plan`, `apply`, `status`, `doctor` surfaces
  and their safety behaviors (import secret redaction, no-overwrite guards).

### Modified Capabilities
- (none — greenfield; no existing specs in `openspec/specs/`.)

## Impact

- New codebase under `internal/` (`config`, `secret`, `state`, `jsonutil`, `link`,
  `adapter/{claude,opencode}`, `engine`, `cli`, `scaffold`, `importer`) plus
  `main.go`, `go.mod`, `README.md`, `.gitignore`.
- Dependencies: `spf13/cobra`, `pelletier/go-toml/v2`, `tidwall/sjson`+`gjson`,
  `tailscale/hujson`; standard `testing` + `crypto/sha256`.
- Runtime effects on the user's machine: writes to `~/.claude.json`,
  `~/.claude/settings.json`, `~/.config/opencode/opencode.jsonc`, and symlinks
  under each tool's `skills/` dir — all surgical and confirmation-gated.
- Supersedes the state-comparison approach in
  `docs/superpowers/plans/2026-06-24-homonto.md` (Tasks 4, 8, 9, 11) with the
  hashed-state idempotency model.
