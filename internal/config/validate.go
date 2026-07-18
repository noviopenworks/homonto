package config

import (
	"fmt"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/noviopenworks/homonto/internal/agentfm"
	"github.com/noviopenworks/homonto/internal/remote"
)

// validate rejects a config that would project nothing or corrupt a tool file.
func validate(c *Config) error {
	for kind, resources := range map[string]map[string]Resource{
		"skills":   c.Skills,
		"commands": c.Commands,
	} {
		if err := validateResources(kind, resources); err != nil {
			return err
		}
	}
	// Frameworks have their own source rule: builtin:<name> (expanded from the
	// embedded catalog) or local:<path> (a local framework root). Every other
	// source expands nothing and is rejected loudly (F35).
	if err := validateFrameworkResources(c.Frameworks); err != nil {
		return err
	}
	// onto and to are an exclusive choice per repository: enterprise tooling
	// vs. simple development. Their skills give conflicting process guidance
	// and their binaries each expect to own the workflow, so declaring both
	// is a config error, not a projection concern.
	if _, hasOnto := c.Frameworks["onto"]; hasOnto {
		if _, hasTo := c.Frameworks["to"]; hasTo {
			return fmt.Errorf("parse config: [frameworks.onto] and [frameworks.to] are mutually exclusive; pick one workflow framework per repository (onto for evidence-gated enterprise changes, to for simple development)")
		}
	}
	if err := validateSubagents(c.Subagents); err != nil {
		return err
	}
	if err := validateModels(c); err != nil {
		return err
	}
	// Every other name becomes a key written into a tool's JSON file. sjson
	// treats index-like segments ("0", "-1") as array positions, silently
	// turning the containing object into a JSON ARRAY; empty names address
	// nothing. Reject both up front with the offending entry named.
	for name, m := range c.MCPs {
		if err := validateKey("mcps", name); err != nil {
			return err
		}
		// An MCP with no command cannot project — both adapters would skip it,
		// so a declared server would silently do nothing. Fail fast instead.
		if len(m.Command) == 0 {
			return fmt.Errorf("parse config: mcps entry %q has no command; an MCP server needs a command to run", name)
		}
		// A target that names no known tool matches no adapter, so the MCP is
		// projected nowhere — a silent typo. Only claude and opencode exist.
		for _, target := range m.Targets {
			if !isMCPTarget(target) {
				return fmt.Errorf("parse config: mcps entry %q targets unknown tool %q; valid targets are \"claude\", \"opencode\", and \"codex\"", name, target)
			}
		}
		switch m.Scope {
		case "", "user", "project":
			// ok
		default:
			return fmt.Errorf("parse config: mcps entry %q scope %q is invalid; valid values are \"user\" and \"project\"", name, m.Scope)
		}
		// Codex has no project-level config in the MCP pilot, so a
		// project-scoped server could only silently project globally there —
		// reject the combination instead.
		if m.ScopeOrDefault() == "project" && slices.Contains(m.Targets, "codex") {
			return fmt.Errorf("parse config: mcps entry %q is project-scoped but targets codex, which supports only user scope (~/.codex/config.toml)", name)
		}
	}
	for _, tool := range []struct {
		name string
		m    map[string]Plugin
	}{
		{"plugins.claude", c.Plugins.Claude},
		{"plugins.opencode", c.Plugins.OpenCode},
	} {
		// Both adapters project keyed by source, so two decl names sharing a
		// source would collide on one projected key with last-writer-wins over
		// random map iteration order — a non-deterministic plan. Reject it.
		seenSource := map[string]string{} // source -> first decl name
		for declName, pl := range tool.m {
			if err := validateKey(tool.name, declName); err != nil {
				return err
			}
			// A plugin with no source projects nothing (no enabledPlugins key /
			// no plugin-array value), so a declared plugin would silently do
			// nothing. Fail fast naming the plugin.
			if strings.TrimSpace(pl.Source) == "" {
				return fmt.Errorf("parse config: %s plugin %q has an empty source", tool.name, declName)
			}
			// OpenCode plugins are a plain array on disk with no per-plugin
			// config slot, so a declared config could project nowhere. Reject it.
			if tool.name == "plugins.opencode" && len(pl.Config) > 0 {
				return fmt.Errorf("parse config: %s plugin %q declares config, but OpenCode has no per-plugin config on disk (its plugins are a plain array); remove config", tool.name, declName)
			}
			if prev, dup := seenSource[pl.Source]; dup {
				return fmt.Errorf("parse config: %s plugins %q and %q share source %q", tool.name, prev, declName, pl.Source)
			}
			seenSource[pl.Source] = declName
		}
	}
	// Marketplace declarations project to extraKnownMarketplaces.<name>. Each
	// source kind requires its locator field(s); an unknown source or a missing
	// locator projects nothing meaningful, so fail fast naming the marketplace.
	for name, mk := range c.Marketplaces.Claude {
		if err := validateKey("marketplaces.claude", name); err != nil {
			return err
		}
		switch mk.Source {
		case "github":
			if mk.Repo == "" {
				return fmt.Errorf("parse config: marketplaces.claude %q with source \"github\" is missing required \"repo\"", name)
			}
		case "url":
			if mk.URL == "" {
				return fmt.Errorf("parse config: marketplaces.claude %q with source \"url\" is missing required \"url\"", name)
			}
		case "git-subdir":
			if mk.URL == "" || mk.Path == "" {
				return fmt.Errorf("parse config: marketplaces.claude %q with source \"git-subdir\" is missing required \"url\" and/or \"path\"", name)
			}
		case "directory":
			if mk.Path == "" {
				return fmt.Errorf("parse config: marketplaces.claude %q with source \"directory\" is missing required \"path\"", name)
			}
		default:
			return fmt.Errorf("parse config: marketplaces.claude %q has unknown source %q; valid sources are \"github\", \"url\", \"git-subdir\", \"directory\"", name, mk.Source)
		}
	}
	// Settings keys that homonto itself manages in the same tool file would
	// collide with its own writes: claude projects plugins as `enabledPlugins`
	// into settings.json; opencode projects MCPs and plugins as the `mcp` and
	// `plugin` structures in opencode.jsonc. Reject those reserved names.
	//
	// `mcpServers` is reserved too: claude's current() deliberately skips that
	// settings.json key when reading managed values back (MCP servers are owned
	// via [mcps], projected into .claude.json). A settings.claude.mcpServers
	// value would be written on apply but never read back, so every plan would
	// re-propose it — a non-idempotent loop. Reject it up front instead.
	for k := range c.Settings.Claude {
		if err := validateKey("settings.claude", k); err != nil {
			return err
		}
		if k == "enabledPlugins" {
			return fmt.Errorf("parse config: settings.claude key %q is reserved (homonto manages plugins there); rename it", k)
		}
		if k == "mcpServers" {
			return fmt.Errorf("parse config: settings.claude key %q is reserved (homonto manages MCP servers via [mcps]); declare the server under [mcps] instead", k)
		}
		if k == "pluginConfigs" {
			return fmt.Errorf("parse config: settings.claude key %q is reserved (homonto manages pluginConfigs via [plugins.claude.<name>.config]); declare per-plugin config there instead", k)
		}
		if k == "extraKnownMarketplaces" {
			return fmt.Errorf("parse config: settings.claude key %q is reserved (homonto manages marketplaces via [marketplaces.claude.<name>]); declare the marketplace there instead", k)
		}
	}
	for k := range c.Settings.OpenCode {
		if err := validateKey("settings.opencode", k); err != nil {
			return err
		}
		if k == "mcp" || k == "plugin" {
			return fmt.Errorf("parse config: settings.opencode key %q is reserved (homonto manages %s there); rename it", k, k)
		}
	}
	// [tui.opencode] keys project into a second managed file (tui.json). Reject
	// index-like/empty names for the same JSON-array-corruption reason as
	// [settings.opencode].
	for k := range c.TUI.OpenCode {
		if err := validateKey("tui.opencode", k); err != nil {
			return err
		}
	}
	return nil
}

