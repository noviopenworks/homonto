# Project Development Instructions Design

## Goal

Provide one project-local development workflow for both OpenCode and Claude
Code. The workflow must guide contributors through the repository's Comet
change process while keeping routine work pragmatic and verifiable.

## Structure

- `AGENTS.md` at the repository root is the canonical instruction file.
- `CLAUDE.md` at the repository root imports `AGENTS.md` and contains no
  duplicate policy.
- Both files are committed with the project so the workflow applies only when
  an assistant operates in this repository.

## Development Workflow

The canonical instructions will require assistants to:

1. Start new development with `/comet` and respect active change state.
2. Use CodeGraph before text search or direct code reads when `.codegraph/`
   exists; use normal repository inspection when it does not.
3. Consult the relevant specs, ADRs, and existing implementation before
   changing behavior.
4. Keep changes focused, avoid unrelated reversions, and preserve user work.
5. Add or update focused tests for behavior changes and run the smallest
   relevant verification command before reporting success.
6. State verification evidence and any remaining test gap clearly.

Graphify is optional: use it for broad architecture, documentation, or
cross-cutting roadmap analysis, not routine code navigation.

## Failure Handling

- If CodeGraph is unavailable or the project is not indexed, continue with
  repository-native search and reads rather than blocking development.
- If an active Comet change conflicts with the requested work, inspect its
  state and ask before changing scope.
- If verification cannot run, report the exact command and reason instead of
  claiming success.

## Validation

Verify that both root instruction files exist, that `CLAUDE.md` imports
`AGENTS.md`, and that the workflow does not contradict the contributor guide
in `README.md`.
