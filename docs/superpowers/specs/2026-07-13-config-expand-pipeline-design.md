---
comet_change: config-expand-pipeline
role: technical-design
canonical_spec: openspec
status: draft
archived-with: 2026-07-13-config-expand-pipeline
status: final
---

# config-expand-pipeline — Technical Design (X3/F43)

OpenSpec is canonical; full approach in the change's `design.md`. The last F43
piece: extract the ~220 lines triplicated across ExpandedSkill/Command/
SubagentEntriesForTool into one generic `expandEntriesForTool(tool, kind, base,
expand)`; the three become thin wrappers (base entries + kind + a per-kind
catalog-Expand adapter). All three expanded types are `{Name, Framework}`, so the
adapter returns names. Pure in-place extraction — every expansion/merge/conflict
rule and error string unchanged; the config expansion suite pins it.

## Risk posture

Low — mechanical dedup; any behavior diff means the extraction slipped. The
comprehensive config expansion tests (explicit+framework, conflicts, precedence,
local/remote frameworks) are the regression gate.

## Out of scope

Changing any expansion rule; the explicit-entry (non-framework) paths.
