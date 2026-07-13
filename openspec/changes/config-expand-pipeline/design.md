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
