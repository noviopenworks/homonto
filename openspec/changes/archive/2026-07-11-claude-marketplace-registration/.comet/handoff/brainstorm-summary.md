# Brainstorm Summary
- Change: claude-marketplace-registration
- Date: 2026-07-11
## Confirmed Technical Approach
v1.2 #3 (final). New `[marketplaces.claude.<name>]` model (Marketplace{Source,Repo,URL,Path; AutoUpdate *bool}) + validation (source ∈ github/url/git-subdir/directory, required locator present). New Claude managed namespace `marketplace.<name>` → `extraKnownMarketplaces.<name>` = `{"source":{"source":<type>,<locator>},["autoUpdate":bool]}`. Mirrors the `pluginconfig.` namespace exactly (desired/read-back/apply/prune/managed-prefix). Formats: [[plugin-config-formats]].
## Key Trade-offs and Risks
- FOUR managed namespaces in settings.json now (settings/enabledPlugins/pluginConfigs/extraKnownMarketplaces); read-back MUST skip all four or the object leaks as a phantom setting. → mixed-namespace re-plan test.
- Canonical source sub-object: only type-relevant locator fields emitted so github never carries empty url/path differing from adopted disk entry.
## Testing Strategy
TDD RED first. E2E: marketplace+plugin → extraKnownMarketplaces + enabledPlugins on disk, idempotent. Full regression.
## Spec Patches
None. Delta specs (config-model + tool-adapters ADDED) carry the model + projection scenarios.
