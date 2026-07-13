package claude

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/noviopenworks/homonto/internal/adapter"
	"github.com/noviopenworks/homonto/internal/adapter/copyproj"
	"github.com/noviopenworks/homonto/internal/adapter/fileproj"
	"github.com/noviopenworks/homonto/internal/adapter/jsoncodec"
	"github.com/noviopenworks/homonto/internal/adapter/structproj"
	"github.com/noviopenworks/homonto/internal/commandpath"
	"github.com/noviopenworks/homonto/internal/config"
	"github.com/noviopenworks/homonto/internal/copyfile"
	"github.com/noviopenworks/homonto/internal/fsutil"
	"github.com/noviopenworks/homonto/internal/jsonutil"
	"github.com/noviopenworks/homonto/internal/secret"
	"github.com/noviopenworks/homonto/internal/skillpath"
	"github.com/noviopenworks/homonto/internal/state"
	"github.com/noviopenworks/homonto/internal/subagentpath"
)

// Adapter projects desired config into Claude Code's files under home.
type Adapter struct {
	home                string
	content             string
	catalogRoot         string // materialized builtin catalog root (.homonto/catalog/skills)
	commandCatalogRoot  string // materialized builtin command root (.homonto/catalog/commands)
	subagentCatalogRoot string // materialized builtin subagent root (.homonto/catalog/subagents)
	remoteSubagentRoot  string // materialized remote subagent root (.homonto/remote/subagents)
	projectRoot         string // directory of homonto.toml; used for project-scope resources
	skills              []config.NamedResource
	commands            []config.NamedResource
	subagents           []config.NamedResource
}

// New builds a Claude adapter at user scope. home is the $HOME root; content
// holds owned skills. Use WithProjectRoot to install project-scope skills.
func New(home, content string) *Adapter { return &Adapter{home: home, content: content} }

// WithProjectRoot sets the project root (the homonto.toml directory). It is
// used for project-scope resource placement. MCP servers and settings always
// project under home.
func (a *Adapter) WithProjectRoot(projectRoot string) *Adapter {
	a.projectRoot = projectRoot
	return a
}

// WithCatalogRoot sets the materialized builtin-catalog root that builtin:<name>
// skills link from. Mirrors WithProjectRoot.
func (a *Adapter) WithCatalogRoot(catalogRoot string) *Adapter {
	a.catalogRoot = catalogRoot
	return a
}

// WithCommandCatalogRoot sets the materialized builtin-command root that
// builtin:<name> commands link from. Mirrors WithCatalogRoot.
func (a *Adapter) WithCommandCatalogRoot(commandCatalogRoot string) *Adapter {
	a.commandCatalogRoot = commandCatalogRoot
	return a
}

// WithSubagentCatalogRoot sets the materialized builtin-subagent root that
// builtin:<name> subagents link from. Mirrors WithCommandCatalogRoot.
func (a *Adapter) WithSubagentCatalogRoot(subagentCatalogRoot string) *Adapter {
	a.subagentCatalogRoot = subagentCatalogRoot
	return a
}

// WithRemoteSubagentRoot sets the materialized remote-subagent root that
// remote:<url> subagents link from (populated by the engine's verify pipeline
// before apply). Mirrors WithSubagentCatalogRoot.
func (a *Adapter) WithRemoteSubagentRoot(remoteSubagentRoot string) *Adapter {
	a.remoteSubagentRoot = remoteSubagentRoot
	return a
}

// managedRoots returns every content root homonto owns links into. catalogRoot,
// commandCatalogRoot, and subagentCatalogRoot are included only when set:
// link.managed() treats an empty-string root as a prefix match for every
// absolute path, so passing "" here would make link calls treat any symlink as
// "ours" — an empty root must never reach link.*.
func (a *Adapter) managedRoots() []string {
	roots := []string{a.content}
	if a.catalogRoot != "" {
		roots = append(roots, a.catalogRoot)
	}
	if a.commandCatalogRoot != "" {
		roots = append(roots, a.commandCatalogRoot)
	}
	if a.subagentCatalogRoot != "" {
		roots = append(roots, a.subagentCatalogRoot)
	}
	if a.remoteSubagentRoot != "" {
		roots = append(roots, a.remoteSubagentRoot)
	}
	return roots
}