// Load reads, parses, and validates a homonto.toml file into a Config. It runs
// the decode → migrate → normalize → validate pipeline, failing closed on the
// first malformed or unsupported declaration.
// validateKey rejects names unusable as literal JSON object keys: empty, or
// index-like (all digits, or "-" followed by digits — sjson array semantics).
func validateKey(kind, name string) error {
	if name == "" {
		return fmt.Errorf("parse config: %s entry %q is empty", kind, name)
	}
	if indexLike(name) {
		return fmt.Errorf("parse config: %s entry %q would be treated as a JSON array index and corrupt the target file; rename it", kind, name)
	}
	return nil
}

// validateResources checks name, scope, source, and targets for every declared
// resource of a given kind (frameworks, skills, commands, subagents).
func validateResources(kind string, resources map[string]Resource) error {
	for name, r := range resources {
		if err := validateResourceName(kind, name); err != nil {
			return err
		}
		label := kind + "." + name
		switch r.Scope {
		case "user", "project":
			// ok
		case "":
			return fmt.Errorf("parse config: %s is missing required scope; valid values are \"user\" and \"project\"", label)
		default:
			return fmt.Errorf("parse config: %s scope %q is invalid; valid values are \"user\" and \"project\"", label, r.Scope)
		}
		if err := validateSource(label, r.Source, r.Digest, false); err != nil {
			return err
		}
		if err := validateLocalPlainName(label, r.Source); err != nil {
			return err
		}
		for _, target := range r.Targets {
			if !isResourceTarget(target) {
				return fmt.Errorf("parse config: %s targets unknown tool %q; valid targets are \"claude\" and \"opencode\"", label, target)
			}
		}
	}
	return nil
}

