---
comet_change: onto-supersedes-edge
role: technical-design
canonical_spec: openspec
status: draft
---

# onto-supersedes-edge — Technical Design (X1)

OpenSpec is canonical; full approach in design.md. Adds the `supersedes` typed
edge: `State.Supersedes []string` (ungated, mirrors Deps) settable via `onto set
supersedes <change> --change <name>` (mirrors deps); `onto graph` derives a
`supersedes` edge (change → each superseded change). Schema decision made:
supersedes is a list of change names on the superseding change. The next derivable
typed edge; tests/released-in/deviates-from still need untracked data.

## Risk posture

Low — mirrors the deps field/setter/graph-edge pattern. Go tests pin the setter
round-trip + immutability of other fields + the graph edge.

## Out of scope

tests/released-in/deviates-from edges; validating the superseded change exists.
