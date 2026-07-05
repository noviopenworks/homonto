# Delta Spec: tool-adapters (address-deep-review)

## ADDED Requirements

### Requirement: Claude MCP schema fidelity

The Claude adapter SHALL emit each MCP server in the schema real Claude
Code writes: `type` `"stdio"`, `command` as a **string** naming the
executable, `args` as the remaining argv array (omitted when empty), and
`env` omitted when empty — matching the output of `claude mcp add`.
Schema conformance SHALL be asserted by fixtures taken from real tool
files, not from homonto's own round-trips.

#### Scenario: Real-schema emission

- **GIVEN** an MCP whose command is `["npx", "-y", "some-server"]`
- **WHEN** apply projects it into `~/.claude.json`
- **THEN** `mcpServers.<name>` contains `"type": "stdio"`,
  `"command": "npx"` (a string), and `"args": ["-y", "some-server"]`

#### Scenario: Legacy array shape self-heals

- **GIVEN** an on-disk `mcpServers` entry in the old all-in-`command`
  array shape
- **WHEN** `homonto plan` runs
- **THEN** the entry is reported as an update to the real schema, and the
  next apply rewrites it

### Requirement: Declarative pruning

State-recorded keys (prefixes `mcp.`, `setting.`, `plugin.`, `skill.`) SHALL
be planned as deletes when absent from the desired config and, on apply,
removed from the tool files and from state. Keys on disk that were never
recorded in state SHALL never be deleted.

#### Scenario: Removed MCP pruned

- **GIVEN** an MCP that was applied (recorded in state) and is then
  removed from `homonto.toml`
- **WHEN** the user runs plan and confirms apply
- **THEN** the plan shows a delete for that key, apply removes it from the
  tool file, and state no longer contains it

#### Scenario: Drift not mistaken for orphan

- **GIVEN** a key present on disk but absent from both the config and
  state (the user's own, or drifted in out-of-band)
- **WHEN** plan and apply run
- **THEN** the key is neither planned as a delete nor removed — pruning
  keys off state records only

### Requirement: Injection-safe key handling

Dynamic path segments SHALL be escaped per the JSON-path library's rules —
every MCP name, setting key, plugin name, and skill name used in a read or
write path — so names containing `.`, `*`, `?`, or `\` land as single
literal keys and converge on re-apply.

#### Scenario: Dotted name lands as a literal key

- **GIVEN** an MCP named `a.b`
- **WHEN** apply projects it
- **THEN** the tool file gains one literal key `"a.b"` (not a nested
  `a` → `b` object), and a second plan reports it as a noop

### Requirement: Deterministic plan output

Two consecutive plans SHALL render identically over an unchanged config
and unchanged disk state; adapters SHALL sort desired keys and delete keys
before emitting changes.

#### Scenario: Consecutive plans are byte-identical

- **GIVEN** any config with multiple MCPs, settings, and skills
- **WHEN** `homonto plan` runs twice with nothing changed in between
- **THEN** the two rendered outputs are identical

## MODIFIED Requirements

### Requirement: Owned content linked by symlink with conflict detection

Owned skills SHALL be linked (not copied) from `content/skills/<name>` into
each tool's skills directory, and pending link work SHALL be visible as plan
changes: a missing link appears as a create, a link pointing at the wrong
target appears as an update, and a correct link is a no-op. `apply` SHALL
create the links even when they are the only pending changes, and SHALL
record each applied link in state (`skill.<name>`: desired target path plus
applied hash) so drift detection and pruning both see it. A skill removed
from the config SHALL have its link pruned only when the existing path is a
symlink pointing into homonto's managed content directory. If the target
already exists and is not homonto's link, the adapter SHALL report a
conflict and SHALL NOT clobber it — for creation and for pruning alike.

#### Scenario: Idempotent link creation

- **WHEN** a skill symlink does not yet exist
- **THEN** plan lists a create for that link, apply creates it, and a second
  plan/apply reports no change for that link

#### Scenario: Skills-only config still applies

- **GIVEN** a config whose only content is `[skills] own`
- **WHEN** the user runs `homonto apply` and confirms
- **THEN** the plan shows one create per missing link and apply creates
  every link (it does not short-circuit as "no changes")

#### Scenario: Relative content dir yields absolute link targets

- **GIVEN** homonto invoked from any working directory with a relative
  content dir (the default `content`)
- **WHEN** apply creates skill links
- **THEN** every symlink target is an absolute path resolved against the
  config file's directory, and the link does not dangle

#### Scenario: Conflict is reported, not clobbered

- **WHEN** the link target exists as a real file or points elsewhere
- **THEN** apply reports a conflict and leaves the existing file untouched

#### Scenario: Applied link recorded in state

- **WHEN** apply creates a skill link
- **THEN** state contains a `skill.<name>` record, and `homonto status`
  reports drift if the link is later changed out-of-band

#### Scenario: De-declared skill pruned only when it is our link

- **GIVEN** a skill removed from `[skills] own` whose target path is a
  real file (or a symlink pointing outside homonto's content dir)
- **WHEN** apply processes the resulting delete
- **THEN** the path is left untouched and a conflict is reported; only a
  symlink into homonto's managed content is removed
