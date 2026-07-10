# Verification Report — catalog-foundation-skills

- **Date:** 2026-07-10
- **Change:** catalog-foundation-skills
- **Mode:** full (29 tasks, 4 delta capabilities, 135 changed files)
- **Result:** PASS
- **Branch:** feature/20260710/catalog-foundation-skills (base `bc85fa2`)

## Fresh command evidence (run 2026-07-10)

| Command | Result |
|---|---|
| `go build ./...` | exit 0 — Success |
| `go vet ./...` | exit 0 — No issues found |
| `go test ./... -count=1` | all 15 packages `ok`, 0 failures, 189 test functions PASS |
| `go run . status` (dogfood `[frameworks.comet]`) | `No drift.` |
| `go run . doctor` | all 31 skills × 2 tools `linked`; only warn = `pass` not on PATH (pre-existing, unrelated) |
| `ls .homonto/catalog/skills/` | 31 materialized skill dirs (comet 8 + superpowers 12 + openspec 11) |
| `readlink .claude/skills/comet` | `.homonto/catalog/skills/comet` (builtin link resolves into materialized catalog) |
| `.homonto/state.json` | `"catalogVersion": "0.1.0"` recorded |

## Full-verification checklist

1. **All tasks complete** — `grep -c '- \[ \]' tasks.md` = 0 unchecked. PASS.
2. **Matches open-phase design.md (D1–D5)** — catalog at repo root + `go:embed` (D1, with task 2.1's non-compiling directive corrected to a root `catalog` embed package per the Design Doc); framework.toml metadata (D2); expansion + transitive deps + dedup (D3); version-gated materialization to `.homonto/catalog/skills/` (D4); adapter builtin resolution + linker managed-root change (D5). PASS.
3. **Matches Design Doc** (`docs/superpowers/specs/2026-07-10-catalog-foundation-skills-design.md`) — final whole-branch review verified all §1–§10 boundaries and the config-agnostic layering. PASS.
4. **All delta-spec scenarios pass** — mapped below. PASS.
5. **proposal.md goals** — embed catalog ✓, TOML framework metadata ✓, `[frameworks.X]` expansion + transitive ✓, materialize + symlink ✓, populate onto/comet/superpowers/openspec ✓, both adapters handle builtin ✓. PASS.
6. **No delta-spec / Design-Doc contradiction** — the two build-phase Spec Patches (framework-expansion scope/targets inheritance; builtin-catalog partial-materialization) were made in the design phase and are recorded in Design Doc §12; no drift introduced during build. PASS.
7. **Design docs locatable** — Design Doc + this report exist under `docs/superpowers/`. PASS.

## Delta-spec scenario → evidence

### builtin-catalog
- Load all frameworks → `TestLoadIndexesFrameworksAndVersion`; real embedded `catalog.New()` exercised by every engine/dogfood run.
- First materialization → `TestApplyMaterializesBuiltinSkills`; dogfood materialized 31 dirs.
- Version-gated re-materialization → `TestApplySkipsRematerializeWhenVersionMatchesAndDirsIntact`; dogfood re-apply → `No drift`.
- Partial materialization not recorded (Spec Patch #2) → `TestApplyRematerializesWhenVersionStale` + `TestApplyRematerializesWhenSkillDirMissing`; engine records version only after `Materialize` returns nil (final-review-verified).

### framework-expansion
- Parse framework metadata → `TestLoadIndexesFrameworksAndVersion` (name/version/deps/skills).
- Framework expands to its skills / Transitive dependency expansion → `TestExpandTransitiveAndDedup`, `TestExpandedSkillsIncludeFrameworkAndDeps` (comet → superpowers + openspec).
- Expanded skills inherit scope/targets (Spec Patch #1) → `TestExpandedSkillsIncludeFrameworkAndDeps` asserts `scope="user"`, `targets=["claude"]` on expanded skills.
- Name collision (atomicity) → `TestExpandedSkillsCollisionWithExplicit`, `TestExpandedSkillsFrameworkVsFrameworkConflict`, `TestExpandedSkillsSameFrameworkDeclDedup`.
- Circular dependency rejected → `TestExpandDetectsCycle`.
- First-release frameworks / comet deps → `catalog/frameworks/comet/framework.toml` deps = `["superpowers","openspec"]`, indexed by `TestLoadIndexesFrameworksAndVersion`.

### config-model
- Catalog available without external files → root `catalog/embed.go` `//go:embed`; real `catalog.New()` loads from the embedded FS with no source tree.
- Builtin skill materializes on first apply / idempotent on re-apply → `TestApplyMaterializesBuiltinSkills`, `TestApplySkipsRematerializeWhenVersionMatchesAndDirsIntact`; dogfood `No drift`.
- Catalog upgrade triggers re-materialization → `TestApplyRematerializesWhenVersionStale`, `TestMaterializeRemovesStaleOnUpgrade`.
- Materialized catalog is generated state (gitignore) → `.gitignore` has `/.homonto/`; dogfood `git status` showed only `homonto.toml` changed.
- Local skill resolves from `homonto/` (MODIFIED) → existing local-source tests still green (`TestEndToEndApplyIsIdempotent`, `TestProjectScopeEndToEnd`).
- Builtin skill resolves from materialized catalog (MODIFIED) → `readlink .claude/skills/comet` → `.homonto/catalog/skills/comet`; `TestApplyMaterializesBuiltinSkills`.

### tool-adapters
- Idempotent link creation → `TestLinkCreatesAndIsIdempotent`, `TestBuiltinSkillLinksToCatalogRoot` (re-plan noop), both adapters.
- Skills-only config still applies → `TestSkillsOnlyRebuildsLostState`; dogfood is a skills-only config.
- Relative local content dir → absolute link targets → `TestRelativeContentDirResolvesAgainstConfig`.
- Builtin skill links to materialized catalog → `TestBuiltinSkillLinksToCatalogRoot` (claude + opencode); dogfood readlink.
- Conflict reported, not clobbered → `TestLinkConflictDoesNotClobber`, `TestBuiltinSkillConflictNotClobbered`, `TestLinkForeignSymlinkIsConflict`.
- Applied link recorded in state → `TestApplyMaterializesBuiltinSkills` records `skill.<name>`.
- De-declared skill pruned only when it is our link → `TestBuiltinSkillPrunedWhenDeDeclared`; managed-root guard covers both `homonto/skills/` and `.homonto/catalog/skills/`.

## Review history
- Per-task reviews on the 5 risk tasks (6,7,8,9,10) — all APPROVED; two Important test-coverage gaps (Tasks 7,10) fixed and re-verified.
- Final whole-branch review (opus): **READY TO MERGE**, 0 Critical / 0 Important.
- Accepted non-blocking follow-ups: (F1) orphaned `.homonto/catalog/skills/` dirs on framework de-declaration — harmless gitignored cache; (M1) `link.managed()` empty-string-root defense-in-depth — fully guarded at call sites today; (M2) expanded-skill `Targets` normalization — unreachable with the current disjoint catalog.

## Adversarial note
The final review adversarially re-verified the four cross-package safety properties (empty-root guard, version-recorded-only-after-success, materialize-before-link, conflict-safety with two roots) against live source and confirmed each; the dogfood exercised the real end-to-end path on this repo's own config with `No drift` and zero conflicts/deletes.

**Conclusion: verification PASSES.** No CRITICAL or IMPORTANT open items.