// skillSource resolves a skill entry's on-disk content directory by source
// scheme: builtin:<n> from the materialized catalog root, otherwise the local
// content dir.
func (a *Adapter) skillSource(entry config.NamedResource) string {
	if s := entry.Resource.Source; strings.HasPrefix(s, "builtin:") {
		return filepath.Join(a.catalogRoot, strings.TrimPrefix(s, "builtin:"))
	}
	return filepath.Join(a.content, "skills", localSourceName(entry.Resource.Source, entry.Name))
}

func (a *Adapter) Name() string { return "claude" }

func (a *Adapter) claudeJSON() string   { return filepath.Join(a.home, ".claude.json") }
func (a *Adapter) settingsJSON() string { return filepath.Join(a.home, ".claude", "settings.json") }

// skillsDir is the directory owned-skill symlinks live in for the given scope.
func (a *Adapter) skillsDir(scope string) string {
	return skillpath.Dir("claude", scope, a.home, a.projectRoot)
}

// inactiveSkillsDir is the other scope's skills directory — where a link may
// linger after a per-resource scope switch. It returns "" when there is nothing
// meaningful to relocate from: no project root is known, or the two scopes
// resolve to the same directory (a homonto.toml that sits in $HOME).
func (a *Adapter) inactiveSkillsDir(scope string) string {
	if a.projectRoot == "" {
		return ""
	}
	d := skillpath.Dir("claude", skillpath.Other(scope), a.home, a.projectRoot)
	if d == a.skillsDir(scope) {
		return ""
	}
	return d
}

// skillFileLinks builds the desired managed skill symlinks for the fileproj
// contract: destination, content source, state key, and the same-named link at
// the other scope (Inactive is "" when there is nothing to relocate from).
func (a *Adapter) skillFileLinks() []fileproj.Link {
	var out []fileproj.Link
	for _, e := range a.skills {
		inact := ""
		if d := a.inactiveSkillsDir(e.Resource.Scope); d != "" {
			inact = filepath.Join(d, e.Name)
		}
		out = append(out, fileproj.Link{
			Dst:      filepath.Join(a.skillsDir(e.Resource.Scope), e.Name),
			Src:      a.skillSource(e),
			Key:      "skill." + e.Name,
			Inactive: inact,
		})
	}
	return out
}

// commandsDir is the directory owned-command symlinks live in for the scope.
func (a *Adapter) commandsDir(scope string) string {
	return commandpath.Dir("claude", scope, a.home, a.projectRoot)
}

// inactiveCommandsDir is the other scope's commands directory — where a link
// may linger after a per-resource scope switch. It returns "" when nothing
// meaningful can be relocated (no project root, or both scopes resolve equal).
func (a *Adapter) inactiveCommandsDir(scope string) string {
	if a.projectRoot == "" {
		return ""
	}
	d := commandpath.Dir("claude", skillpath.Other(scope), a.home, a.projectRoot)
	if d == a.commandsDir(scope) {
		return ""
	}
	return d
}

// commandSource resolves a command entry's on-disk file by source scheme:
// builtin:<n> from the materialized command root (<n>.md), otherwise the local
// content dir (homonto/commands/<n>.md).
func (a *Adapter) commandSource(entry config.NamedResource) string {
	if s := entry.Resource.Source; strings.HasPrefix(s, "builtin:") {
		return filepath.Join(a.commandCatalogRoot, strings.TrimPrefix(s, "builtin:")+".md")
	}
	return filepath.Join(a.content, "commands", localSourceName(entry.Resource.Source, entry.Name)+".md")
}

// commandFileLinks builds the desired managed command symlinks for the fileproj
// contract (destination is <name>.md).
func (a *Adapter) commandFileLinks() []fileproj.Link {
	var out []fileproj.Link
	for _, e := range a.commands {
		inact := ""
		if d := a.inactiveCommandsDir(e.Resource.Scope); d != "" {
			inact = filepath.Join(d, e.Name+".md")
		}
		out = append(out, fileproj.Link{
			Dst:      filepath.Join(a.commandsDir(e.Resource.Scope), e.Name+".md"),
			Src:      a.commandSource(e),
			Key:      "command." + e.Name,
			Inactive: inact,
		})
	}
	return out
}

