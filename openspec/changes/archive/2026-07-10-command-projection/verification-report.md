# Verification Report — command-projection

- **Date:** 2026-07-10
- **Change:** command-projection
- **Mode:** full (29 tasks, 3 delta capabilities, 33 changed files)
- **Result:** PASS
- **Branch:** feature/20260710/command-projection (base `70dd84d`)

## Fresh command evidence (run 2026-07-10)

| Command | Result |
|---|---|
| `go build ./...` | exit 0 |
| `go vet ./...` | exit 0 |
| `go test ./... -count=1` | 16 packages `ok`, 0 failures |
| `go run . status` (dogfood `[commands.example-command]`) | `No drift.` |
| `go run . doctor` | `command "example-command" linked (claude)` + `(opencode)`; all skills ok |
| `readlink .claude/commands/example-command.md` | `.homonto/catalog/commands/example-command.md` |

## Full-verification checklist

1. **All tasks complete** — 0 unchecked in tasks.md. PASS.
2. **Matches open-phase design.md (D1–D7)** — `catalog/commands/` + `all:commands` embed (D1); framework `[commands]` parse + `ExpandCommands` (D2); `internal/commandpath` singular `command/` for opencode (D3); adapter `commandSource`/`commandsDir`/`command.<n>` links + `managedRoots` third root (D4); `ExpandedCommandEntriesForTool` (D5); engine combined skills+commands materialization under one version gate (D6); placeholder fixture (D7). PASS.
3. **Matches Design Doc** — final whole-branch review verified §1–§10 and the command-specific edges. PASS.
4. **All delta-spec scenarios pass** — mapped below. PASS.
5. **proposal.md goals** — builtin/local command projection both tools ✓, `catalog/commands/` embed + single-file materialize ✓, framework `[commands]` expansion ✓, adapters + doctor ✓, placeholder fixture ✓, flat commands only ✓. PASS.
6. **No delta-spec / Design-Doc contradiction** — a build-phase design-detail correction (the fixture must be framework-declared to be in the catalog index; declared under `onto`) is consistent with the delta specs and how skills work; no scope drift. PASS.
7. **Design docs locatable** — Design Doc + this report under `docs/superpowers/`. PASS.

## Delta-spec scenario → evidence

### command-projection
- Builtin command resolves from materialized catalog / Local command resolves from homonto/commands → `TestBuiltinCommandLinksToCommandCatalogRoot` (both adapters, builtin + local branches); dogfood `readlink`.
- Single-file command materialization (first + version-gated) → `TestMaterializeCommandsWritesFile`, `TestApplyMaterializesBuiltinCommand`, `TestApplyRematerializesWhenCommandFileMissing`.
- Command projection into both tools / idempotent / conflict-not-clobbered / de-declared-pruned-only-when-ours → `TestBuiltinCommandLinksToCommandCatalogRoot`, `TestBuiltinCommandConflictNotClobbered`, `TestBuiltinCommandPrunedWhenDeDeclared` (both adapters).
- Framework command expansion → `TestExpandCommandsTransitiveAndDedup` (catalog), `TestExpandedCommandsExplicitAndTargetFilter` (config); the framework-command path is real now that `onto` declares `example-command` (covered transitively + by the dogfood).
- Command link doctor verification → `TestDoctorReportsLinkedCommand`; dogfood `doctor`.
- Placeholder fixture command → dogfood: `example-command` materializes + links into both tools with `No drift`.

### config-model (MODIFIED "Local provider content root")
- Local command resolves from `homonto/commands/<n>.md` → `TestBuiltinCommandLinksToCommandCatalogRoot` local branch.
- Builtin command resolves from `.homonto/catalog/commands/<n>.md` → dogfood `readlink`.

### framework-expansion (MODIFIED "Framework metadata format")
- Parse framework command table → `TestLoadIndexesFrameworkCommands`; `TestLoadRejectsMissingCommandPath` (validates command paths exist).

## Review history
- Per-task reviews on risk tasks 3 (DFS refactor), 6 (config cross-module), 7 (engine keystone), 8 (claude commands), 9 (opencode commands) — all APPROVED; Task 8 had one Important atomicity finding (command fail-fast before writes) fixed + re-tested.
- Final whole-branch review (opus): **READY TO MERGE**, 0 Critical / 0 Important.
- Accepted non-blocking follow-ups: CM1 (cycle-test chain assertion), CM2 (framework-command config test — now addressable via `[frameworks.onto]`), CM3 (commands-present skip test), CM4 (skill/command block duplication → future `resourcepath`), plus delete-before-conflict-check ordering (pre-existing, shared with skills, self-healing) and stale materialized-command GC (harmless gitignored).

## Adversarial note
The final review adversarially re-verified the cross-package safety properties (empty-root guard, version-recorded-only-after-both, materialize-before-link, command fail-fast-before-write in BOTH adapters, conflict/prune-safety with the third root, skills preserved, layering) against live source. The dogfood exercised the real end-to-end path on this repo's own config: `apply [commands.example-command]` → materialize + link into both tools, `No drift`, 0 conflicts/deletes.

**Conclusion: verification PASSES.** No CRITICAL or IMPORTANT open items.
