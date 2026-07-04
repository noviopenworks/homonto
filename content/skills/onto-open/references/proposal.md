# proposal.md — canonical template

Why + what + capability impact. No design content (that's design.md's job)
and no invented scope — everything traces to the confirmed clarification
summary.

## Template

```markdown
# Proposal: <change-name>

<!-- presets only, first line: `Preset: fix` or `Preset: tweak` -->
<!-- optional: `Depends-on: <change-name>[, <change-name>]` — feeds state.yaml deps -->

## Why

<problem/opportunity, 1–3 paragraphs; why now>

## What Changes

- <bulleted, specific; mark breaking changes **BREAKING**>

## Capability Impact

- **New**: `<capability>` — <one line> (creates docs/specs/<capability>.md)
- **Modified**: `<capability>` — <which requirements change> (delta required)
- Untouched: <capabilities explicitly out of scope>

## Not split  <!-- only when the split-preflight considered a split -->

<why this stays one change>

## Impact

<files/systems/dependencies touched; risks worth naming at open time>
```

## Rules

- `Preset:`/`Depends-on:` markers are machine-read (state rebuild) — keep
  them on their own lines, exactly as shown.
- Capability names must match existing `docs/specs/*.md` files or declare
  a new one.
- Breaking changes are marked at the bullet, not buried in prose.
