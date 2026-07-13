# Design — onto supersedes edge

## State field

`ontostate.State` gains `Supersedes []string` (`yaml:"supersedes,omitempty"
json:"supersedes,omitempty"`), mirroring `Deps`. It is not gated (Validate
ignores it — B1: shape not judgment). `Save` round-trips it; only the set command
writes it.

## Set command

`supersedesCmd` mirrors `depsCmd`: `onto set supersedes <change> --change <name>
[--change …]` → `st.Supersedes = <names>` via `runTransition`. Registered in
`setCmd`. Repeatable `--change` (not a comma-split positional) so names carrying
edge characters are unambiguous.

## Graph edge

`buildGraph`'s per-change `add` emits, for each `st.Supersedes` entry, an edge
`{from: change, to: superseded, type: "supersedes"}` — after the depends-on and
implements edges. Deterministic ordering already sorts edges by (type, from, to).

## Test

- `onto set supersedes alpha --change old1 --change old2` → reload → Supersedes ==
  [old1 old2], other fields unchanged.
- `onto graph --json` over a change with `supersedes: [old]` → a supersedes edge
  change→old.

## Risk
Low — mirrors the deps field/setter and the graph edge pattern. Go tests pin it.