// subagentsDir is the directory owned-subagent symlinks live in for the scope.
func (a *Adapter) subagentsDir(scope string) string {
	return subagentpath.Dir("claude", scope, a.home, a.projectRoot)
}

// inactiveSubagentsDir is the other scope's subagent directory — where a link
// may linger after a per-resource scope switch. It returns "" when nothing
// meaningful can be relocated (no project root, or both scopes resolve equal).
func (a *Adapter) inactiveSubagentsDir(scope string) string {
	if a.projectRoot == "" {
		return ""
	}
	d := subagentpath.Dir("claude", skillpath.Other(scope), a.home, a.projectRoot)
	if d == a.subagentsDir(scope) {
		return ""
	}
	return d
}

// subagentSource resolves a subagent entry's on-disk file by source scheme:
// builtin:<n> from the materialized subagent root (<n>.md), remote:<url> from the
// materialized remote root (<name>.md), otherwise the local content dir
// (homonto/subagents/<n>.md).
func (a *Adapter) subagentSource(entry config.NamedResource) string {
	if s := entry.Resource.Source; strings.HasPrefix(s, "builtin:") {
		return filepath.Join(a.subagentCatalogRoot, strings.TrimPrefix(s, "builtin:")+".md")
	}
	if strings.HasPrefix(entry.Resource.Source, "remote:") {
		return filepath.Join(a.remoteSubagentRoot, entry.Name+".md")
	}
	return filepath.Join(a.content, "subagents", localSourceName(entry.Resource.Source, entry.Name)+".md")
}

// subagentFileLinks builds the desired managed subagent symlinks for the
// fileproj contract. Copy-mode subagents are projected as content files (not
// links), so they are skipped here and reconciled by applyCopySubagents.
func (a *Adapter) subagentFileLinks() []fileproj.Link {
	var out []fileproj.Link
	for _, e := range a.subagents {
		if e.Mode == "copy" {
			continue
		}
		inact := ""
		if d := a.inactiveSubagentsDir(e.Resource.Scope); d != "" {
			inact = filepath.Join(d, e.Name+".md")
		}
		out = append(out, fileproj.Link{
			Dst:      filepath.Join(a.subagentsDir(e.Resource.Scope), e.Name+".md"),
			Src:      a.subagentSource(e),
			Key:      "subagent." + e.Name,
			Inactive: inact,
		})
	}
	return out
}

// copySubagentDesired returns dst -> resolved content for each copy-mode
// subagent (a real managed file rather than a symlink).
func (a *Adapter) copySubagentDesired() (map[string][]byte, error) {
	out := map[string][]byte{}
	for _, entry := range a.subagents {
		if entry.Mode != "copy" {
			continue
		}
		content, err := os.ReadFile(a.subagentSource(entry))
		if err != nil {
			return nil, err
		}
		out[filepath.Join(a.subagentsDir(entry.Resource.Scope), entry.Name+".md")] = content
	}
	return out, nil
}

// planCopyOps computes the reconciler ops for copy-mode subagents against state
// through the shared copyproj core.
func (a *Adapter) planCopyOps(st *state.State) ([]copyfile.Op, error) {
	desired, err := a.copySubagentDesired()
	if err != nil {
		return nil, err
	}
	return copyproj.Plan("claude", desired, st)
}

// applyCopySubagents reconciles copy-mode subagent content files through the
// shared copyproj core (write/update/prune + local-edit .bak backup + state,
// conflict abort, F7 prune-root guard).
func (a *Adapter) applyCopySubagents(st *state.State) error {
	desired, err := a.copySubagentDesired()
	if err != nil {
		return err
	}
	return copyproj.Apply("claude", desired, st, a.copyPruneRoots())
}

// copyPruneRoots are the directories a copy-mode subagent file may legitimately
// live in (user + project agent dirs). copyfile.Apply refuses to delete a prune
// destination — reconstructed from an untrusted state entry — that resolves
// outside these roots, so a tampered state.json path cannot delete an arbitrary
// file (F7). The project dir is included only when a project root is known.
func (a *Adapter) copyPruneRoots() []string {
	roots := []string{a.subagentsDir("user")}
	if a.projectRoot != "" {
		roots = append(roots, a.subagentsDir("project"))
	}
	return roots
}

