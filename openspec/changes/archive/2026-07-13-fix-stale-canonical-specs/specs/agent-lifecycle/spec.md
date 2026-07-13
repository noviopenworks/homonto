# agent-lifecycle (delta)

The imperative agent lifecycle capability is retired. `[agents]` was collapsed
into the declarative `[subagents]` + `apply` model (item 9, `38b32ec`): the
imperative `agents` command group is no longer registered, and the
`internal/agentlock` and `internal/agentblob` packages no longer exist. Every
removed requirement below describes removed behavior. OpenSpec cannot rebuild an
empty spec, so the capability is reduced to a single retirement tombstone; the
surviving truth (that `[agents.<name>]` folds into a subagent at load) lives in
`config-model`.

## REMOVED Requirements

### Requirement: homonto agents list reports declared agents
**Reason:** The imperative `agents` command group is no longer registered
(`internal/cli/root.go` exposes only plan/apply/status/doctor/init/import/version).

### Requirement: homonto agents add installs a declared agent
**Reason:** Imperative install removed; `[agents.<name>]` now folds into a
copy-mode subagent materialized by `apply`.

### Requirement: homonto agents doctor reports agent health
**Reason:** No imperative agent command surface remains; health is covered by
`homonto doctor` over projected subagents.

### Requirement: homonto agents update re-materializes an installed agent
**Reason:** Update/merge-on-upgrade of a separate agent install is removed; the
folded subagent reconciles through `apply`.

### Requirement: homonto agents update --all reconciles every installed agent
**Reason:** Same as update; no imperative bulk reconcile remains.

### Requirement: homonto agents prune removes stale managed installs
**Reason:** The lockfile and its prune were removed with the imperative surface.

### Requirement: Three-way merge engine
**Reason:** Merge-on-agent-upgrade is removed with the imperative surface;
`internal/merge` has no non-test callers.

### Requirement: Agent base-content blob store
**Reason:** The `internal/agentblob` content-addressed base store no longer
exists.

## ADDED Requirements

### Requirement: Imperative agent lifecycle is retired

`homonto` SHALL NOT provide an imperative agent lifecycle: there SHALL be no
`agents` command group, no content-addressed base-blob store, and no
lockfile-driven prune. A `[agents.<name>]` declaration SHALL be folded into a
subagent at config load (see the `config-model` capability) and reconciled
declaratively by `apply`.

#### Scenario: No imperative agent command surface

- **WHEN** a user looks for an imperative agent install, update, doctor, or prune command
- **THEN** none is registered
- **AND** a declared `[agents.<name>]` is projected declaratively through `apply` as a subagent
