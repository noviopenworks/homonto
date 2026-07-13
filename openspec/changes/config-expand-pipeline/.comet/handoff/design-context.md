# Comet Design Handoff

- Change: config-expand-pipeline
- Phase: design
- Mode: compact
- Context hash: c0122134f15366845ac6f8758869a91e93d3d27def6a202ac5387ad77beb3eae

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/config-expand-pipeline/proposal.md

- Source: openspec/changes/config-expand-pipeline/proposal.md
- Lines: 1-38
- SHA256: 5108ae2b37532c13d10def74a3f6b5038b9bf8a402ff86164ef0e450e23097a7

```md
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

```

## openspec/changes/config-expand-pipeline/design.md

- Source: openspec/changes/config-expand-pipeline/design.md
- Lines: 1-44
- SHA256: 03b15b4fea28469244c642267ebda4a199e1b2ca6c821b95926fbbb3b461ca01

```md
# Design — generic expansion pipeline

## The generic

```go
func (c *Config) expandEntriesForTool(
    tool, kind string,
    base []NamedResource,
    expand func(cl *cat.Catalog, catName string) ([]string, error),
) ([]NamedResource, error)
```
Body = today's shared loop verbatim: seed byName/explicitNames from `base`;
iterate `c.Frameworks` sorted; skip non-catalog or non-targeting frameworks;
lazily build `c.FrameworkCatalog()`; `names, err := expand(cl, catName)`; for each
name, reject an explicit+framework clash, build `NamedResource{Source:
"builtin:"+name, Scope/Targets from fwRes, Mode: "link"}`, merge with the
`sameResource` conflict check; sort and return. `kind` fills the error messages
("skill"/"command"/"subagent").

## The wrappers

```go
func (c *Config) ExpandedSkillEntriesForTool(tool string) ([]NamedResource, error) {
    return c.expandEntriesForTool(tool, "skill", c.SkillEntriesForTool(tool),
        func(cl *cat.Catalog, n string) ([]string, error) {
            exp, err := cl.Expand([]string{n}); return skillNames(exp), err
        })
}
```
Command uses `cl.ExpandCommands`; subagent uses `cl.ExpandSubagents`. Each
expanded type is `{Name, Framework}`, so a tiny `namesOf` per kind extracts the
Names (only Name is used downstream). Subagent's framework-expanded resources are
already `Mode: "link"` today (copy-mode is only for explicit `[subagents]`), so
no special-casing.

## Behavior identity

A pure extraction: the loop, the deterministic ordering, every error string
(kind-parameterized to the same text), and the merge/precedence rules are
unchanged. The config expansion suite (explicit+framework, conflicts, precedence,
local/remote frameworks) pins it.

## Risk
Low — mechanical. Any diff means the extraction slipped.

```

## openspec/changes/config-expand-pipeline/tasks.md

- Source: openspec/changes/config-expand-pipeline/tasks.md
- Lines: 1-9
- SHA256: 382db583f5abb3f4eb4fadcb749c24bfd5856d0776d67ed064465cfd80dadc52

```md
# Tasks — config-expand-pipeline

## 1. Extract the generic pipeline
- [ ] Extract expandEntriesForTool(tool, kind, base, expand) from the three
      Expanded* functions; they become thin wrappers (base + kind + catalog
      adapter). No behavior change. Config suite green unchanged.

## 2. Verify
- [ ] `go test ./... -race`, vet, build, `openspec validate --all` green.

```

## openspec/changes/config-expand-pipeline/specs/config-model/spec.md

- Source: openspec/changes/config-expand-pipeline/specs/config-model/spec.md
- Lines: 1-20
- SHA256: 10cc07a4edf02397dc4b660a48341485c3c2c47691d53480f4a5f08805ddf96d

```md
# config-model

## ADDED Requirements

### Requirement: Framework resource expansion runs through one generic per-kind pipeline

Framework resource expansion SHALL run through a single generic pipeline
parameterized by the resource kind (skills, commands, subagents), rather than a
per-kind copy of the expansion logic. Every kind MUST expand,
tag (`builtin:<name>`), merge, and conflict-check identically — an
explicitly-declared resource also expanded by a framework, and a resource
expanded by two frameworks with conflicting scope/targets, MUST each fail with
the same rule for every kind, and the resulting expanded entries MUST be
identical to the prior per-kind implementation.

#### Scenario: Every kind expands through the same pipeline

- **WHEN** skills, commands, or subagents are expanded from framework declarations
- **THEN** the same expansion, tagging, merge, and conflict rules apply, producing
  the same entries as before

```