// localSourceName resolves a skill resource's content subdirectory: a local:
// source names that directory directly; any other source falls back to the
// skill's declared name.
func localSourceName(source, fallback string) string {
	if strings.HasPrefix(source, "local:") {
		return strings.TrimPrefix(source, "local:")
	}
	return fallback
}

// desired returns managed key -> unresolved JSON-encoded desired value.
func (a *Adapter) desired(c *config.Config) map[string]string {
	out := map[string]string{}
	for name, m := range c.MCPs {
		if !contains(m.TargetsOrAll(), "claude") {
			continue
		}
		if len(m.Command) == 0 {
			continue
		}
		// Claude Code's real schema: command is a string with a separate args
		// array (matching `claude mcp add` output; empty keys omitted).
		obj := map[string]any{"type": "stdio", "command": m.Command[0]}
		if len(m.Command) > 1 {
			obj["args"] = m.Command[1:]
		}
		if len(m.Env) > 0 {
			obj["env"] = m.Env
		}
		out["mcp."+name] = mustJSON(obj)
	}
	for k, v := range c.Settings.Claude {
		out["setting."+k] = mustJSON(v)
	}
	for _, pl := range c.Plugins.Claude {
		// Source-keyed: enabledPlugins[<source>] carries the plugin's enabled
		// value, so a disabled plugin emits a managed `false` (not absence).
		out["plugin."+pl.Source] = mustJSON(pl.IsEnabled())
	}
	for _, pl := range c.Plugins.Claude {
		// pluginConfigs[<source>] carries the whole {options:…} object so write
		// and read-back stay symmetric (see current()); no config → no key.
		if len(pl.Config) > 0 {
			out["pluginconfig."+pl.Source] = mustJSON(map[string]any{"options": pl.Config})
		}
	}
	for name, mk := range c.Marketplaces.Claude {
		// extraKnownMarketplaces[<name>] carries the whole {source:…} object so
		// write and read-back stay symmetric (see current()).
		out["marketplace."+name] = mustJSON(marketplaceValue(mk))
	}
	return out
}

// marketplaceValue builds the canonical extraKnownMarketplaces value for a
// declared marketplace. Only the type-relevant locator fields are emitted, so a
// github marketplace never carries an empty url/path that would differ from an
// adopted on-disk entry. autoUpdate is present only when auto_update was set.
func marketplaceValue(mk config.Marketplace) map[string]any {
	src := map[string]any{"source": mk.Source}
	switch mk.Source {
	case "github":
		src["repo"] = mk.Repo
	case "url":
		src["url"] = mk.URL
	case "git-subdir":
		src["url"] = mk.URL
		src["path"] = mk.Path
	case "directory":
		src["path"] = mk.Path
	}
	out := map[string]any{"source": src}
	if mk.AutoUpdate != nil {
		out["autoUpdate"] = *mk.AutoUpdate
	}
	return out
}

// Document-path mappings for each structured-document namespace, threaded into
// structproj.Project/Apply/Observe. Config-supplied names are escaped so a name
// containing dots/@/|/# addresses the literal key rather than nesting or being
// dropped (mirroring the prior inline SetJSON/GetJSON escaping).
func mcpPath(key string) string     { return "mcpServers." + jsonutil.EscapePath(trim(key, "mcp.")) }
func settingPath(key string) string { return jsonutil.EscapePath(trim(key, "setting.")) }
func pluginPath(key string) string {
	return "enabledPlugins." + jsonutil.EscapePath(trim(key, "plugin."))
}
func pluginConfigPath(key string) string {
	return "pluginConfigs." + jsonutil.EscapePath(trim(key, "pluginconfig."))
}
func marketplacePath(key string) string {
	return "extraKnownMarketplaces." + jsonutil.EscapePath(trim(key, "marketplace."))
}

