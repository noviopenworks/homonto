package claude

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/noviopenworks/homonto/internal/adapter"
	"github.com/noviopenworks/homonto/internal/adapter/jsoncodec"
	"github.com/noviopenworks/homonto/internal/adapter/structproj"
	"github.com/noviopenworks/homonto/internal/commandpath"
	"github.com/noviopenworks/homonto/internal/config"
	"github.com/noviopenworks/homonto/internal/copyfile"
	"github.com/noviopenworks/homonto/internal/fsutil"
	"github.com/noviopenworks/homonto/internal/jsonutil"
	"github.com/noviopenworks/homonto/internal/link"
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

// links maps each owned skill's destination to its content source. Each skill
// resource carries its own scope, so dst is computed per entry.
func (a *Adapter) links() map[string]string {
	out := map[string]string{}
	for _, entry := range a.skills {
		out[filepath.Join(a.skillsDir(entry.Resource.Scope), entry.Name)] = a.skillSource(entry)
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

// commandLinks maps each owned command's destination (<name>.md) to its source.
func (a *Adapter) commandLinks() map[string]string {
	out := map[string]string{}
	for _, entry := range a.commands {
		out[filepath.Join(a.commandsDir(entry.Resource.Scope), entry.Name+".md")] = a.commandSource(entry)
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

// subagentLinks maps each owned subagent's destination (<name>.md) to its source.
func (a *Adapter) subagentLinks() map[string]string {
	out := map[string]string{}
	for _, entry := range a.subagents {
		if entry.Mode == "copy" {
			continue // copy-mode subagents are projected as content files, not links
		}
		out[filepath.Join(a.subagentsDir(entry.Resource.Scope), entry.Name+".md")] = a.subagentSource(entry)
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

// recordedCopyHashes returns dst -> recorded content hash for every
// subagentcopy.* key in state (Desired holds the dst, Applied the content hash).
func recordedCopyHashes(st *state.State, tool string) map[string]string {
	out := map[string]string{}
	for _, key := range st.Keys(tool) {
		if !strings.HasPrefix(key, "subagentcopy.") {
			continue
		}
		if e, ok := st.Get(tool, key); ok {
			out[e.Desired] = e.Applied
		}
	}
	return out
}

// copySubagentName recovers the subagent name from a managed copy-file dst.
func copySubagentName(dst string) string {
	return strings.TrimSuffix(filepath.Base(dst), ".md")
}

// planCopyOps computes the reconciler ops for copy-mode subagents against state.
func (a *Adapter) planCopyOps(st *state.State) ([]copyfile.Op, error) {
	desired, err := a.copySubagentDesired()
	if err != nil {
		return nil, err
	}
	return copyfile.Plan(desired, recordedCopyHashes(st, "claude"))
}

// applyCopySubagents reconciles copy-mode subagent content files: it writes
// created/updated files, prunes de-declared ones, and backs up any local edit to
// <dst>.bak before overwriting or pruning (never losing a user's edit) — the
// pre-merge behavior; three-way merge replaces the backup+overwrite later. A
// destination occupied by a foreign file or a symlink is a conflict and aborts.
func (a *Adapter) applyCopySubagents(st *state.State) error {
	ops, err := a.planCopyOps(st)
	if err != nil {
		return err
	}
	for i, op := range ops {
		switch op.Action {
		case copyfile.Conflict:
			return fmt.Errorf("claude: %s exists and is not a homonto-managed copy-mode subagent; not overwriting", op.Dst)
		case copyfile.LocalEdit:
			if err := fsutil.WriteAtomic(op.Dst+".bak", op.OnDisk); err != nil {
				return err
			}
			if op.Content == nil {
				ops[i].Action = copyfile.Prune // de-declared + edited: backed up, now remove
			} else {
				ops[i].Action = copyfile.Update // declared + edited: backed up, now overwrite
			}
		}
	}
	rec, pruned, _, err := copyfile.Apply(ops, a.copyPruneRoots())
	if err != nil {
		return err
	}
	for dst, h := range rec {
		st.Set("claude", "subagentcopy."+copySubagentName(dst), dst, h)
	}
	// Refused prunes (dst outside the managed root — a tampered state entry) are
	// deliberately NOT in `pruned`, so their ownership record is retained rather
	// than dropped and the out-of-root file is never deleted.
	for _, dst := range pruned {
		st.Delete("claude", "subagentcopy."+copySubagentName(dst))
	}
	return nil
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
	ops, err := link.Plan(a.links(), a.managedRoots()...)
	if err != nil {
		return adapter.ChangeSet{}, err
	}
	entryByName := map[string]config.NamedResource{}
	for _, entry := range a.skills {
		entryByName[entry.Name] = entry
	}
	for _, op := range ops {
		name := filepath.Base(op.Dst)
		entry := entryByName[name]
		inactive := a.inactiveSkillsDir(entry.Resource.Scope)
		// A create whose same-named link still exists (as our managed symlink) at
		// the other scope is a scope switch: render it as a relocate so the move —
		// and the prune of the old link Apply performs — is visible before confirm.
		if op.Cur == "" && inactive != "" && link.IsManaged(filepath.Join(inactive, name), a.managedRoots()...) {
			cs.Changes = append(cs.Changes, adapter.Change{Action: "update", Key: "skill." + name, Old: filepath.Join(inactive, name), New: op.Dst + " -> " + op.Src})
		} else if op.Cur == "" {
			cs.Changes = append(cs.Changes, adapter.Change{Action: "create", Key: "skill." + name, New: op.Dst + " -> " + op.Src})
		} else {
			cs.Changes = append(cs.Changes, adapter.Change{Action: "update", Key: "skill." + name, Old: op.Cur, New: op.Src})
		}
	}
	// Adopt a correct-but-unrecorded skill link — one already on disk pointing at
	// its content, but absent from state (or stale). link.Plan omits a correct
	// link, so without this a lost state.json for a skills-only config could never
	// be rebuilt (apply short-circuits with no change). Mirrors mcp/setting/plugin
	// adoption: state-only, the on-disk link is left untouched.
	opDst := map[string]bool{}
	for _, op := range ops {
		opDst[op.Dst] = true
	}
	for _, entry := range a.skills {
		name := entry.Name
		dst := filepath.Join(a.skillsDir(entry.Resource.Scope), name)
		if opDst[dst] {
			continue // a create/relink/relocate already covers it
		}
		src := a.skillSource(entry)
		if tgt, err := os.Readlink(dst); err != nil || tgt != src {
			continue // not a correct link into content
		}
		if e, ok := st.Get("claude", "skill."+name); ok && e.Applied == secret.Hash(dst+" -> "+src) {
			continue // already recorded → a true noop, nothing to do
		}
		cs.Changes = append(cs.Changes, adapter.Change{Action: "adopt", Key: "skill." + name, New: dst + " -> " + src})
	}
	// ---- command links (parallel to skills) ----
	cmdOps, err := link.Plan(a.commandLinks(), a.managedRoots()...)
	if err != nil {
		return adapter.ChangeSet{}, err
	}
	cmdByName := map[string]config.NamedResource{}
	for _, entry := range a.commands {
		cmdByName[entry.Name] = entry
	}
	for _, op := range cmdOps {
		name := strings.TrimSuffix(filepath.Base(op.Dst), ".md")
		entry := cmdByName[name]
		inactive := a.inactiveCommandsDir(entry.Resource.Scope)
		if op.Cur == "" && inactive != "" && link.IsManaged(filepath.Join(inactive, name+".md"), a.managedRoots()...) {
			cs.Changes = append(cs.Changes, adapter.Change{Action: "update", Key: "command." + name, Old: filepath.Join(inactive, name+".md"), New: op.Dst + " -> " + op.Src})
		} else if op.Cur == "" {
			cs.Changes = append(cs.Changes, adapter.Change{Action: "create", Key: "command." + name, New: op.Dst + " -> " + op.Src})
		} else {
			cs.Changes = append(cs.Changes, adapter.Change{Action: "update", Key: "command." + name, Old: op.Cur, New: op.Src})
		}
	}
	cmdOpDst := map[string]bool{}
	for _, op := range cmdOps {
		cmdOpDst[op.Dst] = true
	}
	for _, entry := range a.commands {
		dst := filepath.Join(a.commandsDir(entry.Resource.Scope), entry.Name+".md")
		if cmdOpDst[dst] {
			continue
		}
		src := a.commandSource(entry)
		if tgt, err := os.Readlink(dst); err != nil || tgt != src {
			continue
		}
		if e, ok := st.Get("claude", "command."+entry.Name); ok && e.Applied == secret.Hash(dst+" -> "+src) {
			continue
		}
		cs.Changes = append(cs.Changes, adapter.Change{Action: "adopt", Key: "command." + entry.Name, New: dst + " -> " + src})
	}
	// ---- subagent links (parallel to commands) ----
	subOps, err := link.Plan(a.subagentLinks(), a.managedRoots()...)
	if err != nil {
		return adapter.ChangeSet{}, err
	}
	subByName := map[string]config.NamedResource{}
	for _, entry := range a.subagents {
		subByName[entry.Name] = entry
	}
	for _, op := range subOps {
		name := strings.TrimSuffix(filepath.Base(op.Dst), ".md")
		entry := subByName[name]
		inactive := a.inactiveSubagentsDir(entry.Resource.Scope)
		if op.Cur == "" && inactive != "" && link.IsManaged(filepath.Join(inactive, name+".md"), a.managedRoots()...) {
			cs.Changes = append(cs.Changes, adapter.Change{Action: "update", Key: "subagent." + name, Old: filepath.Join(inactive, name+".md"), New: op.Dst + " -> " + op.Src})
		} else if op.Cur == "" {
			cs.Changes = append(cs.Changes, adapter.Change{Action: "create", Key: "subagent." + name, New: op.Dst + " -> " + op.Src})
		} else {
			cs.Changes = append(cs.Changes, adapter.Change{Action: "update", Key: "subagent." + name, Old: op.Cur, New: op.Src})
		}
	}
	subOpDst := map[string]bool{}
	for _, op := range subOps {
		subOpDst[op.Dst] = true
	}
	for _, entry := range a.subagents {
		dst := filepath.Join(a.subagentsDir(entry.Resource.Scope), entry.Name+".md")
		if subOpDst[dst] {
			continue
		}
		src := a.subagentSource(entry)
		if tgt, err := os.Readlink(dst); err != nil || tgt != src {
			continue
		}
		if e, ok := st.Get("claude", "subagent."+entry.Name); ok && e.Applied == secret.Hash(dst+" -> "+src) {
			continue
		}
		cs.Changes = append(cs.Changes, adapter.Change{Action: "adopt", Key: "subagent." + entry.Name, New: dst + " -> " + src})
	}
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
		name := copySubagentName(op.Dst)
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
	for _, key := range st.Keys("claude") {
		if hasPrefix(key, "skill.") {
			// skill.* lives on disk as a symlink, not a JSON value. Its Applied was
			// stored as Hash(dst + " -> " + src); reproduce it by reading the link at
			// the dst state recorded — NOT at the current scope's skillsDir. A pending
			// [skills] scope switch changes skillsDir but leaves the applied link in
			// place; reading the new scope's (empty) location would make an intact old
			// link look "missing" (false drift) instead of a pending relocation Plan
			// already surfaces.
			e, ok := st.Get("claude", key)
			if !ok {
				continue
			}
			dst, ok := recordedDst(e.Desired)
			if !ok {
				continue
			}
			target, err := os.Readlink(dst)
			if err != nil {
				continue // missing or not a symlink → omit (engine infers missing)
			}
			out[key] = secret.Hash(dst + " -> " + target)
			continue
		}
		if hasPrefix(key, "command.") {
			e, ok := st.Get("claude", key)
			if !ok {
				continue
			}
			dst, ok := recordedDst(e.Desired)
			if !ok {
				continue
			}
			target, err := os.Readlink(dst)
			if err != nil {
				continue
			}
			out[key] = secret.Hash(dst + " -> " + target)
			continue
		}
		if hasPrefix(key, "subagent.") {
			e, ok := st.Get("claude", key)
			if !ok {
				continue
			}
			dst, ok := recordedDst(e.Desired)
			if !ok {
				continue
			}
			target, err := os.Readlink(dst)
			if err != nil {
				continue
			}
			out[key] = secret.Hash(dst + " -> " + target)
			continue
		}
		if hasPrefix(key, "subagentcopy.") {
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
			continue
		}
		// Structured-document keys were re-hashed above via structproj.Observe.
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
	// link.Link pass below; noop and subagentcopy.* are handled elsewhere.
	for _, c := range cs.Changes {
		if !(hasPrefix(c.Key, "skill.") || hasPrefix(c.Key, "command.") || hasPrefix(c.Key, "subagent.")) {
			continue
		}
		switch c.Action {
		case "adopt":
			// A correct-but-unrecorded symlink recorded into state without touching
			// disk; its value is "dst -> src", recorded like a freshly linked one.
			st.Set("claude", c.Key, c.New, secret.Hash(c.New))
		case "delete":
			// Only a symlink into our content dir is removed; anything else is a
			// conflict error inside link.Remove. A de-declared resource is no longer
			// in a.skills/commands/subagents, so recover the on-disk location from the
			// dst state recorded for it. Fall back to user scope when state is missing.
			var dst string
			if e, ok := st.Get("claude", c.Key); ok {
				dst, _ = recordedDst(e.Desired)
			}
			if dst == "" {
				switch {
				case hasPrefix(c.Key, "skill."):
					dst = filepath.Join(a.skillsDir("user"), trim(c.Key, "skill."))
				case hasPrefix(c.Key, "command."):
					dst = filepath.Join(a.commandsDir("user"), trim(c.Key, "command.")+".md")
				case hasPrefix(c.Key, "subagent."):
					dst = filepath.Join(a.subagentsDir("user"), trim(c.Key, "subagent.")+".md")
				}
			}
			if err := link.Remove(dst, a.managedRoots()...); err != nil {
				return err
			}
			st.Delete("claude", c.Key)
		}
	}
	// Fail fast on link conflicts before writing any file. Both skill and
	// command conflicts must be detected here, before any JSON write or state
	// mutation below — otherwise a command conflict could let Apply partially
	// write JSON and commit skill-link state before erroring.
	links := a.links()
	if _, err := link.Plan(links, a.managedRoots()...); err != nil {
		return err
	}
	cmdLinks := a.commandLinks()
	if _, err := link.Plan(cmdLinks, a.managedRoots()...); err != nil {
		return err
	}
	subLinks := a.subagentLinks()
	if _, err := link.Plan(subLinks, a.managedRoots()...); err != nil {
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
	// Prune a link left at a skill's inactive scope after a per-resource scope
	// switch, so no orphan remains. Only our own managed symlink is removed
	// (IsManaged guards it); a foreign file or an absent path is left untouched
	// — never an error. Each skill carries its own scope, so the inactive dir is
	// computed per entry.
	for _, entry := range a.skills {
		inactive := a.inactiveSkillsDir(entry.Resource.Scope)
		if inactive == "" {
			continue
		}
		old := filepath.Join(inactive, entry.Name)
		if link.IsManaged(old, a.managedRoots()...) {
			if err := link.Remove(old, a.managedRoots()...); err != nil {
				return err
			}
		}
	}
	for dst, src := range links {
		if _, err := link.Link(src, dst, a.managedRoots()...); err != nil {
			return err
		}
		// Record the link in state so pruning sees de-declared skills later.
		st.Set("claude", "skill."+filepath.Base(dst), dst+" -> "+src, secret.Hash(dst+" -> "+src))
	}
	// Prune a command link left at its inactive scope after a scope switch.
	for _, entry := range a.commands {
		inactive := a.inactiveCommandsDir(entry.Resource.Scope)
		if inactive == "" {
			continue
		}
		old := filepath.Join(inactive, entry.Name+".md")
		if link.IsManaged(old, a.managedRoots()...) {
			if err := link.Remove(old, a.managedRoots()...); err != nil {
				return err
			}
		}
	}
	for dst, src := range cmdLinks {
		if _, err := link.Link(src, dst, a.managedRoots()...); err != nil {
			return err
		}
		st.Set("claude", "command."+strings.TrimSuffix(filepath.Base(dst), ".md"), dst+" -> "+src, secret.Hash(dst+" -> "+src))
	}
	// Prune a subagent link left at its inactive scope after a scope switch.
	for _, entry := range a.subagents {
		inactive := a.inactiveSubagentsDir(entry.Resource.Scope)
		if inactive == "" {
			continue
		}
		old := filepath.Join(inactive, entry.Name+".md")
		if link.IsManaged(old, a.managedRoots()...) {
			if err := link.Remove(old, a.managedRoots()...); err != nil {
				return err
			}
		}
	}
	for dst, src := range subLinks {
		if _, err := link.Link(src, dst, a.managedRoots()...); err != nil {
			return err
		}
		st.Set("claude", "subagent."+strings.TrimSuffix(filepath.Base(dst), ".md"), dst+" -> "+src, secret.Hash(dst+" -> "+src))
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
