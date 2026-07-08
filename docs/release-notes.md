# Release notes intro

This file is prepended to every GitHub release's auto-generated notes by the
`release` workflow (`--notes-file docs/release-notes.md --generate-notes`), so
every release states the accepted limitations up front. Keep it short; the
per-release changelog is generated automatically below it.

---

## Known limitations

homonto is a young, deliberately narrow tool. For the v0.1.0 beta line:

- **OpenCode JSONC comments are not preserved** on any apply that writes
  `opencode.jsonc` (the file is rewritten as normalized JSON). Accepted for beta.
- **`import` is a narrow Claude MCP bootstrap** — Claude global MCP servers only,
  best-effort secret redaction, no skills/plugins/settings/OpenCode import.
- **Two adapters only:** Claude Code and OpenCode.
- **Secrets require `pass` or an env var** at apply time (`${pass:...}` /
  `${ENV_VAR}`).

See the README's "Known limitations" section for details.