func (a *Adapter) Plan(c *config.Config, st *state.State) (adapter.ChangeSet, error) {
	skills, err := c.ExpandedSkillEntriesForTool("claude")
	if err != nil {
		return adapter.ChangeSet{}, err
	}
	a.skills = skills
	commands, err := c.ExpandedCommandEntriesForTool("claude")
	if err != nil {
		return adapter.ChangeSet{}, err
	}
	a.commands = commands
	subagents, err := c.ExpandedSubagentEntriesForTool("claude")
	if err != nil {
		return adapter.ChangeSet{}, err
	}
	a.subagents = subagents
	mj, err := readStandardized(a.claudeJSON())
	if err != nil {
		return adapter.ChangeSet{}, err
	}
	sj, err := readStandardized(a.settingsJSON())
	if err != nil {
		return adapter.ChangeSet{}, err
	}
	cs := adapter.ChangeSet{Tool: "claude"}
	des := a.desired(c)
	codec := jsoncodec.Codec{}
	// Structured-document namespaces go through the shared projection contract:
	// mcp.* lives in .claude.json; setting./plugin./pluginconfig./marketplace.*
	// all live in settings.json. Each Project call sees only its prefix's desired
	// keys and prunes only its own recorded keys, so the generic delete loop below
	// no longer touches these prefixes.
	cs.Changes = append(cs.Changes, structproj.Project("claude", "mcp.", filterDesired(des, "mcp."), mj, st, codec, mcpPath)...)
	cs.Changes = append(cs.Changes, structproj.Project("claude", "setting.", filterDesired(des, "setting."), sj, st, codec, settingPath)...)
	cs.Changes = append(cs.Changes, structproj.Project("claude", "plugin.", filterDesired(des, "plugin."), sj, st, codec, pluginPath)...)
	cs.Changes = append(cs.Changes, structproj.Project("claude", "pluginconfig.", filterDesired(des, "pluginconfig."), sj, st, codec, pluginConfigPath)...)
	cs.Changes = append(cs.Changes, structproj.Project("claude", "marketplace.", filterDesired(des, "marketplace."), sj, st, codec, marketplacePath)...)
	// File-projection namespaces go through the shared symlink contract: each
	// Project call emits create/relocate/relink + adopt for its links and plans
	// NO deletes — the generic delete loop below stays the single source of
	// file-prefix deletes.
	roots := a.managedRoots()
	skillChanges, err := fileproj.Project("claude", a.skillFileLinks(), st, roots)
	if err != nil {
		return adapter.ChangeSet{}, err
	}
	cs.Changes = append(cs.Changes, skillChanges...)
	commandChanges, err := fileproj.Project("claude", a.commandFileLinks(), st, roots)
	if err != nil {
		return adapter.ChangeSet{}, err
	}
	cs.Changes = append(cs.Changes, commandChanges...)
	subagentChanges, err := fileproj.Project("claude", a.subagentFileLinks(), st, roots)
	if err != nil {
		return adapter.ChangeSet{}, err
	}
	cs.Changes = append(cs.Changes, subagentChanges...)
	// Orphans: a state key no longer declared in config is de-declared — plan a
	// delete. (A declared key missing from disk is drift, handled above.) Old is
	// always redacted: a removed key's provenance is stale by definition.
	declared := map[string]bool{}
	for k := range des {
		declared[k] = true
	}
	for _, entry := range a.skills {
		declared["skill."+entry.Name] = true
	}
	for _, entry := range a.commands {
		declared["command."+entry.Name] = true
	}
	for _, entry := range a.subagents {
		declared["subagent."+entry.Name] = true
	}
	// Copy-mode subagents are managed content files (not symlinks): surface their
	// create/update/prune in the plan and abort on a foreign-file conflict. Apply
	// reconciles them in a dedicated pass (a.applyCopySubagents); subagentcopy.* is
	// deliberately outside filePrefix so the generic delete loop never touches
	// it.
	copyOps, err := a.planCopyOps(st)
	if err != nil {
		return adapter.ChangeSet{}, err
	}
	for _, op := range copyOps {
		name := copyproj.Name(op.Dst)
		switch op.Action {
		case copyfile.Conflict:
			return adapter.ChangeSet{}, fmt.Errorf("claude: %s exists and is not a homonto-managed copy-mode subagent; not overwriting", op.Dst)
		case copyfile.Create:
			cs.Changes = append(cs.Changes, adapter.Change{Action: "create", Key: "subagentcopy." + name, New: op.Dst})
		case copyfile.Update, copyfile.LocalEdit:
			cs.Changes = append(cs.Changes, adapter.Change{Action: "update", Key: "subagentcopy." + name, New: op.Dst})
		case copyfile.Prune:
			cs.Changes = append(cs.Changes, adapter.Change{Action: "delete", Key: "subagentcopy." + name, Old: op.Dst})
		}
	}
	// The generic prune covers only the file-projection prefixes; the structured
	// prefixes are pruned by their structproj.Project calls above (avoiding a
	// double delete). subagentcopy.* is pruned by its own reconciler pass.
	for _, k := range st.Keys("claude") {
		if declared[k] || !filePrefix(k) {
			continue
		}
		cs.Changes = append(cs.Changes, adapter.Change{Action: "delete", Key: k, Old: adapter.SecretRedaction})
	}
	// Keys come from map iteration (random order); a plan must render the
	// same way every run. Keys are unique within a changeset.
	sort.SliceStable(cs.Changes, func(i, j int) bool { return cs.Changes[i].Key < cs.Changes[j].Key })
	return cs, nil
}

