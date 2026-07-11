# Development Instructions

Start new development with `/comet`; inspect active change state before
starting separate change work. Treat `openspec/changes/` as active work and
`docs/superpowers/` as active design and implementation planning; completed
change history lives in `openspec/changes/archive/`.

When `.codegraph/` exists, use CodeGraph before grep, glob, or direct reads to
locate and understand code. If it is absent or unavailable, continue with the
repository's normal inspection tools. Use Graphify only for broad architecture,
documentation, or cross-cutting analysis.

Read the relevant capability specs, ADRs, and nearby implementation before
changing behavior. Keep changes focused. Do not revert unrelated user work.

For behavior changes, add or update focused tests and run the narrowest useful
verification command. Before reporting completion, state the command result
and any verification gap.
