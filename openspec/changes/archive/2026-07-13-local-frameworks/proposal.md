# Local framework resolution end-to-end

## Why

Roadmap E1 (F36), the flagship local/custom-framework capability. The catalog
can already merge overlay framework sources (`catalog.LoadOverlays`, the
foundation) but nothing consumes it: config still rejects every non-builtin
`[frameworks.X]` (F35), the catalog materializes only from the embedded FS, and
expansion only handles `builtin:` sources. This change wires local frameworks
end-to-end so a user can install a framework from their own filesystem through
the same validated, versioned path as a builtin (E1 exit gate). Decision (D1):
`local:<path>` is structurally validated, no digest — the user owns their own
filesystem, exactly like a `local:` skill source.

## What Changes

- **Config**: accept `[frameworks.X] source = "local:<path>"`; `<path>` is a
  framework root (a `framework.toml` whose `name` equals `X`, plus its
  `skills/`/`commands/`/`subagents/` with framework-root-relative paths). Other
  non-builtin sources still fail loudly (F35 preserved for non-local).
- **Catalog**: a resource index that tracks each resource's source filesystem so
  `Materialize`/`MaterializeCommands`/`MaterializeSubagents` resolve content from
  the framework's own FS (the embedded base for builtins, the local dir for local
  frameworks). A local framework is merged from its dir via the overlay path.
- **Config expansion**: a `local:` framework's transitively-expanded resources
  project as `builtin:<name>` (they materialize into the same catalog root),
  reusing the entire existing projection path unchanged.
- **Engine**: build the catalog with the config's local-framework overlays, so
  materialization writes their content into the catalog root like a builtin.

## Impact

- **Specs:** `framework-expansion` gains a requirement that a `local:` framework
  installs through the same validated path as a builtin.
- **Behavior:** builtin-only configs are unchanged (the base FS is every
  resource's source; expansion/materialization identical). New: a `local:`
  framework's resources install.
- **Risk:** medium — new cross-subsystem behavior (config + catalog + engine).
  Guarded by an end-to-end acceptance test (a `local:` framework's skill is
  materialized by apply) plus the full existing suite (builtin path unchanged).

## Non-goals

- Remote/digest-pinned frameworks (a later phase via the trust pipeline).
- `[compat].homonto`, capabilities (later/decision-gated phases).