// ObserveHashes hashes the current on-disk value of every recorded key still
// present, so an unchanged key reproduces its Entry.Applied (see the plan's
// noop identity: Applied == secret.Hash(jsonutil.Canonical(disk))). Only hashes
// escape — raw values (possibly resolved secrets) never leave the adapter.
func (a *Adapter) ObserveHashes(st *state.State) (map[string]string, error) {
	mj, err := readStandardized(a.claudeJSON())
	if err != nil {
		return nil, err
	}
	sj, err := readStandardized(a.settingsJSON())
	if err != nil {
		return nil, err
	}
	codec := jsoncodec.Codec{}
	out := map[string]string{}
	// Structured-document keys (mcp.* in .claude.json; setting./plugin./
	// pluginconfig./marketplace.* in settings.json) re-hash through the contract.
	for k, v := range structproj.Observe("claude", "mcp.", mj, st, codec, mcpPath) {
		out[k] = v
	}
	for _, o := range []struct {
		prefix  string
		pathFor structproj.PathFor
	}{
		{"setting.", settingPath},
		{"plugin.", pluginPath},
		{"pluginconfig.", pluginConfigPath},
		{"marketplace.", marketplacePath},
	} {
		for k, v := range structproj.Observe("claude", o.prefix, sj, st, codec, o.pathFor) {
			out[k] = v
		}
	}
	// File-projection keys (skill./command./subagent.*) live on disk as symlinks;
	// each re-hashes its recorded link through the shared contract, reading at the
	// recorded dst so a pending scope switch is not misread as drift.
	for k, v := range fileproj.Observe("claude", "skill.", st) {
		out[k] = v
	}
	for k, v := range fileproj.Observe("claude", "command.", st) {
		out[k] = v
	}
	for k, v := range fileproj.Observe("claude", "subagent.", st) {
		out[k] = v
	}
	for _, key := range st.Keys("claude") {
		if !hasPrefix(key, "subagentcopy.") {
			continue
		}
		// A copy-mode subagent lives on disk as a real file; its Applied is the
		// content hash and Desired holds the dst path.
		e, ok := st.Get("claude", key)
		if !ok {
			continue
		}
		content, err := os.ReadFile(e.Desired)
		if err != nil {
			continue // missing → omit (engine infers missing)
		}
		out[key] = copyfile.Hash(content)
	}
	return out, nil
}

