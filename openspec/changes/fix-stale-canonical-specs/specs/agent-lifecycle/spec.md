# agent-lifecycle (delta)

The imperative agent lifecycle capability is retired. `[agents]` was collapsed
into the declarative `[subagents]` + `apply` model (item 9, `38b32ec`): the
`homonto agents` command group is no longer registered, and the `internal/agentlock`
and `internal/agentblob` packages no longer exist. Every requirement below
describes removed behavior. The one surviving truth — that `[agents.<name>]` still
parses and is folded into a subagent at load — moves to `config-model`.

On archive sync, this capability's spec is removed in full.

## REMOVED Requirements

### Requirement: homonto agents list reports declared agents
**Reason:** The `homonto agents` command group is no longer registered
(`internal/cli/root.go` exposes only `plan/apply/status/doctor/init/import`).

### Requirement: homonto agents add installs a declared agent
**Reason:** Imperative install removed; `[agents.<name>]` is now folded into a
copy-mode subagent and materialized by `apply`.

### Requirement: homonto agents doctor reports agent health
**Reason:** No imperative agent command surface remains; health is covered by
`homonto doctor` over projected subagents.

### Requirement: homonto agents update re-materializes an installed agent
**Reason:** Update/merge-on-upgrade of a separate agent install is removed; the
folded subagent reconciles through `apply`.

### Requirement: homonto agents update --all reconciles every installed agent
**Reason:** Same as `update`; no imperative bulk reconcile remains.

### Requirement: homonto agents prune removes stale managed installs
**Reason:** The `agentlock` lockfile and its prune were removed with the
imperative surface.

### Requirement: Three-way merge engine
**Reason:** Merge-on-agent-upgrade is removed with the imperative surface. If any
surviving caller of `internal/merge` remains, its behavior is owned by the spec
of the capability that calls it, not by a retired agent-lifecycle capability.

### Requirement: Agent base-content blob store
**Reason:** The `internal/agentblob` content-addressed base store no longer
exists.
