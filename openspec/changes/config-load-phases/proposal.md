# Split config.Load into explicit decode → migrate → normalize → validate phases

## Why

Roadmap X3 (F43). `config.Load` is a ~200-line monolith that interleaves reading,
TOML decoding, the schema-version guard, the `[agents]`→`[subagents]` fold, scope
normalization, and a large inline block of validation. The X3 exit gate calls for
config loading to split "into explicit phases (decode → migrate → normalize →
validate → expand)… ending the monolith."

## What Changes

Extract `Load`'s existing steps — **in the same order, with no behavior change** —
into named phase functions:

- `decode([]byte) (*Config, error)` — TOML unmarshal + the schema-version
  forward-safety guard.
- `migrate(*Config)` — the `[agents]`→copy-mode `[subagents]` fold (Option C).
- `normalize(*Config)` — subagent scope defaulting.
- `validate(*Config) error` — the resource/framework/subagent/model/MCP/plugin/
  marketplace validation block (unchanged, same order).

`Load` becomes: read file → `decode` → `migrate` → `normalize` → `validate` →
return. Each phase is now individually legible and testable.

Expansion (`Expanded*EntriesForTool`) is left as-is — unifying it into a generic
per-kind pipeline is a larger follow-on; this slice ends the `Load` monolith.

## Impact

- **Specs:** `config-model` gains a requirement that config loading runs as
  explicit ordered phases (decode → migrate → normalize → validate).
- **Behavior:** none — a pure in-order extract-method refactor; every load,
  validation error, and fold behaves exactly as before, pinned by the config
  suite.
- **Risk:** low — mechanical extraction with no reordering; the comprehensive
  config load/validation tests are the safety net.

## Non-goals

- The generic per-kind expansion pipeline (the "expand" phase) — a larger
  follow-on.
- Any validation-rule or behavior change.