func (a *Adapter) Apply(cs adapter.ChangeSet, res *secret.Resolver, st *state.State) error {
	mj, err := readStandardized(a.claudeJSON())
	if err != nil {
		return err
	}
	sj, err := readStandardized(a.settingsJSON())
	if err != nil {
		return err
	}
	// Write a tool file only when a managed key living in it actually changed.
	// adopt/noop are state-only and must leave the file byte-for-byte untouched
	// (comments/formatting preserved); skill.* is symlink work, not JSON.
	codec := jsoncodec.Codec{}
	// Structured-document prefixes go through the shared contract: mcp.* lives in
	// .claude.json; setting./plugin./pluginconfig./marketplace.* all live in
	// settings.json (threaded through one doc so a change to any of them writes the
	// file exactly once). Each Apply reports whether its document actually changed.
	mj, mjChanged, err := structproj.Apply("claude", "mcp.", filterChanges(cs.Changes, "mcp."), mj, codec, res, st, mcpPath)
	if err != nil {
		return err
	}
	sjChanged := false
	// Prefixes are applied in lexicographic key order (marketplace < plugin <
	// pluginconfig < setting) so newly-created keys are appended to settings.json
	// in the same order the prior single sorted-change loop produced — preserving
	// byte-for-byte output.
	for _, p := range []struct {
		prefix  string
		pathFor structproj.PathFor
	}{
		{"marketplace.", marketplacePath},
		{"plugin.", pluginPath},
		{"pluginconfig.", pluginConfigPath},
		{"setting.", settingPath},
	} {
		var ch bool
		sj, ch, err = structproj.Apply("claude", p.prefix, filterChanges(cs.Changes, p.prefix), sj, codec, res, st, p.pathFor)
		if err != nil {
			return err
		}
		sjChanged = sjChanged || ch
	}
	// File-projection keys (skill./command./subagent.): adopt records state only;
	// delete removes the managed symlink. Their create/update are handled by the
	// fileproj.ApplyLinks pass below; noop and subagentcopy.* are handled
	// elsewhere. The fallback recovers a de-declared key's on-disk dst at user
	// scope when state lacks a recorded dst, matching the prior inline behavior.
	roots := a.managedRoots()
	if err := fileproj.ApplyState("claude", filterChanges(cs.Changes, "skill."), st, roots, func(k string) string {
		return filepath.Join(a.skillsDir("user"), trim(k, "skill."))
	}); err != nil {
		return err
	}
	if err := fileproj.ApplyState("claude", filterChanges(cs.Changes, "command."), st, roots, func(k string) string {
		return filepath.Join(a.commandsDir("user"), trim(k, "command.")+".md")
	}); err != nil {
		return err
	}
	if err := fileproj.ApplyState("claude", filterChanges(cs.Changes, "subagent."), st, roots, func(k string) string {
		return filepath.Join(a.subagentsDir("user"), trim(k, "subagent.")+".md")
	}); err != nil {
		return err
	}
	// Fail fast on link conflicts before writing any file. A conflict in any of
	// the three namespaces must be detected here, before any JSON write or state
	// mutation below — otherwise a command conflict could let Apply partially
	// write JSON and commit skill-link state before erroring.
	if err := fileproj.Conflicts(a.skillFileLinks(), roots); err != nil {
		return err
	}
	if err := fileproj.Conflicts(a.commandFileLinks(), roots); err != nil {
		return err
	}
	if err := fileproj.Conflicts(a.subagentFileLinks(), roots); err != nil {
		return err
	}
	// Fail fast on a copy-mode subagent conflict too, before any file is written.
	copyOps, err := a.planCopyOps(st)
	if err != nil {
		return err
	}
	for _, op := range copyOps {
		if op.Action == copyfile.Conflict {
			return fmt.Errorf("claude: %s exists and is not a homonto-managed copy-mode subagent; not overwriting", op.Dst)
		}
	}
	if mjChanged {
		if err := fsutil.WriteAtomic(a.claudeJSON(), mj); err != nil {
			return err
		}
	}
	if sjChanged {
		if err := fsutil.WriteAtomic(a.settingsJSON(), sj); err != nil {
			return err
		}
	}
	// Prune each namespace's inactive-scope orphan (left after a per-resource
	// scope switch), then create the link and record state. Runs after the JSON
	// writes. Only our own managed symlink is ever removed (IsManaged guards it);
	// a foreign file or an absent path is left untouched.
	if err := fileproj.ApplyLinks("claude", a.skillFileLinks(), st, roots); err != nil {
		return err
	}
	if err := fileproj.ApplyLinks("claude", a.commandFileLinks(), st, roots); err != nil {
		return err
	}
	if err := fileproj.ApplyLinks("claude", a.subagentFileLinks(), st, roots); err != nil {
		return err
	}
	// Reconcile copy-mode subagent content files (write/update/prune + state),
	// backing up any local edit. Conflicts were already rejected above.
	if err := a.applyCopySubagents(st); err != nil {
		return err
	}
	return nil
}

func readStandardized(path string) ([]byte, error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return jsonutil.Standardize(nil)
	}
	if err != nil {
		return nil, err
	}
	doc, err := jsonutil.Standardize(b)
	if err != nil {
		return nil, err
	}
	if err := jsonutil.ObjectRoot(doc); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return doc, nil
}
