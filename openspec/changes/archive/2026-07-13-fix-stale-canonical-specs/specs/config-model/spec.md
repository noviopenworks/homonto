# config-model (delta)

## REMOVED Requirements

### Requirement: Agent lifecycle declaration
**Reason:** `[agents.<name>]` is no longer a distinct lifecycle-managed model. It
is folded into the declarative subagent model at config load
(`internal/config/config.go` Option C), so the "lifecycle-managed, declarable"
framing — and its citation of the removed `internal/cli/agents.go` — is false.
Replaced by the deprecated-alias requirement below.

## ADDED Requirements

### Requirement: Deprecated agents table folds into subagents

The `[agents.<name>]` table SHALL be a deprecated backward-compatibility alias
with no separate lifecycle and no command surface: at config load `homonto` SHALL
fold every declared `[agents.<name>]` into an equivalent subagent and then drop
the agents table, so it is projected by `apply` like any other subagent. The fold
SHALL:

- set the subagent `scope` to `user`;
- carry `source`, `version`, and `targets` through unchanged;
- default `mode` to `copy` for a `builtin:` source with an omitted `mode` (a
  builtin source has no linkable on-disk path), and otherwise carry `mode`
  through unchanged;
- let a declared `[agents.<name>]` win over an explicit `[subagents.<name>]` of
  the same name.

After the fold no `agents` table remains on the loaded config.

#### Scenario: An agent declaration folds into a user-scope subagent

- **GIVEN** `[agents.review]` with `source = "local:review"`, `version = "1.2.0"`, `targets = ["claude","opencode"]`
- **WHEN** the config is loaded
- **THEN** it yields a subagent `review` with that source, version, and targets, scope `user`
- **AND** the loaded config has no `agents` table

#### Scenario: A builtin agent with omitted mode folds to copy

- **GIVEN** `[agents.x]` with only `source = "builtin:x"`
- **WHEN** the config is loaded
- **THEN** the folded subagent `x` has mode `copy`

#### Scenario: A declared agent wins over a same-named subagent

- **GIVEN** both `[agents.x]` and `[subagents.x]` are declared
- **WHEN** the config is loaded
- **THEN** the folded `[agents.x]` definition is the effective subagent `x`
