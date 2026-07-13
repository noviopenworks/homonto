# Design — consolidate structured-doc projection

## High-level approach

Follow the Codex template (`internal/adapter/codex/codex.go`), generalized
to (a) a JSON codec and (b) multiple documents per adapter.

### Shared JSON codec

`internal/jsonutil` already exposes every primitive `structproj.Codec`
requires. Add one small adapter type (in `structproj` or a new
`internal/adapter/jsoncodec`, decided in build) mapping:

| structproj.Codec | jsonutil |
|------------------|----------|
| `EnsureRoot`     | `ObjectRoot` (normalize empty→`{}`) |
| `Get`            | `GetJSON` |
| `Set`            | `SetJSON` |
| `Delete`         | `DeleteJSON` |
| `Canonical`      | `Canonical` |

Both `claude` and `opencode` share this one codec (both are JSON).

### Per-document namespaces

`structproj.Project(tool, prefix, desired, disk, st, codec, pathFor)` acts on
**one document** with **one key prefix**. Each adapter maps its managed keys
to documents:

- **claude**
  - `settings.json` ← keys with prefix `setting.`; `pathFor` → the settings
    JSON path for that key.
  - `.claude.json` ← prefixes `mcp.`, `plugin.`, `pluginconfig.`,
    `marketplace.`; `pathFor` → `mcpServers.<n>`, `enabledPlugins.<source>`,
    `pluginConfigs.<source>`, `extraKnownMarketplaces.<name>`.
- **opencode**
  - `opencode.json` ← prefixes `mcp.`, `setting.`.

Because `structproj.Project`/`Observe`/`Apply` filter recorded keys by
`strings.HasPrefix(k, prefix)`, a single document holding several prefixes is
handled by calling the trio once per prefix against that document (the
existing multi-prefix docs), OR by a prefix that is the empty-string-free
common cut. Build step decides the cleanest split; the invariant is that the
union of per-namespace outputs equals today's flat output.

### Migration order (per adapter, each step green before the next)

1. Introduce the shared JSON codec + its unit test.
2. claude: route `setting.*` (settings.json) through structproj; delete that
   branch of the bespoke loop; run claude + conformance suites.
3. claude: route `.claude.json` prefixes through structproj; delete those
   branches; run suites.
4. opencode: route `opencode.json` prefixes through structproj; delete the
   bespoke loop; run suites.
5. Confirm file-projection code paths in both adapters are untouched.

### Correctness invariant

`structproj` "reproduces the built-in adapters' semantics exactly, including
secret-safe redaction of Old" (its own doc). The migration is behavior-
preserving iff, for every fixture in the conformance suite and every existing
claude/opencode test, plan/apply/observe output is unchanged. Any diff is a
migration bug, fixed before proceeding — never by editing a test to match.

## Key risk (surfaced in open)

Does `structproj` cover every structured-doc behavior the two adapters have
today — specifically the **adopt** path (disk matches desired but state is
stale), **secret-bearing** desired values (never read/expose on-disk value),
and **Old redaction** for updates/deletes of unknown provenance? Reading
`structproj.Project`, all three are already implemented identically to the
claude loop. If build uncovers a claude/opencode structured-doc behavior with
no structproj equivalent (e.g. a canonicalization quirk), the change pauses:
extend `structproj` minimally (additive) or document the divergence — it does
**not** silently change adapter behavior.

## Alternatives considered

- **Full F40 (incl. file-projection) now** — rejected as too large/high-risk
  for one change; file-projection needs its own contract.
- **A per-adapter codec each** — rejected; both are JSON, so one shared codec
  removes more duplication.
