# Design — onto graph implements edges

## Node kinds

`graphNode` gains `Kind string` (`"change"` | `"capability"`). Change nodes set
`kind: "change"` (id/phase/archived as today); capability nodes are
`{kind: "capability", change: <capability>}` (id/phase empty), deduplicated by
name across all changes.

## implements edges

For each change directory, read `<dir>/specs/` for `*.md` files (onto's delta-
spec layout is `specs/<capability>.md`). Each file names a capability
(`<capability>` = filename without `.md`); emit an `implements` edge
`{from: change, to: capability, type: "implements"}` and ensure a capability
node exists. A change with no `specs/` dir (or an empty one) contributes nothing
— unchanged.

## Output

Nodes now carry `kind`; edges include both `depends-on` and `implements`.
Deterministic: nodes sorted by (kind, name), edges by (type, from, to). `--json`
shape is `{nodes:[{id,change,phase,archived,kind}], edges:[{from,to,type}]}`.

## Risk

Low — read-only `os.ReadDir` of each change's specs dir added to the existing
enumerator. Tests build a change with a `specs/<cap>.md` and assert the
capability node + implements edge + JSON.

## Alternatives
- Emit implements edges to spec *files* rather than capability names — rejected;
  the capability (the file's basename) is the stable traceability target.
