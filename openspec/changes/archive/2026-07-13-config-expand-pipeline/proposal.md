# Generic per-kind framework expansion pipeline

## Why

Roadmap X3 (F43), the remaining "generic per-kind expansion pipeline" piece
(the decode→migrate→normalize→validate split already shipped in
`config-load-phases`). `ExpandedSkillEntriesForTool`,
`ExpandedCommandEntriesForTool`, and `ExpandedSubagentEntriesForTool` are three
~74-line functions that are byte-for-byte identical except the base-entries
accessor, the catalog `Expand*` method, and the resource-kind word in error
messages — ~220 lines of triplicated framework-expansion/merge/conflict logic.

## What Changes

- Extract the shared framework-expansion loop into one generic
  `expandEntriesForTool(tool, kind, base, expand)` that iterates frameworks in
  deterministic order, expands each declared framework's resources of the kind
  through the catalog, tags them `builtin:<name>` with the framework's
  scope/targets, and merges with the same explicit-declaration and
  conflicting-scope/targets checks — parameterized by the kind word (for errors)
  and an `expand func(*catalog.Catalog, catName string) ([]string, error)`
  adapter over the per-kind catalog method.
- The three `Expanded*` methods become thin wrappers (base entries + kind + the
  catalog adapter). Behavior is byte-identical.

## Impact

- **Specs:** `config-model` gains a requirement that framework expansion runs
  through one generic per-kind pipeline (the same behavior for every kind).
- **Behavior:** none — a pure in-place extraction; every expansion, conflict
  error, and precedence rule is unchanged, pinned by the config suite.
- **Risk:** low — mechanical dedup; the comprehensive config expansion tests are
  the regression gate.

## Non-goals

- Changing any expansion/conflict rule; unifying the non-framework
  (explicit-entry) expansion paths.
