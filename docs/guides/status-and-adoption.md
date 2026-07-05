# Status, drift, and adoption

This guide explains what `homonto status` reports and how `homonto apply`
adopts resources that already exist on disk. For the full command surface see
`cli-commands`; for the guarantees behind this behavior see the
`apply-pipeline` and `tool-adapters` specs.

## `homonto status`: drift vs. pending

`homonto status` compares the values homonto currently manages on disk against
the last-applied snapshot in `.homonto/state.json` (the `Applied` hash), and
reports two independent things:

- **Drift** — a managed key whose on-disk value was changed *outside* homonto
  since the last apply, or was deleted on disk:

  ```
  claude setting.model drifted (will reset on apply)
  claude mcp.foo missing (deleted out of band)
  ```

- **Pending** — edits you made to `homonto.toml` that have not been applied
  yet. These are **not** drift (the disk still matches the last apply); they
  are reported as a count:

  ```
  1 config change(s) awaiting apply (run `homonto apply`)
  ```

When neither is present, `status` prints `No drift.`

The distinction matters: editing your config no longer looks like something
changed your files behind your back. Use `homonto plan` to see exactly what a
pending apply would write.

## Adoption: matching resources are recorded quietly

When a resource you declare in `homonto.toml` already exists on disk with
exactly the value homonto would write — for example an MCP server you added by
hand, or config imported from an existing setup — `apply` **adopts** it: it
records the resource in `.homonto/state.json` without rewriting the tool file
and without showing a diff line.

- `plan` stays silent about adoption (it prints `No changes` when adoption is
  the only outstanding work — nothing will be *written*).
- `apply` performs the adoption even when it is the only work, without a
  confirmation prompt (only `state.json` is touched), and reports it:

  ```
  Reconciled 1 pre-existing resource(s) into state.
  ```

Why it matters: an adopted resource becomes fully managed — it is now visible
to drift detection and, if you later remove it from `homonto.toml`, to pruning.
Before adoption, a hand-created or imported resource could look managed while
silently escaping both.

Adoption never touches secret-bearing values (those are always re-applied so
their resolved value is verified), and it never writes your tool files — an
`apply` whose only effect is adoption leaves `~/.claude.json`,
`~/.claude/settings.json`, and `opencode.jsonc` byte-for-byte unchanged
(comments included).

## Typical flow

```
$ homonto status
1 config change(s) awaiting apply (run `homonto apply`)   # you edited homonto.toml

$ homonto plan                                            # see the diff
claude:
  ~ setting.model: "opus" -> "sonnet"

$ homonto apply                                           # write it
...
Applied.

$ homonto status
No drift.                                                 # disk matches state
```
