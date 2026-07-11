# Verification Report: claude-marketplace-registration (v1.2 #3)

- **Change**: `claude-marketplace-registration` — `[marketplaces.claude.<name>]` → `extraKnownMarketplaces.<name>`
- **Date**: 2026-07-11
- **Phase**: verify
- **Verify mode**: full (2 capabilities, config + adapter)
- **Result**: PASS — final review found no bugs

## Scope

`internal/config/config.go` (`Marketplace`/`Marketplaces` model + validation +
reserved key), `internal/adapter/claude/{claude,util}.go` (new `marketplace.<name>`
managed namespace → `extraKnownMarketplaces.<name>`), tests, README + roadmap.

## Full verification checks

| # | Check | Result |
|---|-------|--------|
| 1 | All tasks completed `[x]` | PASS |
| 2 | Matches `design.md` decisions (D1 model, D2 validation, D3 namespace) | PASS |
| 3 | Matches Design Doc (marketplaceValue helper, four-namespace exclusion) | PASS |
| 4 | All delta-spec scenarios pass | PASS |
| 5 | `proposal.md` goals satisfied | PASS |
| 6 | No delta-spec / Design Doc contradictions | PASS |
| 7 | Design Doc locatable | PASS |

## Delta-spec scenario → test mapping

| Scenario | Test | Result |
|---|---|---|
| parse github marketplace | `TestLoadMarketplace` | PASS |
| unknown source rejected | `TestLoadRejectsUnknownMarketplaceSource` | PASS |
| missing locator rejected | `TestLoadRejectsMarketplaceMissingLocator` | PASS |
| reserved `extraKnownMarketplaces` key | `TestLoadRejectsReservedMarketplaceSetting` | PASS |
| github marketplace projected | `TestClaudeProjectsMarketplace` | PASS |
| autoUpdate only when set | `TestClaudeMarketplaceAutoUpdate` | PASS |
| de-declared pruned | `TestClaudeMarketplaceDeDeclared` | PASS |
| adopt pre-existing | `TestClaudeAdoptsMarketplace` | PASS |
| deterministic plan | `TestClaudeMarketplacePlanDeterministic` | PASS |
| four-namespace idempotency | `TestClaudeFourNamespaceIdempotency` | PASS |

## Commands run

| Command | Result |
|---|---|
| `go build ./...` | Success |
| `go test ./... -count=1` | 347 passed, 23 packages |
| `go test -race ./internal/config/... ./internal/adapter/claude/...` | 96 passed |
| `go vet ./...` | No issues |
| `gofmt -l .` | empty |

## E2E (real `homonto` binary, temp $HOME)

A github `[marketplaces.claude.official]` (`repo="anthropics/claude-plugins"`,
`auto_update=true`) + `[plugins.claude.hud] source="hud@official"` → `apply`
wrote `extraKnownMarketplaces.official = {autoUpdate:true, source:{repo:…,source:github}}`
AND `enabledPlugins:{"hud@official":true}`; a second `plan` reported **"No
changes. Everything up to date."** (idempotent — the four-namespace read-back
exclusion holds).

## Code review (review_mode: standard) — no bugs

The final review verified: the `extraKnownMarketplaces` read-back exclusion
(idempotency), autoUpdate-nil symmetry (emitted only when set, on both desired
and read-back), canonical locator sub-object (only type-relevant fields, no
phantom empty url/path), `marketplace.` managed-prefix registration, `@`/name
EscapePath symmetry across desired/apply/prune/read-back, and prune isolation. No
CRITICAL/IMPORTANT/MINOR fixes. One accepted non-blocking observation: partial-
object projection could show a perpetual `update` if the Claude CLI itself
injects fields into a managed entry — inherent to every partial-object namespace
here (pluginConfigs included), an external-tool behavior, accepted tradeoff.

## Conclusion

Verification PASS. **This completes roadmap v1.2 Plugin Configuration** — declare
(#1), enable/disable (#1), per-plugin config (#2), and marketplace registration
(#3). The next roadmap phase is v1.3 Tool TUI Configuration.
