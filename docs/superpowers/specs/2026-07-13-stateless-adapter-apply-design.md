---
comet_change: stateless-adapter-apply
role: technical-design
canonical_spec: openspec
status: draft
archived-with: 2026-07-13-stateless-adapter-apply
status: final
---

# stateless-adapter-apply — Technical Design

Deep design for X2's stateless-Apply concern. OpenSpec is the canonical spec;
this defers to `openspec/changes/stateless-adapter-apply/specs` for normative
requirements; the full approach is in the change's `design.md`.

## Decision

Give `Adapter.Apply` a leading `cfg *config.Config` param and have each adapter
re-derive its file entries from cfg (a shared `expand(cfg)` helper called by both
Plan and Apply), removing Apply's hidden dependence on instance fields set by a
prior Plan. Codex (MCP-only) ignores cfg. engine.Apply passes e.Cfg.

## Why

The X2 problem statement names it: "Apply reads mutable adapter fields set by a
prior Plan, not the plan alone." Apply silently under-applies if Plan wasn't
called on the same instance. Re-deriving from the supplied cfg makes Apply
self-sufficient and behavior-identical (same cfg → same entries).

## Risk posture

Low-per-site but broad: an interface signature change across 3 adapters, the
engine, and ~61 adapter test call sites (each already has cfg in scope from its
preceding Plan). No behavior change; conformance suite + all adapter/engine tests
are the safety net. The mechanical test sweep is delegated then rigorously
verified.

## Out of scope

Driving Apply purely from the ChangeSet; transaction journals (F42).
