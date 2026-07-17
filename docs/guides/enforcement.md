# Enforcing the onto workflow with hooks

onto's gates are hard **when the binary is invoked** — but nothing forces an
inattentive agent to invoke it. A tool **hook** closes that gap: it runs a check
on an event (e.g. when the agent stops) and fails loudly on a workflow-integrity
problem, so a skipped gate or a broken workspace can't pass silently.

The enforcement primitive is:

```
onto doctor --quiet
```

`onto doctor` is read-only and config-independent; `--quiet` prints nothing and
signals health **only through its exit code** (non-zero when there are findings —
a missing/broken change state, an unresolved dependency, a version skew, ≥3
failed verify rounds, an active change marked archived). That exit code is what a
hook acts on.

## Claude Code — via `settings.json` hooks (works today)

homonto projects `[settings.claude]` surgically into `~/.claude/settings.json`,
and `hooks` is an ordinary settings key — so you can install the guard from
`homonto.toml` with no extra machinery:

```toml
[settings.claude]
hooks = { Stop = [ { matcher = "", hooks = [ { type = "command", command = "onto doctor --quiet" } ] } ] }
```

`homonto apply` writes exactly:

```json
{ "hooks": { "Stop": [ { "matcher": "", "hooks": [ { "type": "command", "command": "onto doctor --quiet" } ] } ] } }
```

Now a Claude session that stops with the onto workspace in a bad state gets the
non-zero hook, surfacing the problem instead of ending on it. Use `PreToolUse`
with a matcher instead of `Stop` to guard *before* a specific action.

## OpenCode — via a plugin

OpenCode has no declarative command hooks; hooks live in a plugin. Drop this
minimal plugin at `.opencode/plugins/onto-guard.ts` (or your global
`~/.config/opencode/plugins/`) and reference it in your `plugin` array
(`[plugins.opencode.onto-guard] source = "./.opencode/plugins/onto-guard.ts"`):

```ts
import type { Plugin } from "@opencode-ai/plugin"

// Runs `onto doctor --quiet` when a session goes idle; a non-zero exit means the
// onto workspace has an integrity finding. Read-only — it never mutates state.
// The exit code is surfaced to the session log so a violated gate cannot end a
// session silently.
export const OntoGuard: Plugin = async ({ $, directory }) => ({
  event: async ({ event }) => {
    if (event.type === "session.idle") {
      const result = await $`onto doctor --quiet`.cwd(directory).nothrow()
      if (result.exitCode !== 0) {
        console.error(`onto doctor failed (exit ${result.exitCode}): onto workspace integrity finding`)
      }
    }
  },
})
```

Adjust the event (`session.idle`, `session.completed`) to taste. The guard
above logs the failure through `console.error`; if you would rather have it
abort the event handler, drop `.nothrow()` and let the non-zero throw. Because
the OpenCode side is a code artifact rather than declarative config, install and
review it yourself — homonto does not project or test the plugin's execution.

## What this buys you

The binary already owns the gates; the hook makes them **non-skippable at the
tool boundary**. Pair it with `onto gate --json` (structured decisions) and the
`onto doctor` findings (version skew, verify rounds) for a workflow that surfaces
its own violations rather than trusting the agent to.