// validateFrameworkResources validates [frameworks.X] entries. A framework
// source must be builtin:<name> (expanded from the embedded catalog),
// local:<path> (a local framework root resolved relative to the config dir), or
// remote:<url> (a framework root fetched through the trust pipeline, which
// REQUIRES a sha256 digest pin). Unlike skills/commands, a local FRAMEWORK
// source MAY carry path components, so the plain-name guard is deliberately not
// applied here. A bare name or a typo expands nothing and is rejected loudly
// (F35); a digest on a builtin/local source is a no-op and rejected.
func validateFrameworkResources(resources map[string]Resource) error {
	for name, r := range resources {
		if err := validateResourceName("frameworks", name); err != nil {
			return err
		}
		label := "frameworks." + name
		switch r.Scope {
		case "user", "project":
			// ok
		case "":
			return fmt.Errorf("parse config: %s is missing required scope; valid values are \"user\" and \"project\"", label)
		default:
			return fmt.Errorf("parse config: %s scope %q is invalid; valid values are \"user\" and \"project\"", label, r.Scope)
		}
		// A remote: framework installs through the same trust pipeline as a remote
		// subagent, so it REQUIRES a valid sha256 digest pin (parsed here so a
		// malformed remote framework fails at load, mirroring remote subagents).
		// builtin:/local: keep their existing rule: a digest on them is a no-op and
		// rejected.
		if remote.IsRemoteSource(r.Source) {
			if _, err := remote.ParseRemoteSource(r.Source); err != nil {
				return fmt.Errorf("parse config: %s: %w", label, err)
			}
			if r.Digest == "" {
				return fmt.Errorf("parse config: %s remote source %q requires a digest = \"sha256:<hex>\" pin", label, r.Source)
			}
			if _, err := remote.ParseDigest(r.Digest); err != nil {
				return fmt.Errorf("parse config: %s: %w", label, err)
			}
			for _, target := range r.Targets {
				if !isResourceTarget(target) {
					return fmt.Errorf("parse config: %s targets unknown tool %q; valid targets are \"claude\" and \"opencode\"", label, target)
				}
			}
			continue
		}
		if r.Digest != "" {
			return fmt.Errorf("parse config: %s digest is only valid on a remote: source", label)
		}
		builtinOK := strings.HasPrefix(r.Source, "builtin:") && strings.TrimPrefix(r.Source, "builtin:") != ""
		localOK := strings.HasPrefix(r.Source, "local:") && strings.TrimPrefix(r.Source, "local:") != ""
		if !builtinOK && !localOK {
			return fmt.Errorf("parse config: %s source %q must be a builtin:<name>, local:<path>, or remote:<url> source (another source would expand nothing)", label, r.Source)
		}
		for _, target := range r.Targets {
			if !isResourceTarget(target) {
				return fmt.Errorf("parse config: %s targets unknown tool %q; valid targets are \"claude\" and \"opencode\"", label, target)
			}
		}
	}
	return nil
}

