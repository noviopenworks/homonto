# homonto CLI reference

Every command, flag, and exit-code contract of the `homonto` binary. For the
`onto` binary see the [onto reference](onto-reference.md).

Global behavior:

- `--config <path>` (persistent flag, default `homonto.toml`) selects the
  config file for `plan`, `apply`, `status`, `doctor`, and `import`.
  `init` instead takes an optional target directory argument.
- All output is written to **stderr** (cobra's default) — redirect with `2>&1`
  when capturing output in scripts.
- Unless `--exit-code` is noted below, commands exit `0` on success and
  non-zero on error.

## `homonto init [dir]`

Scaffold a starter repo: `homonto.toml`, `.gitignore` (excluding `.homonto/`),
`.env.example`, and `homonto/skills/`. Writes into `dir` (default: the current
directory) and **never overwrites** an existing file.

```console
$ homonto init
$ homonto init ~/dotfiles/ai     # scaffold somewhere else
```

## `homonto plan`

Print the diff between the desired state (`homonto.toml`) and what is on disk.
Writes nothing and **never resolves or prints a secret** — references stay
`${…}` tokens.

| Flag | Effect |
|---|---|
| `--output text\|json` | output format (default `text`) |
| `--exit-code` | opt-in exit taxonomy: exit `2` when changes are pending |

The diff is Terraform-style: `+` create, `~` update, `-` delete; unchanged keys
are silent.

```console
$ homonto plan
claude:
  ~ setting.model: "opus" -> "sonnet"

$ homonto plan --output json | jq .        # machine-readable
$ homonto plan --exit-code && echo clean   # CI: fail when an apply is pending
```

## `homonto apply`

Project the config into the tools: print the plan, confirm (`[y/N]`), then
write.

| Flag | Effect |
|---|---|
| `--yes` | skip the confirmation prompt |

Guarantees (details in [projection & state](projection-and-state.md) and
[secrets](secrets.md)):

- **Two-phase**: every secret is resolved up front, *before* any file is
  written; one failed resolution aborts the whole apply untouched.
- **Atomic writes**: each file is written via temp + rename — an interrupted
  run never leaves a half-written file.
- **Surgical**: only managed keys are written; unmanaged keys are preserved.
  An unparseable tool file makes that adapter abort and report — never
  overwrite.
- **State per adapter**: state is saved after each successful adapter, so a
  failure in the second tool never loses the first tool's records.
- **Adoption**: a declared resource that already exists on disk exactly as
  homonto would write it is recorded into state quietly, with no file write.

## `homonto status`

Compare managed values on disk against the last-applied snapshot and report
two independent things:

- **Drift** — a managed value changed on disk *outside homonto* (or was
  deleted): `claude setting.model drifted (will reset on apply)`.
- **Pending** — unapplied `homonto.toml` edits, reported as a count:
  `1 config change(s) awaiting apply (run `homonto apply`)`.

When neither is present it prints `No drift.`

| Flag | Effect |
|---|---|
| `--output text\|json` | output format (default `text`) |
| `--exit-code` | opt-in taxonomy: exit `2` on pending, `3` on drift |

## `homonto doctor`

Environment health check: is `pass` on `PATH`, do the tool config locations
exist, and does each owned skill have intact content plus both tool links?

| Flag | Effect |
|---|---|
| `--output text\|json` | output format (default `text`) |

## `homonto update`

Re-materialize this binary's embedded catalog (frameworks, skills, commands,
subagents) and re-project it, bringing installed content up to the running
version. Prints the version transition (binary, catalog, per-framework) and
shares apply's plan → confirm → apply flow.

| Flag | Effect |
|---|---|
| `--yes` | skip the confirmation prompt |

It does **not** download or replace the binaries themselves — install those
the usual way (`go install …@latest` or the release archives), *then* run
`homonto update`. State records the versions behind each apply, and
`onto doctor` warns when the `onto` binary and the homonto that installed its
framework have drifted apart.

## `homonto import`

Experimental adoption helper: bootstrap a starter `homonto.toml` from your
current setup. Deliberately narrow — it reads **Claude's global MCP servers
only** (`~/.claude.json` `mcpServers`):

- refuses to overwrite an existing config unless you pass `--force`;
- redacts env values that *look* like secrets into `${pass:…}` references
  (best-effort, not exhaustive — review before sharing);
- copies `command`/`args` verbatim;
- skips non-stdio (url/http) servers with a warning;
- does **not** import skills, plugins, settings, or OpenCode config.

Treat its output as a starting point to review, not a complete migration.

| Flag | Effect |
|---|---|
| `--force` | overwrite an existing config file |

## `homonto cache gc`

Reclaim entries in the content-addressed remote cache
(`.homonto/cache/remote/`) that no `.homonto/remote.lock.json` entry
references. Kept out of `apply` on purpose, so reverting a `digest` pin can
still roll back from cache. See
[remote source trust](remote-source-trust.md).

## `homonto version`

Print the release-stamped build version (`homonto --version` works too).

## `homonto completion <shell>`

Generate a shell autocompletion script (bash, zsh, fish, powershell) — a
standard cobra facility:

```console
$ homonto completion zsh > "${fpath[1]}/_homonto"
```
