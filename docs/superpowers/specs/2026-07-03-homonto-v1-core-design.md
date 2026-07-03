---
comet_change: homonto-v1-core
role: technical-design
canonical_spec: openspec
---

# homonto v1 core — Technical Design

Requirements are canonical in the OpenSpec delta specs under
`openspec/changes/homonto-v1-core/specs/` (`config-model`, `apply-pipeline`,
`secret-references`, `tool-adapters`, `cli-commands`). This document covers **how**
to build them. Background: `docs/superpowers/specs/2026-06-24-homonto-design.md`
(design spec), `docs/superpowers/plans/2026-06-24-homonto.md` (14-task TDD plan),
`docs/superpowers/specs/2026-07-03-homonto-roadmap.md` (roadmap + the required
idempotency adjustment this design implements).

## Architecture

Normalized desired-state model + per-tool adapters with shared services:

```
homonto.toml ─▶ config.Load ─▶ *Config ─▶ engine ─▶ [claude.Adapter, opencode.Adapter]
                                             │            Plan → (confirm) → Apply
shared: secret.Resolver · link (symlinks) · jsonutil (surgical merge) · state.Store · plan.Render
```

Packages (`internal/`): `config`, `secret`, `state`, `jsonutil`, `link`,
`adapter` (+ `adapter/claude`, `adapter/opencode`), `plan`, `engine`, `cli`,
`scaffold`, `importer`. Entry: `main.go`. Adding a tool later = one new `Adapter`.

Stack: Go 1.22+, `spf13/cobra`, `pelletier/go-toml/v2`, `tidwall/sjson`+`gjson`,
`tailscale/hujson`, stdlib `crypto/sha256`. Module `github.com/noviopenworks/homonto`.

## Apply pipeline (six stages)

1. **Parse** `homonto.toml` → `*Config`.
2. **Read** per adapter: current managed values from the tool's real files.
3. **Plan** per adapter: `diff(desired, current, state)` → `[]Change` (tokens
   still unresolved).
4. **Print + confirm** (`plan.Render`, `[y/N]`, `--yes`). Never resolves secrets.
5. **Resolve** all confirmed changes' `${…}` tokens — all-at-once, abort on any
   error before any write (two-phase).
6. **Apply** atomic file writes + symlinks; record state (`{desired, hash}`) and
   `state.Save` last.

`homonto plan` runs stages 1–4 only.

## Secret-idempotency model (the core design point)

The original plan compared the *unresolved* desired value against the *resolved*
on-disk value, so every secret-backed key showed a spurious `~ update`, and a
drift/update would place the resolved secret into `Change.Old`. Fix:

**State schema.** `state.json` stores, per tool/key, an entry:

```go
type Entry struct {
    Desired string `json:"desired"` // unresolved value, may contain ${...}
    Applied string `json:"applied"` // sha256(resolved value written to disk)
}
type State struct { Managed map[string]map[string]Entry `json:"managed"` }
```

**Plan decision (per managed key).**

```
disk absent                                                   → create
desired has NO secret ref:  disk == desired ? noop : update   (direct JSON compare)
desired HAS secret ref (secret.ContainsRef):
    in-state && state.Desired == desired && state.Applied == sha256(disk)
                                                              ? noop : update
```

**Apply.** After resolving a change's value to `resolved`, write it, then
`state.Set(tool, key, Entry{Desired: c.New, Applied: sha256hex(resolved)})`.

**Redaction.** For any change on a secret-bearing key (`secret.ContainsRef(New)`),
`Change.Old` is set to `«secret»` (never the on-disk resolved value), so plan
output and logs stay plaintext-free. `Change.New` already carries the unresolved
token. `plan.Render` prints values verbatim; the redaction happens where the
`Change` is constructed in each adapter's `Plan`.

**Hash.** `sha256` hex of the exact resolved value string written for the key. A
small `Hash(s string) string` helper lives in `internal/secret` (or `internal/state`).
Value formatting is normalized (marshal→unmarshal→marshal) before hashing and
before non-secret comparison so `"opus"` vs `opus` cannot cause false diffs.

Why this over alternatives: token-only match loses drift detection for secret
values; resolving at plan time violates "plan never touches `pass`"; storing
plaintext in state violates "state is shareable". (See `design.md`.)

## Adapters — concept → file mapping

| Concept | Claude | OpenCode |
|---|---|---|
| MCP | `~/.claude.json` `mcpServers.<n>` | `opencode.jsonc` `mcp.<n>` (`type:local`, `command`, `enabled`, `environment?`) |
| Settings | `~/.claude/settings.json` top-level | `opencode.jsonc` top-level |
| Plugin | `settings.json` `enabledPlugins.<n>=true` | `opencode.jsonc` `plugin[]` append (`jsonutil.EnsureArrayElem`) |
| Skill (owned) | symlink `content/skills/<n>` → `~/.claude/skills/<n>` | → `~/.config/opencode/skills/<n>` |

Managed-key namespacing in plan/state: `mcp.<n>`, `setting.<k>`, `plugin.<n>`.
JSONC normalized via `hujson.Standardize` before edit (comment-loss caveat in
rewritten regions — documented in README). Surgical edits via `sjson`; unmanaged
keys preserved. `New(home, content string)` constructor for both adapters; tests
inject a temp `$HOME`.

## Error handling — fail safe

- Two-phase: resolve all confirmed secrets before any write; abort naming the
  missing ref (`apply-pipeline` scenario).
- Atomic temp+rename per file; `state.json` written last.
- Unparseable existing tool file → that adapter aborts/reports; others proceed.
- Symlink conflict (target exists, not our link) → reported, not clobbered.
- Dangling refs (missing skill in `content/`) surfaced by `doctor`/plan.
- Missing tool config dir → adapter skips; `doctor` reports.

## Testing strategy (TDD, table-driven)

- **config**: TOML→model, target defaulting, missing-file error, token preserved.
- **secret**: env+pass resolve, missing-ref error, `ContainsRef`, `Hash` stable.
- **state**: absent→empty, save/reload, `{Desired,Applied}` round-trip.
- **jsonutil**: `SetJSON` preserves unmanaged; `Standardize` strips comments,
  empty→`{}`; `EnsureArrayElem` idempotent.
- **link**: create+idempotent; conflict does not clobber.
- **adapters (golden)**: surgical merge keeps unmanaged keys/comments; MCP shapes;
  plugin append; **idempotency incl. a secret-backed MCP is a noop on 2nd plan**.
- **secret-safety**: plan output AND `state.json` contain no resolved secret —
  including the **drift-of-a-secret-key** update path (redacted `Change.Old`).
- **engine**: two-phase abort writes nothing on missing secret.
- **e2e**: `init`→edit→`plan`→`apply` projects into both tools + symlinks; second
  apply is a no-op including a secret-backed value.

## Build plan mapping

Implements the 14-task plan with ⚑ adjustments folded into
`openspec/changes/homonto-v1-core/tasks.md`: hashed `state.Entry` (Task 4/2.1), a
`Hash` helper (1.4), state-aware + redacting adapter `Plan` (Tasks 8–9 / 3.2–3.3),
extended secret-safety test (3.4), drift via same logic (Task 11 / 4.2), and the
idempotent-with-secret e2e assertion (Task 14 / 5.1).

## Risks

- JSONC comment loss in rewritten regions — documented caveat, not a goal.
- Low-entropy secrets under sha256 — acceptable for real API keys; documented.
- Value-formatting false diffs — mitigated by JSON normalization before compare/hash.
