# Real dual-binary E2E matrix

End-to-end tests that build `homonto` **and** `onto` from source and run them
against the **actual** Claude Code and OpenCode CLIs installed inside a
container — not stub config files. The `live` cells reuse the invoking user's
own accounts (credentials mounted read-only at run time; never baked into the
image), and OpenCode uses a free model so those checks cost nothing.

## Run

```bash
scripts/e2e-matrix.sh                    # build image + full matrix + internals dump
scripts/e2e-matrix.sh --no-build         # reuse the existing image
scripts/e2e-matrix.sh --skip-live        # projection + onto only (no credentials)
scripts/e2e-matrix.sh onto               # a single cell
scripts/e2e-matrix.sh projection/claude live/opencode
```

Per-cell logs and the internals analysis land in `test/e2e/.out/` (gitignored).

## The matrix

| suite \ tool | claude | opencode | what it proves |
|---|:---:|:---:|---|
| **projection** | ✓ | ✓ | `homonto apply` projects an MCP + setting + skill, then the real tool reads it back (`claude mcp list` / `opencode mcp list`), the setting lands in the tool's config, the skill is symlinked, re-apply is idempotent, and `doctor` is healthy. No account needed. |
| **live** | ✓ | ✓ | The real tool authenticates with the mounted account and answers a trivial prompt (`PONG`). claude uses the oauth credential; opencode uses a free model. |
| **onto** | shared (tool-independent) | | `homonto apply` installs `[frameworks.onto]`, then `onto` drives a change through the full `open → design → build → verify → close` lifecycle and archives it — exercising the framework-install gate, per-phase artifact gates, the tasks-checked gate, dirty-worktree blocking on the release-critical transitions, and `doctor`. |
| **analyze** | — | — | Not pass/fail: dumps the container's internal state (projected config files, skill symlinks, materialized `.homonto/catalog`, state hashes, and the real tools' MCP view) for inspection. |

## Credentials (reused, never stored)

The orchestrator mounts, read-only, only for the `live` cells:

- claude → `~/.claude/.credentials.json`
- opencode → `~/.local/share/opencode/auth.json`

Override the paths with `CLAUDE_CRED` / `OPENCODE_AUTH`, the model with
`E2E_OPENCODE_MODEL`. If a credential file is absent, that live cell is skipped
(reported as `SKIP`) rather than failing.

## Layout

- `Dockerfile` — builds both binaries, installs the real tools via their
  official installers, verifies all four resolve at build time.
- `entrypoint.sh` — selects a suite by `$E2E_SUITE` / `$E2E_TOOL`.
- `suites/` — `projection.sh`, `live.sh`, `onto.sh`, `analyze.sh`, and shared
  `lib.sh`.
- `../../scripts/e2e-matrix.sh` — host orchestrator (build, matrix, mounts,
  summary).

## Relation to the smoke test

`scripts/docker-test.sh` (`test/docker/`) is the fast `homonto`-only smoke image
with stub target files. This matrix is the heavier, real-tool, dual-binary
counterpart that maps to roadmap backlog items 3–4 (dual-binary Docker E2E).
