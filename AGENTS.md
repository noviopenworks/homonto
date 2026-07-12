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

## graphify

This project has a knowledge graph at graphify-out/ with god nodes, community structure, and cross-file relationships.

When the user types `/graphify`, use the installed graphify skill or instructions before doing anything else.

Rules:
- For codebase questions, first run `graphify query "<question>"` when graphify-out/graph.json exists. Use `graphify path "<A>" "<B>"` for relationships and `graphify explain "<concept>"` for focused concepts. These return a scoped subgraph, usually much smaller than GRAPH_REPORT.md or raw grep output.
- Dirty graphify-out/ files are expected after hooks or incremental updates; dirty graph files are not a reason to skip graphify. Only skip graphify if the task is about stale or incorrect graph output, or the user explicitly says not to use it.
- If graphify-out/wiki/index.md exists, use it for broad navigation instead of raw source browsing.
- Read graphify-out/GRAPH_REPORT.md only for broad architecture review or when query/path/explain do not surface enough context.
- After modifying code, run `graphify update .` to keep the graph current (AST-only, no API cost).