// validateLocalPlainName rejects a local: source that is not a plain name (no
// `.`/`..`/path separators), so it can never resolve/link/materialize a file
// outside the provider root. It is a no-op for non-local sources. Shared by
// validateResources (skills/commands) and validateSubagents so the two paths
// cannot drift.
func validateLocalPlainName(label, source string) error {
	src, ok := strings.CutPrefix(source, "local:")
	if !ok {
		return nil
	}
	if src == "" || src == "." || src == ".." || strings.ContainsAny(src, `/\`) || src != filepath.Base(src) {
		return fmt.Errorf("parse config: %s local source %q must be a plain name (no path components)", label, source)
	}
	return nil
}

// validateSubagents checks each [subagents.<name>]: a valid name, a builtin/local
// source, known targets, a user|project scope (already normalized to project when
// omitted), and a mode of link. copy is reserved for the forthcoming copy-mode
// projection and rejected until that lands, so the field is never a silent no-op.
func validateSubagents(subagents map[string]Subagent) error {
	for name, s := range subagents {
		if err := validateResourceName("subagents", name); err != nil {
			return err
		}
		label := "subagents." + name
		// A tune-only entry ([subagents.<name>.<tool>] with no source) retunes an
		// agent a framework already declared, so the declaration rules — source,
		// scope, local-name safety — are not its to satisfy. Its model blocks are
		// still validated, by validateSubagentOverrides.
		if s.IsTuneOnly() {
			continue
		}
		switch s.Scope {
		case "user", "project":
			// ok (empty was normalized to project at load)
		default:
			return fmt.Errorf("parse config: %s scope %q is invalid; valid values are \"user\" and \"project\"", label, s.Scope)
		}
		if err := validateSource(label, s.Source, s.Digest, true); err != nil {
			return err
		}
		// A local: source is resolved to a file by name; reject a path-traversal
		// name so it cannot read/link outside the provider root.
		if err := validateLocalPlainName(label, s.Source); err != nil {
			return err
		}
		for _, target := range s.Targets {
			if !isResourceTarget(target) {
				return fmt.Errorf("parse config: %s targets unknown tool %q; valid targets are \"claude\" and \"opencode\"", label, target)
			}
		}
		switch s.Mode {
		case "", "link", "copy":
			// ok — link projects a symlink, copy projects a managed content file
		default:
			return fmt.Errorf("parse config: %s mode %q is invalid; valid values are \"link\" and \"copy\"", label, s.Mode)
		}
	}
	return nil
}

// mcpTargetTools are valid MCP targets. codex is a pilot adapter that projects
// MCP servers only, so it is a valid MCP target but NOT a valid target for
// skills/commands/subagents/frameworks (which it cannot project, and which would
// otherwise demand an unsatisfiable models.codex.* route via validateModels).
var mcpTargetTools = []string{"claude", "opencode", "codex"}
var resourceTargetTools = []string{"claude", "opencode"}

func isMCPTarget(t string) bool      { return slices.Contains(mcpTargetTools, t) }
func isResourceTarget(t string) bool { return slices.Contains(resourceTargetTools, t) }

func validateResourceName(kind, name string) error {
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, `/\`) || name != filepath.Base(name) {
		return fmt.Errorf("parse config: %s entry %q is not a plain name", kind, name)
	}
	return validateKey(kind, name)
}

func validSource(source string) bool {
	for _, prefix := range []string{"builtin:", "local:"} {
		if strings.HasPrefix(source, prefix) && strings.TrimPrefix(source, prefix) != "" {
			return true
		}
	}
	return false
}

// validateSource accepts builtin:/local: sources unchanged, and a remote:
// source only when allowRemote is set (subagents only today), it parses, and it
// carries a well-formed sha256 digest pin. A non-remote source carrying a digest
// is rejected as unexpected so the field is never a silent no-op.
func validateSource(label, source, digest string, allowRemote bool) error {
	if remote.IsRemoteSource(source) {
		if !allowRemote {
			return fmt.Errorf("parse config: %s remote sources are only supported for subagents", label)
		}
		if _, err := remote.ParseRemoteSource(source); err != nil {
			return fmt.Errorf("parse config: %s: %w", label, err)
		}
		if digest == "" {
			return fmt.Errorf("parse config: %s remote source %q requires a digest = \"sha256:<hex>\" pin", label, source)
		}
		if _, err := remote.ParseDigest(digest); err != nil {
			return fmt.Errorf("parse config: %s: %w", label, err)
		}
		return nil
	}
	if digest != "" {
		return fmt.Errorf("parse config: %s digest is only valid on a remote: source", label)
	}
	if !validSource(source) {
		return fmt.Errorf("parse config: %s source %q is invalid; use builtin:<name>, local:<name>, or remote:<url>", label, source)
	}
	return nil
}

// The Claude effort/alias sets live in agentfm (the render is what actually
// speaks Claude's dialect); validation references the same maps so the two
// can never drift apart.
var (
	claudeEffortLevels = agentfm.ClaudeEffortLevels
	claudeModelAliases = agentfm.ClaudeAliases
)

// validateModelSpec checks one model/variant/effort triple against what `tool`
// can actually express, naming label as the offender. `model` is required only
// of a tier (a per-subagent override may set effort alone and inherit the rest),
// which requireModel selects.
//
// The tools differ, so the rules do:
//   - Claude renders a variant by bracketing an ALIAS (`opus[1m]`), and takes
//     `effort:` from a fixed set.
//   - OpenCode has a first-class `variant` field (any provider-defined string)
//     and no effort concept at all.
func validateModelSpec(tool, label string, r ModelRoute, requireModel bool) error {
	model := strings.TrimSpace(r.Model)
	variant := strings.TrimSpace(r.Variant)
	effort := strings.TrimSpace(r.Effort)
	if requireModel && model == "" {
		return fmt.Errorf("parse config: %s model is required", label)
	}
	switch tool {
	case "claude":
		if effort != "" && !claudeEffortLevels[effort] {
			return fmt.Errorf("parse config: %s effort %q is not a Claude effort level (low, medium, high, xhigh, max)", label, effort)
		}
		// Only meaningful against a model we can see; an override that sets a
		// variant alone is checked against the tier it merges into, below.
		if variant != "" && model != "" && !claudeModelAliases[model] {
			return fmt.Errorf("parse config: %s variant %q needs a model alias (opus, sonnet, haiku, fable, opusplan) — Claude takes no variant on the full model id %q", label, variant, model)
		}
	case "opencode":
		if effort != "" {
			return fmt.Errorf("parse config: %s sets effort %q, but OpenCode has no effort setting — use variant, or drop it", label, effort)
		}
	}
	return nil
}

// validateModels ensures every tool enabled by a non-skill resource declares all
// four model tiers with a model, and that every model/variant/effort value —
// tier or per-subagent override — is one the target tool can actually express.
//
// Effort and variant are OPTIONAL: a tier naming just a model is complete. They
// were once mandatory while being projected nowhere, which meant homonto forced
// you to write a field it then discarded — and never checked, so configs filled
// with values no tool accepts.
func validateModels(c *Config) error {
	// An unknown tier name ([models.opencode.reviewing], say) matches no agent
	// role and no default-model projection, so it would validate clean and then
	// do nothing — reject it naming the offender. agentfm.TierNames is the
	// single source of truth the role check in rendering uses too.
	for tool, routes := range map[string]map[string]ModelRoute{
		"claude":   c.Models.Claude,
		"opencode": c.Models.OpenCode,
	} {
		levels := make([]string, 0, len(routes))
		for level := range routes {
			levels = append(levels, level)
		}
		sort.Strings(levels) // deterministic: the same config must fail on the same offender
		for _, level := range levels {
			if !agentfm.Tiers[level] {
				return fmt.Errorf("parse config: models.%s.%s is not a model tier; valid tiers are %q, %q, %q, %q (agents pick one via their role)", tool, level, agentfm.TierNames[0], agentfm.TierNames[1], agentfm.TierNames[2], agentfm.TierNames[3])
			}
		}
	}
	for _, tool := range c.EnabledModelTools() {
		for _, level := range agentfm.TierNames {
			route, ok := modelRouteFor(c.Models, tool, level)
			label := "models." + tool + "." + level
			if !ok {
				return fmt.Errorf("parse config: %s is required for enabled target tool %q", label, tool)
			}
			if err := validateModelSpec(tool, label, route, true); err != nil {
				return err
			}
		}
	}
	return validateSubagentOverrides(c)
}

// validateSubagentOverrides checks every [subagents.<name>.<tool>] block —
// deliberately IGNORING the entry's targets, because the engine applies
// overrides unconditionally when it renders both tools' variants. The previous
// version iterated TargetsOrAll(), which let an untargeted tool's block skip
// validation entirely and stamp any value straight into a live agent file.
//
// It also rejects the two silent-no-op classes the review found: an override on
// a local:/remote: source (that content is never rendered, so the override can
// never apply), and a tune-only entry naming an agent that is not installed (a
// typo'd name would otherwise validate, plan, and apply clean while retuning
// nothing).
func validateSubagentOverrides(c *Config) error {
	names := make([]string, 0, len(c.Subagents))
	for name := range c.Subagents {
		names = append(names, name)
	}
	sort.Strings(names) // deterministic: the same config must fail on the same offender

	// The installed builtin agents, by catalog name — what a tune-only entry
	// must resolve against. Computed lazily: only configs that carry overrides
	// pay for the framework expansion.
	var installed map[string]bool
	installedBuiltins := func() (map[string]bool, error) {
		if installed != nil {
			return installed, nil
		}
		installed = map[string]bool{}
		for _, tool := range []string{"claude", "opencode"} {
			entries, err := c.ExpandedSubagentEntriesForTool(tool)
			if err != nil {
				return nil, err
			}
			for _, e := range entries {
				if cat, ok := SubagentCatalogName(e.Resource.Source); ok {
					installed[cat] = true
				}
			}
		}
		return installed, nil
	}

	seen := map[string]map[string]struct {
		entry string
		ov    ModelRoute
	}{} // catalog name -> tool -> first override seen
	for _, name := range names {
		sa := c.Subagents[name]
		hasOverride := sa.Claude != (ModelRoute{}) || sa.OpenCode != (ModelRoute{})
		if !hasOverride {
			continue
		}

		// Resolve the catalog name the override applies to. Overrides only make
		// sense for builtin (catalog-rendered) agents: local:/remote: content is
		// projected verbatim, so an override there would be accepted and then
		// silently discarded — reject it instead.
		cat := name
		if !sa.IsTuneOnly() {
			var ok bool
			if cat, ok = SubagentCatalogName(sa.Source); !ok {
				return fmt.Errorf("parse config: subagents.%s declares a model override, but its source %q is not builtin: — local:/remote: agents are projected verbatim and never rendered, so the override could never apply", name, sa.Source)
			}
		} else {
			known, err := installedBuiltins()
			if err != nil {
				return err
			}
			if !known[cat] {
				return fmt.Errorf("parse config: subagents.%s tunes an agent that is not installed — no framework or [subagents.*] declaration provides builtin:%s (typo?)", name, cat)
			}
		}

		for _, tool := range []string{"claude", "opencode"} {
			ov := sa.ModelOverrideFor(tool)
			if ov == (ModelRoute{}) {
				continue
			}
			label := "subagents." + name + "." + tool
			// Validate the fragment itself. A variant whose model comes from the
			// tier cannot be judged here — which tier depends on the agent's
			// frontmatter role, known only at render — so agentfm.Render errors
			// loudly on an unrenderable merged combination instead.
			if err := validateModelSpec(tool, label, ov, false); err != nil {
				return err
			}
			// Conflicts are judged per CATALOG name: one builtin renders one
			// file, so two entries' overrides for it must agree or the winner
			// would be map-iteration luck (a different render — and a different
			// materialize fingerprint — every run).
			if seen[cat] == nil {
				seen[cat] = map[string]struct {
					entry string
					ov    ModelRoute
				}{}
			}
			if prev, dup := seen[cat][tool]; dup && prev.ov != ov {
				return fmt.Errorf("parse config: subagents.%s.%s conflicts with subagents.%s.%s — one builtin (%s) renders one file, so its overrides must agree", name, tool, prev.entry, tool, cat)
			}
			seen[cat][tool] = struct {
				entry string
				ov    ModelRoute
			}{name, ov}
		}
	}
	return nil
}

func modelRouteFor(models ModelConfig, tool, level string) (ModelRoute, bool) {
	switch tool {
	case "claude":
		r, ok := models.Claude[level]
		return r, ok
	case "opencode":
		r, ok := models.OpenCode[level]
		return r, ok
	default:
		return ModelRoute{}, false
	}
}

// indexLike reports whether sjson would treat name as an array index:
// all-digit ("0", "42") or "-" followed by digits ("-1", the append form).
func indexLike(name string) bool {
	t := strings.TrimPrefix(name, "-")
	if t == "" {
		return false // "-" alone is a plain key
	}
	for i := 0; i < len(t); i++ {
		if t[i] < '0' || t[i] > '9' {
			return false
		}
	}
	return true
}
