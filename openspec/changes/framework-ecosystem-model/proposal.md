# A real framework ecosystem model (design)

## Why

Roadmap E1 (F36/F38). Today a framework is an embedded `framework.toml`
(name, content version, unversioned `[dependencies].frameworks`, and
skill/command/subagent name→path maps). The catalog already resolves
dependencies transitively with cycle detection and rejects a resource name
mapped to two different paths, and `F35` rejects any non-builtin
`[frameworks.X]` source at load. What is missing, per the E1 exit gate, is the
model that lets **a fourth framework or a local framework install through the
same validated, versioned path**:

- a **manifest schema version** (distinct from a framework's content version) so
  an external manifest from a newer schema fails closed instead of being
  silently half-read;
- **provided/required capabilities** so a framework can depend on a *capability*
  (e.g. "spec-workflow") rather than a hard framework name;
- **compatibility ranges** (against the homonto version and against dependency
  versions) so an incompatible pairing fails loudly, not at runtime;
- **local/custom framework resolution** — the validated path that lifts F35's
  blanket rejection for a trusted local source;
- an **explicit conflict policy** for two frameworks providing the same resource
  or capability (today: different-path = hard error, same = first-wins collapse);
- **manifest migrations** across schema versions;
- **F38**: either a real plugin *lifecycle* capability or an honest rename of the
  current enable/disable `[plugins]` projection.

## What Changes

This change delivers the E1 ecosystem-model **design** (the manifest schema,
resolution/validation pipeline, capability + compatibility model, local-source
trust reuse, conflict policy, phased plan, and the D1-D5 maintainer decisions)
**and implements the phase-1 MVP** — the additive `manifest_schema` field plus a
fail-closed guard in the framework loader — which the design shows is independent
of D1-D4. The broader model (capabilities, compat ranges, local/custom
resolution, F38) remains design for phased follow-on changes pending those
decisions.

## Impact

- **Specs:** the eventual implementation will extend `framework-expansion`;
  this change records the target requirement so the design is traceable. No code
  or behavior changes here.
- **Behavior:** none (design only).
- **Risk:** none in this change; the point is to de-risk implementation by
  settling the model and its decisions first.

## Non-goals (of this design change)

- Implementing any of the model — the deliverable is the design + decisions.
- Changing the current builtin-only behavior.
