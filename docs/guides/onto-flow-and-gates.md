# onto — flow, entries, and exit gates

This is the precise reference for how a change **enters** the onto workflow, how
it **moves** between phases, and the exact **gates** each transition enforces.
For the conceptual overview and the skills side, read
[`onto-workflow.md`](onto-workflow.md) first.

The **`onto` binary owns the state and the gates**; the `onto-*` skills own the
work inside each phase. Every state change goes through the binary (`onto new`,
`onto set …`, `onto advance`, `onto close`) — the skills never hand-edit
`onto-state.yaml`, and the phase is always cross-checked against real file state.

## General flow

```
                 ┌─────────────────── onto advance (one phase per call) ───────────────────┐
                 ▼                                                                          │
   onto new → [ open ] → [ design ] → [ build ] → [ verify ] → [ close ] ──── onto close ──→ archived/
                                                                                   (terminal, success)

   presets:   onto-fix / onto-tweak run a reduced path (open-lite → build → verify → close),
              and upgrade to the full path when scope grows.

   failure:   onto abandon <change>  →  abandoned  (the unsuccessful terminal state)
```

A change tracks its phase and evidence in `docs/changes/<name>/onto-state.yaml`.
The phase set is exactly `open → design → build → verify → close`; `close` is the
terminal phase (reached by advancing), after which `onto close` **archives** the
change. There is no `archive` phase.

## Entering the workflow — `onto new`

`onto new <name>` creates `docs/changes/<name>/` with:

- `onto-state.yaml` at **phase `open`**, `workflow: full` (the default),
- `proposal.md` and `tasks.md` skeletons.

It requires the onto framework to be installed (`homonto apply`), refuses to
clobber an existing change, and validates the name is kebab-case with no path
traversal. (`onto init` scaffolds the `docs/{changes,specs,adr,guides}/` layout
first, once, idempotently.)

## Advancing — `onto advance <change>`

Each call attempts **one** transition and writes nothing unless every gate
below passes, in this order:

1. **Framework installed** (the install gate) and a **valid change name**.
2. State **loads** and the change is **not abandoned**.
3. The current phase has a **next phase** (advancing from `close` is an error).
4. **Required artifacts** for the *current* phase all exist (they accumulate):

   | Leaving phase | Files that must exist |
   |---|---|
   | `open`   | `proposal.md`, `tasks.md` |
   | `design` | + `design.md` |
   | `build`  | + `plan.md` |
   | `verify` | + `verification.md` |

5. **Leaving `build`:** `tasks.md` has **no unchecked items** (`- [ ]`).
6. **Evidence / entry tokens** (recorded via `onto set`, not inferred from files):
   - **Entering `build`** (design→build): `isolation` is set (`branch` or
     `worktree`), so planning/build work is never committed unisolated — **and**
     the change is **not in a dependency cycle** (no valid build order exists).
   - **Leaving `verify`** (verify→close): `verify.result == pass`.
7. **Worktree cleanliness:** entering `close` is **blocked** by a dirty worktree
   *and* blocked if cleanliness can't even be determined (no git). Every other
   transition only **warns** on a dirty worktree and proceeds.

A failed gate exits non-zero and leaves the recorded phase unchanged.

## Exiting — `onto close <change>`

`onto close` archives a change that has reached the `close` phase. Gates, in
order:

1. Framework installed; valid name; state loads.
2. Phase **is `close`** (advance until it reaches close first).
3. **Close-evidence gate** — the tokens the workflow actually produces:
   - `verify.result == pass`, **and**
   - `close.merged == true`, **and**
   - for the **full** workflow only, **guides resolved** — `guides` is `updated`
     or `waived:<reason>` (the `fix`/`tweak` presets don't produce guides, so
     they skip this; an empty/unknown workflow is treated as full — strictest).
4. **Dependencies resolved** — every change in `deps` is already archived
   (an `docs/changes/archive/*-<dep>/` exists).
5. **Clean, determinable worktree** (same rule as entering close).
6. **No-clobber** — the dated archive target must not already exist.

On success it sets `archived: true` and moves the workspace to
`docs/changes/archive/<YYYY-MM-DD>-<name>/`. The move is transactional: if it
fails after the flag is written, the flag is rolled back so a failed close never
leaves a change marked archived at its original path.

`onto abandon <change>` is the other terminal state — the unsuccessful one — for
work that is stopped rather than completed.

## Recording evidence — `onto set`

Gate tokens live in `onto-state.yaml` and are set through `onto set <field>`
(never by hand):

| `onto set` field | Gate it satisfies / records |
|---|---|
| `isolation <branch\|worktree>` | required to **enter build** |
| `verify-result <pass\|fail\|…>` | `pass` required to **leave verify** and to **close** |
| `close-merged` | sets `close.merged=true`, required to **close** |
| `guides <updated\|waived:<reason>>` | required to **close** a full workflow |
| `deps --dep <name> …` | dependency list; each must be archived before **close** |
| `verify-scale` | records the verification level for the verify phase |
| `build-mode`, `tdd-mode` | records how build executes |
| `base-ref` | the git ref the change branched from |
| `supersedes`, `deviates-from` | cross-change relationships (surfaced by `onto graph`) |
| `directive` | a verbatim pre-authorization directive on the change |

## Read-only inspection (no gates, config-independent)

- `onto status` — each active change's derived phase and skeleton validity.
- `onto state <change> [--json]` — a change's full state.
- `onto graph` — the change dependency graph (also detects the cycles the build
  gate rejects).
- `onto doctor [--dir <root>]` — workspace health across layout, state,
  phase/artifact match, dependency resolution, and archive layout; non-zero on
  any finding. It also reports a **version skew** when the `onto` binary and the
  homonto that installed the onto framework have drifted apart — run
  `homonto update` (or align the two binaries) to re-sync.

## Driving it from the tool — slash commands

`homonto apply` installs a slash command per phase and preset, so you can drive
the flow from the command palette: `/onto` (dispatcher — derives the active
change's phase and routes automatically), plus `/onto-open`, `/onto-design`,
`/onto-build`, `/onto-verify`, `/onto-close`, `/onto-fix`, `/onto-tweak`, and
`/onto-no-slop`. Each command loads its matching skill; the binary still owns
every state change.
