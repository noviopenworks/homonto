# Script Location and Commands

Canonical path: `comet/reference/scripts.md`

This file is the single source of truth for locating Comet scripts and the state/guard/handoff/archive command surface. Load it once per session, then reuse the cached env vars.

## Bootstrap (run once per session)

Comet scripts are distributed in `comet/scripts/`. **Do not hardcode paths** — locate once, cache in env vars. Sub-skills may reference this section directly and only inline this block when they must be fully self-contained; this file is the single source of truth for updates:

```bash
COMET_ENV="${COMET_ENV:-$(find . "$HOME"/.*/skills "$HOME/.config" "$HOME/.gemini" -path '*/comet/scripts/comet-env.mjs' -type f -print -quit 2>/dev/null)}"
if [ -z "$COMET_ENV" ]; then
  echo "ERROR: comet-env.mjs not found. Ensure the comet skill is installed." >&2
  return 1
fi
COMET_SCRIPTS_DIR="$(node "$COMET_ENV")"
COMET_STATE="$COMET_SCRIPTS_DIR/comet-state.mjs"
COMET_GUARD="$COMET_SCRIPTS_DIR/comet-guard.mjs"
COMET_HANDOFF="$COMET_SCRIPTS_DIR/comet-handoff.mjs"
COMET_ARCHIVE="$COMET_SCRIPTS_DIR/comet-archive.mjs"
COMET_INTENT="$COMET_SCRIPTS_DIR/comet-intent.mjs"

# Stop workflow when script location fails
if [ -z "$COMET_SCRIPTS_DIR" ]; then
  echo "ERROR: Comet scripts not found. Ensure the comet skill is installed." >&2
  return 1
fi
```

After loading comet, agents should run this bootstrap block once, then reuse `$COMET_GUARD`, `$COMET_STATE`, `$COMET_HANDOFF`, `$COMET_ARCHIVE`, and `$COMET_INTENT` throughout the session.

## Auto state update

Guard supports `--apply` flag, automatically updating `.comet.yaml` state fields after checks pass:

```bash
node "$COMET_GUARD" <change-name> <phase> --apply
```

`--apply` delegates to `comet-state transition`. Use these semantic events when state changes need to be expressed directly:

```bash
node "$COMET_STATE" transition <change-name> open-complete
node "$COMET_STATE" transition <change-name> design-complete
node "$COMET_STATE" transition <change-name> build-complete
node "$COMET_STATE" transition <change-name> verify-pass
node "$COMET_STATE" transition <change-name> verify-fail
```

Archive completion is handled by `node "$COMET_ARCHIVE" <change-name>` after OpenSpec moves the change into its date-prefixed archive directory; do not manually transition an `<archive-name>`.

## Resolve next action

After guard-based phase advancement, use the `next` subcommand to determine whether to auto-invoke the next skill:

```bash
node "$COMET_STATE" next <change-name>
```

Output format: `NEXT: auto|manual|done` + `SKILL: <skill-name>` (omitted for `done`) + `HINT` (for `manual` only). With `auto_transition: false`, output is `manual`, which pauses only the next skill invocation and does not block phase updates.

## Archive script

Complete all archive steps in one command:

```bash
node "$COMET_ARCHIVE" <change-name>
```
