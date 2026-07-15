# Release notes intro

This file is prepended to every GitHub release's auto-generated notes by the
`release` workflow (`--notes-file docs/release-notes.md --generate-notes`), so
every release states the accepted limitations up front. Keep it short; the
per-release changelog is generated automatically below it.

---

## What's in this release

This release ships **two binaries** — `homonto` (config projector) and `onto`
(spec-driven workflow operator) — for every supported OS/arch as separate
archives under one `SHA256SUMS`. `onto` requires `homonto` to have installed the
`onto` framework first (`[frameworks.onto]` + `homonto apply`).

### Fixed in v0.1.17 — subagents now track their model routes

Changing a `[models.<tool>.<role>]` route did **not** re-render the subagents
stamped from it. The projected agents stayed frozen at the model they were first
materialized with, while the tool's own `setting.model` — re-read from the routes
on every apply — moved correctly: one config, two different answers. If you have
edited a model route since installing a framework or subagent, **upgrade and run
`homonto apply`** to re-stamp your agents; verify with
`grep '^model:' .homonto/catalog/subagents/*.md`.

Three related defects went with it: a deleted rendered agent variant is now
restored instead of stranding a dangling symlink that `plan`/`status`/`doctor`
all called healthy; `apply` now re-materializes the catalog even when the
projection plan is empty; and `doctor` no longer reports a permanent, unfixable
finding for an OpenCode-primary agent's by-design absent Claude variant.

## Known limitations

homonto is a young, deliberately narrow tool. For the v0.1 beta line:

- **OpenCode JSONC comments are not preserved** on any apply that writes
  `opencode.jsonc` (the file is rewritten as normalized JSON). Accepted for beta.
- **`import` is a narrow Claude MCP bootstrap** — Claude global MCP servers only,
  best-effort secret redaction, no skills/plugins/settings/OpenCode import.
- **Frameworks resolve from the bundled catalog only.** Remote sources exist for
  **subagents** only, and require a `digest = "sha256:…"` pin; homonto never
  re-resolves a pin to newer content on its own.
- **Two full adapters:** Claude Code and OpenCode. **Codex** is an opt-in pilot
  that projects **MCP servers only**.
- **Secrets require `pass` or an env var** at apply time (`${pass:...}` /
  `${ENV_VAR}`).
- **Moving or renaming the repo** breaks skill symlinks (absolute targets):
  delete the stale links and re-apply.

See the README's "Caveats" section and
[`docs/guides/troubleshooting.md`](docs/guides/troubleshooting.md) for details.
