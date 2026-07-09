# Explicit Config Resource Model Implementation Plan

Status: Executed on 2026-07-09. The unchecked boxes below are historical task
text from the implementation plan, not the current work queue.

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace Homonto's legacy list-style skill config with an explicit per-resource config model for frameworks, skills, commands, subagents, local provider content, scopes, targets, and model routing.

**Architecture:** Keep the existing MCP/plugins/settings projection working while introducing the new resource model as the single config API. The config package owns parsing and validation; adapters consume normalized resource entries instead of raw TOML fields; engine/scaffold/docs move from `content/` to the approved `homonto/` local provider root.

**Tech Stack:** Go 1.x, `github.com/pelletier/go-toml/v2`, Cobra CLI, existing adapter/engine/state packages, `go test`.

## Global Constraints

- Scope is always required for `frameworks`, `skills`, `commands`, and `subagents`; valid values are exactly `user` and `project`.
- Resource sources are first-release local/bundled only: `builtin:<name>` or `local:<name>`.
- Local provider content lives under `homonto/`, resolved relative to the directory containing `homonto.toml`.
- Generated state/cache remains under `.homonto/` only.
- Valid target tools are exactly `claude` and `opencode`; omitted resource targets mean both tools.
- All three levels are required for each model-enabled target tool: `architectural`, `coding`, and `trivial`. Model-enabled tools come from frameworks, commands, and subagents; skills alone do not require model routing.
- Each model route requires `model` and at least one of `effort` or `variant`; values are validated for presence only.
- Skills do not use model levels; commands and subagents will use levels in later catalog/projection plans.
- The existing MCP, plugin, setting, secret, state, pruning, drift, and surgical merge behavior must keep passing unless explicitly changed by this plan.
- No remote fetching, registry behavior, marketplace behavior, or public third-party package format is introduced in this plan.

---

## File Structure

- Modify `internal/config/config.go`: replace legacy `Skills` with explicit resource maps, add model routing structs, validation helpers, and normalized resource accessors.
- Modify `internal/config/config_test.go`: replace legacy skill tests with explicit resource/model validation tests while preserving MCP/plugin/settings tests.
- Modify `internal/adapter/claude/claude.go`: consume `Config.SkillEntriesForTool("claude")`, compute per-skill scope destinations, and use `homonto/skills` as the local provider root.
- Modify `internal/adapter/opencode/opencode.go`: same as Claude, for OpenCode.
- Modify affected adapter/engine tests under `internal/adapter/{claude,opencode}` and `internal/engine`: construct configs with explicit skill resources instead of `Skills{Scope, Own}`.
- Modify `internal/engine/engine.go`: default content root becomes `homonto`, remove global skill scope wiring.
- Modify `internal/engine/status.go`: `doctor` checks explicit skill resources and per-resource scopes.
- Modify `internal/scaffold/scaffold.go` and `internal/scaffold/scaffold_test.go`: scaffold explicit scopes, model routing examples, and `homonto/skills/.gitkeep`.
- Modify `internal/cli/plan.go` and `internal/cli/apply.go`: call `engine.Build(cfgPath, home, "homonto")`.
- Modify root `homonto.toml`: dogfood explicit local skills and required model tables.
- Move `content/skills/*` to `homonto/skills/*` and remove the old `content/` source tree.
- Modify docs/specs touched by config behavior: `docs/specs/config-model.md`, `docs/specs/tool-adapters.md`, `README.md`, and `docs/NEXT_AGENT.md`.

---

### Task 1: Add Explicit Resource Types And Parser Tests

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`

**Interfaces:**
- Consumes: Existing `config.Load(path string) (*Config, error)`.
- Produces: `config.Resource`, `config.NamedResource`, `config.ModelRoute`, `(*Config).SkillEntriesForTool(tool string) []NamedResource`, `(*Config).EnabledModelTools() []string`.

- [ ] **Step 1: Replace the sample config test with the explicit shape**

Edit `internal/config/config_test.go` so the `sample` constant becomes:

```go
const sample = `
[mcps.codegraph]
command = ["codegraph", "serve", "--mcp"]

[mcps.brave]
command = ["npx", "-y", "server-brave"]
env = { BRAVE_API_KEY = "${pass:ai/brave}" }
targets = ["claude"]

[frameworks.onto]
source = "builtin:onto"
scope = "project"

[skills.graphify]
source = "local:graphify"
scope = "project"

[skills.comet]
source = "builtin:comet"
scope = "user"
targets = ["claude"]

[commands.review]
source = "builtin:review"
scope = "project"
targets = ["opencode"]

[subagents.architect]
source = "builtin:architect"
scope = "project"

[plugins]
claude = ["claude-hud@official"]
opencode = ["@slkiser/opencode-quota"]

[settings.claude]
model = "opus"

[settings.opencode]
model = "anthropic/claude-opus-4-8"

[models.claude.architectural]
model = "opus"
variant = "max"

[models.claude.coding]
model = "sonnet"
effort = "normal"

[models.claude.trivial]
model = "haiku"
effort = "fast"

[models.opencode.architectural]
model = "anthropic/claude-opus-4-8"
effort = "high"

[models.opencode.coding]
model = "anthropic/claude-sonnet-4"
effort = "medium"

[models.opencode.trivial]
model = "openai/gpt-5-mini"
variant = "cheap"
`
```

Update `TestLoad` assertions to check explicit resources:

```go
func TestLoad(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "homonto.toml")
	if err := os.WriteFile(p, []byte(sample), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := Load(p)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got := c.MCPs["codegraph"].Command; len(got) != 3 || got[0] != "codegraph" {
		t.Fatalf("codegraph command = %v", got)
	}
	if got := c.MCPs["brave"].Env["BRAVE_API_KEY"]; got != "${pass:ai/brave}" {
		t.Fatalf("brave env = %q", got)
	}
	if got := c.MCPs["codegraph"].TargetsOrAll(); len(got) != 2 {
		t.Fatalf("default targets = %v", got)
	}
	if got := c.MCPs["brave"].TargetsOrAll(); len(got) != 1 || got[0] != "claude" {
		t.Fatalf("brave targets = %v", got)
	}
	if c.Settings.Claude["model"] != "opus" {
		t.Fatalf("claude model = %v", c.Settings.Claude["model"])
	}
	if got := c.Frameworks["onto"].Scope; got != "project" {
		t.Fatalf("framework onto scope = %q", got)
	}
	if got := c.Skills["graphify"].Source; got != "local:graphify" {
		t.Fatalf("skill graphify source = %q", got)
	}
	claudeSkills := c.SkillEntriesForTool("claude")
	if len(claudeSkills) != 2 || claudeSkills[0].Name != "comet" || claudeSkills[1].Name != "graphify" {
		t.Fatalf("claude skill entries = %#v", claudeSkills)
	}
	opencodeSkills := c.SkillEntriesForTool("opencode")
	if len(opencodeSkills) != 1 || opencodeSkills[0].Name != "graphify" {
		t.Fatalf("opencode skill entries = %#v", opencodeSkills)
	}
	if got := c.Models.Claude["architectural"].Variant; got != "max" {
		t.Fatalf("claude architectural variant = %q", got)
	}
}
```

Delete the legacy tests `TestLoadRejectsBadSkillNames` and `TestLoadSkillScope` in the same edit. They reference the removed `Skills.Scope` / `Skills.Own` fields and would keep the test package from compiling even when running a focused `-run TestLoad`.

- [ ] **Step 2: Run the config test to verify it fails**

Run: `go test ./internal/config -run TestLoad -count=1`

Expected: FAIL with compile errors or field errors showing that `Config.Frameworks`, explicit `Config.Skills` maps, `Config.Commands`, `Config.Subagents`, `Config.Models`, and `SkillEntriesForTool` do not exist yet.

- [ ] **Step 3: Add the explicit resource types**

Edit `internal/config/config.go`. Replace the existing `Skills` struct with these types:

```go
type Resource struct {
	Source  string   `toml:"source"`
	Scope   string   `toml:"scope"`
	Targets []string `toml:"targets"`
}

func (r Resource) TargetsOrAll() []string {
	if len(r.Targets) == 0 {
		return []string{"claude", "opencode"}
	}
	return r.Targets
}

type NamedResource struct {
	Name     string
	Resource Resource
}

type ModelRoute struct {
	Model   string `toml:"model"`
	Effort  string `toml:"effort"`
	Variant string `toml:"variant"`
}

type ModelConfig struct {
	Claude   map[string]ModelRoute `toml:"claude"`
	OpenCode map[string]ModelRoute `toml:"opencode"`
}
```

Update `Config` to this shape while keeping MCP/plugins/settings fields:

```go
type Config struct {
	MCPs       map[string]MCP      `toml:"mcps"`
	Frameworks map[string]Resource `toml:"frameworks"`
	Skills     map[string]Resource `toml:"skills"`
	Commands   map[string]Resource `toml:"commands"`
	Subagents  map[string]Resource `toml:"subagents"`
	Models     ModelConfig         `toml:"models"`
	Plugins    Plugins             `toml:"plugins"`
	Settings   Settings            `toml:"settings"`
}
```

Add these helper methods near `TargetsOrAll`:

```go
func (c *Config) SkillEntriesForTool(tool string) []NamedResource {
	return entriesForTool(c.Skills, tool)
}

func (c *Config) EnabledModelTools() []string {
	seen := map[string]bool{}
	for _, resources := range []map[string]Resource{c.Frameworks, c.Commands, c.Subagents} {
		for _, r := range resources {
			for _, target := range r.TargetsOrAll() {
				seen[target] = true
			}
		}
	}
	out := make([]string, 0, len(seen))
	for tool := range seen {
		out = append(out, tool)
	}
	sort.Strings(out)
	return out
}

func entriesForTool(resources map[string]Resource, tool string) []NamedResource {
	var out []NamedResource
	for name, r := range resources {
		if containsString(r.TargetsOrAll(), tool) {
			out = append(out, NamedResource{Name: name, Resource: r})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func containsString(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}
```

Add `sort` to the import list in `internal/config/config.go`.

- [ ] **Step 4: Remove the legacy skill-scope defaulting block**

Delete the `switch c.Skills.Scope` block and the loop over `c.Skills.Own` from `Load`. Leave MCP/plugins/settings validation unchanged for now.

- [ ] **Step 5: Run the focused config test**

Run: `go test ./internal/config -run TestLoad -count=1`

Expected: PASS for `TestLoad`. Other config tests may still fail until Task 2 updates validation coverage.

- [ ] **Step 6: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go
git commit -m "feat(config): add explicit resource model"
```

---

### Task 2: Enforce Resource Scope, Source, Target, And Model Validation

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`

**Interfaces:**
- Consumes: `Resource`, `ModelRoute`, `Config.EnabledModelTools()` from Task 1.
- Produces: load-time validation for explicit resources and model routing.

- [ ] **Step 1: Replace legacy skill validation tests**

In `internal/config/config_test.go`, delete `TestLoadRejectsBadSkillNames` and `TestLoadSkillScope`. Add these tests:

```go
func TestLoadRejectsBadResourceNames(t *testing.T) {
	for _, tc := range []struct{ kind, table, name string }{
		{"framework", "frameworks", "../evil"},
		{"skill", "skills", ".."},
		{"command", "commands", ""},
		{"subagent", "subagents", "a/b"},
		{"subagent", "subagents", `a\b`},
		{"skill", "skills", "0"},
	} {
		doc := "[" + tc.table + "." + strconv.Quote(tc.name) + "]\nsource=\"local:x\"\nscope=\"project\"\n" + validModelsBothTools()
		err := loadDoc(t, doc)
		if err == nil {
			t.Fatalf("%s name %q accepted; want load error", tc.kind, tc.name)
		}
		if !strings.Contains(err.Error(), strconv.Quote(tc.name)) {
			t.Fatalf("error for %q does not name the entry: %v", tc.name, err)
		}
	}
}

func TestLoadRejectsResourceWithoutExplicitScope(t *testing.T) {
	err := loadDoc(t, "[skills.graphify]\nsource=\"local:graphify\"\n" + validModelsBothTools())
	if err == nil {
		t.Fatal("resource without scope accepted; want load error")
	}
	for _, want := range []string{"skills.graphify", "scope"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %v does not mention %q", err, want)
		}
	}
}

func TestLoadRejectsInvalidResourceScope(t *testing.T) {
	err := loadDoc(t, "[commands.review]\nsource=\"builtin:review\"\nscope=\"global\"\n" + validModelsBothTools())
	if err == nil {
		t.Fatal("scope global accepted; want load error")
	}
	for _, want := range []string{`"global"`, "user", "project"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %v does not mention %q", err, want)
		}
	}
}

func TestLoadRejectsInvalidResourceSource(t *testing.T) {
	for _, source := range []string{"", "https://example.com/x", "github:owner/repo", "builtin:", "local:"} {
		doc := "[skills.graphify]\nsource=" + strconv.Quote(source) + "\nscope=\"project\"\n" + validModelsBothTools()
		err := loadDoc(t, doc)
		if err == nil {
			t.Fatalf("source %q accepted; want load error", source)
		}
		if !strings.Contains(err.Error(), strconv.Quote(source)) {
			t.Fatalf("error %v does not name source %q", err, source)
		}
	}
}

func TestLoadRejectsUnknownResourceTargets(t *testing.T) {
	err := loadDoc(t, "[subagents.architect]\nsource=\"builtin:architect\"\nscope=\"project\"\ntargets=[\"claud\"]\n" + validModelsBothTools())
	if err == nil {
		t.Fatal("unknown target accepted; want load error")
	}
	if !strings.Contains(err.Error(), strconv.Quote("claud")) {
		t.Fatalf("error does not name unknown target: %v", err)
	}
}

func TestLoadRequiresAllModelLevelsForEnabledTools(t *testing.T) {
	doc := `
[commands.review]
source = "builtin:review"
scope = "project"
targets = ["opencode"]

[models.opencode.architectural]
model = "anthropic/claude-opus-4-8"
effort = "high"

[models.opencode.coding]
model = "anthropic/claude-sonnet-4"
effort = "medium"
`
	err := loadDoc(t, doc)
	if err == nil {
		t.Fatal("missing opencode trivial model accepted; want load error")
	}
	for _, want := range []string{"models.opencode.trivial", "model"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %v does not mention %q", err, want)
		}
	}
}

func TestLoadRequiresModelAndEffortOrVariant(t *testing.T) {
	doc := `
[commands.review]
source = "builtin:review"
scope = "project"
targets = ["claude"]

[models.claude.architectural]
model = "opus"
variant = "max"

[models.claude.coding]
model = "sonnet"

[models.claude.trivial]
model = "haiku"
effort = "fast"
`
	err := loadDoc(t, doc)
	if err == nil {
		t.Fatal("model without effort or variant accepted; want load error")
	}
	if !strings.Contains(err.Error(), "models.claude.coding") {
		t.Fatalf("error does not name route: %v", err)
	}
}

func TestLoadDoesNotRequireModelsForSkillsOnly(t *testing.T) {
	err := loadDoc(t, "[skills.graphify]\nsource=\"local:graphify\"\nscope=\"project\"\n")
	if err != nil {
		t.Fatalf("skills-only config required model routing: %v", err)
	}
}

func validModelsBothTools() string {
	return `
[models.claude.architectural]
model = "opus"
variant = "max"
[models.claude.coding]
model = "sonnet"
effort = "normal"
[models.claude.trivial]
model = "haiku"
effort = "fast"
[models.opencode.architectural]
model = "anthropic/claude-opus-4-8"
effort = "high"
[models.opencode.coding]
model = "anthropic/claude-sonnet-4"
effort = "medium"
[models.opencode.trivial]
model = "openai/gpt-5-mini"
variant = "cheap"
`
}
```

- [ ] **Step 2: Run validation tests to verify failure**

Run: `go test ./internal/config -count=1`

Expected: FAIL because resource source/scope/target/model validation is not implemented, and legacy tests still need removal if they were not removed in Step 1.

- [ ] **Step 3: Add validation helpers**

In `internal/config/config.go`, add these helpers below `validateKey`:

```go
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
		if !validSource(r.Source) {
			return fmt.Errorf("parse config: %s source %q is invalid; use builtin:<name> or local:<name>", label, r.Source)
		}
		for _, target := range r.Targets {
			if target != "claude" && target != "opencode" {
				return fmt.Errorf("parse config: %s targets unknown tool %q; valid targets are \"claude\" and \"opencode\"", label, target)
			}
		}
	}
	return nil
}

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

func validateModels(c *Config) error {
	for _, tool := range c.EnabledModelTools() {
		for _, level := range []string{"architectural", "coding", "trivial"} {
			route, ok := modelRouteFor(c.Models, tool, level)
			label := "models." + tool + "." + level
			if !ok {
				return fmt.Errorf("parse config: %s is required for enabled target tool %q", label, tool)
			}
			if strings.TrimSpace(route.Model) == "" {
				return fmt.Errorf("parse config: %s model is required", label)
			}
			if strings.TrimSpace(route.Effort) == "" && strings.TrimSpace(route.Variant) == "" {
				return fmt.Errorf("parse config: %s requires effort or variant", label)
			}
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
```

In `Load`, after TOML unmarshal and before MCP validation, call:

```go
	for kind, resources := range map[string]map[string]Resource{
		"frameworks": c.Frameworks,
		"skills":     c.Skills,
		"commands":   c.Commands,
		"subagents":  c.Subagents,
	} {
		if err := validateResources(kind, resources); err != nil {
			return nil, err
		}
	}
	if err := validateModels(&c); err != nil {
		return nil, err
	}
```

- [ ] **Step 4: Run config package tests**

Run: `go test ./internal/config -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go
git commit -m "fix(config): validate explicit resources and model routes"
```

---

### Task 3: Update Skill Projection To Use Explicit Per-Resource Scopes

**Files:**
- Modify: `internal/adapter/claude/claude.go`
- Modify: `internal/adapter/opencode/opencode.go`
- Modify: relevant tests under `internal/adapter/claude/` and `internal/adapter/opencode/`

**Interfaces:**
- Consumes: `Config.SkillEntriesForTool(tool string) []config.NamedResource`.
- Produces: Skill link planning based on each skill resource's `scope` and `source`.

- [ ] **Step 1: Add test helper constructors in adapter tests**

In both `internal/adapter/claude/claude_test.go` and `internal/adapter/opencode/opencode_test.go`, add this helper near the top of each file:

```go
func cfgWithSkills(scope string, names ...string) *config.Config {
	c := &config.Config{Skills: map[string]config.Resource{}}
	for _, name := range names {
		c.Skills[name] = config.Resource{Source: "local:" + name, Scope: scope}
	}
	return c
}
```

For tests in other files in the same package, reuse this helper because Go test files share package scope.

- [ ] **Step 2: Update adapter tests to use explicit resources**

Replace struct literals like:

```go
&config.Config{Skills: config.Skills{Scope: "project", Own: []string{"onto"}}}
```

with:

```go
cfgWithSkills("project", "onto")
```

Replace literals like:

```go
&config.Config{Skills: config.Skills{Own: []string{"foo"}}}
```

with:

```go
cfgWithSkills("user", "foo")
```

Where a test also sets MCPs/settings/plugins, build the config explicitly:

```go
c := cfgWithSkills("user", "onto")
c.Settings = config.Settings{Claude: map[string]any{"model": "opus"}}
c.Plugins = config.Plugins{Claude: []string{"repo-plugin"}}
```

- [ ] **Step 3: Run adapter tests to verify compile failure**

Run: `go test ./internal/adapter/claude ./internal/adapter/opencode -count=1`

Expected: FAIL because production adapters still reference `c.Skills.Own` and `c.Skills.Scope`.

- [ ] **Step 4: Update Claude adapter fields and link helpers**

In `internal/adapter/claude/claude.go`, replace the adapter struct fields:

```go
	scope       string // "" or "user" → home layout; "project" → projectRoot layout
	projectRoot string // directory of homonto.toml; used only for project scope
	skills      []string
```

with:

```go
	projectRoot string // directory of homonto.toml; used for project-scope resources
	skills      []config.NamedResource
```

Replace `WithScope` with:

```go
// WithProjectRoot sets the project root (the homonto.toml directory). It is
// used for project-scope resource placement. MCP servers and settings always
// project under home.
func (a *Adapter) WithProjectRoot(projectRoot string) *Adapter {
	a.projectRoot = projectRoot
	return a
}
```

Replace `skillsDir` with:

```go
func (a *Adapter) skillsDir(scope string) string {
	return skillpath.Dir("claude", scope, a.home, a.projectRoot)
}
```

Replace `inactiveSkillsDir` with:

```go
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
```

Replace `links()` with:

```go
func (a *Adapter) links() map[string]string {
	out := map[string]string{}
	for _, entry := range a.skills {
		name := entry.Name
		out[filepath.Join(a.skillsDir(entry.Resource.Scope), name)] = filepath.Join(a.content, "skills", localSourceName(entry.Resource.Source, name))
	}
	return out
}

func localSourceName(source, fallback string) string {
	if strings.HasPrefix(source, "local:") {
		return strings.TrimPrefix(source, "local:")
	}
	return fallback
}
```

Add `strings` to the Claude adapter imports.

- [ ] **Step 5: Update Claude Plan skill loops**

In `Plan`, replace:

```go
	a.skills = c.Skills.Own
```

with:

```go
	a.skills = c.SkillEntriesForTool("claude")
```

In the link operation loop, replace:

```go
	inactive := a.inactiveSkillsDir()
	for _, op := range ops {
		name := filepath.Base(op.Dst)
		if op.Cur == "" && inactive != "" && link.IsManaged(filepath.Join(inactive, name), a.content) {
```

with:

```go
	entryByName := map[string]config.NamedResource{}
	for _, entry := range a.skills {
		entryByName[entry.Name] = entry
	}
	for _, op := range ops {
		name := filepath.Base(op.Dst)
		entry := entryByName[name]
		inactive := a.inactiveSkillsDir(entry.Resource.Scope)
		if op.Cur == "" && inactive != "" && link.IsManaged(filepath.Join(inactive, name), a.content) {
```

Replace the adoption loop over `c.Skills.Own` with:

```go
	for _, entry := range a.skills {
		name := entry.Name
		dst := filepath.Join(a.skillsDir(entry.Resource.Scope), name)
		if opDst[dst] {
			continue
		}
		src := filepath.Join(a.content, "skills", localSourceName(entry.Resource.Source, name))
		if tgt, err := os.Readlink(dst); err != nil || tgt != src {
			continue
		}
		if e, ok := st.Get("claude", "skill."+name); ok && e.Applied == secret.Hash(dst+" -> "+src) {
			continue
		}
		cs.Changes = append(cs.Changes, adapter.Change{Action: "adopt", Key: "skill." + name, New: dst + " -> " + src})
	}
```

Replace the declared skill loop over `c.Skills.Own` with:

```go
	for _, entry := range a.skills {
		declared["skill."+entry.Name] = true
	}
```

- [ ] **Step 6: Apply equivalent changes to OpenCode adapter**

Make the same field, `WithProjectRoot`, `skillsDir(scope)`, `inactiveSkillsDir(scope)`, `links`, `localSourceName`, `a.skills = c.SkillEntriesForTool("opencode")`, link-loop, adoption-loop, and declared-loop changes in `internal/adapter/opencode/opencode.go`.

Use `skillpath.Dir("opencode", scope, a.home, a.projectRoot)` in OpenCode.

- [ ] **Step 7: Run adapter tests**

Run: `go test ./internal/adapter/claude ./internal/adapter/opencode -count=1`

Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add internal/adapter/claude internal/adapter/opencode
git commit -m "refactor(adapters): project explicit skill resources"
```

---

### Task 4: Update Engine, Doctor, CLI Defaults, And Scaffold

**Files:**
- Modify: `internal/engine/engine.go`
- Modify: `internal/engine/status.go`
- Modify: `internal/engine/*_test.go`
- Modify: `internal/cli/plan.go`
- Modify: `internal/cli/apply.go`
- Modify: `internal/scaffold/scaffold.go`
- Modify: `internal/scaffold/scaffold_test.go`
- Modify: `homonto.toml`

**Interfaces:**
- Consumes: adapter `WithProjectRoot(projectRoot)` methods and explicit config model.
- Produces: default local provider root `homonto/`, explicit-scope scaffold, and doctor checks based on explicit skill resources.

- [ ] **Step 1: Update engine wiring test expectations**

In engine tests, replace any old config snippets like:

```toml
[skills]
scope = "project"
own = ["onto"]
```

with:

```toml
[skills.onto]
source = "local:onto"
scope = "project"

[models.claude.architectural]
model = "opus"
variant = "max"
[models.claude.coding]
model = "sonnet"
effort = "normal"
[models.claude.trivial]
model = "haiku"
effort = "fast"
[models.opencode.architectural]
model = "anthropic/claude-opus-4-8"
effort = "high"
[models.opencode.coding]
model = "anthropic/claude-sonnet-4"
effort = "medium"
[models.opencode.trivial]
model = "openai/gpt-5-mini"
variant = "cheap"
```

Skills-only tests do not need model tables because levels apply to commands and subagents, not skills. Framework tests should include model tables because frameworks are model-enabled bundles.

- [ ] **Step 2: Run engine and CLI/scaffold tests to verify failure**

Run: `go test ./internal/engine ./internal/cli ./internal/scaffold -count=1`

Expected: FAIL because engine still calls `cfg.Skills.Scope`, adapters still expect `WithScope`, and scaffold still writes `content/`.

- [ ] **Step 3: Update engine Build**

In `internal/engine/engine.go`, replace the comment on `Build` with:

```go
// Build loads config and wires both adapters. home is $HOME; contentDir is the
// local provider root; state lives in <repo>/.homonto next to the config.
```

Replace:

```go
	scope := cfg.Skills.Scope
	return &Engine{
		Cfg: cfg,
		Adapters: []adapter.Adapter{
			claude.New(home, contentDir).WithScope(scope, projectRoot),
			opencode.New(home, contentDir).WithScope(scope, projectRoot),
		},
```

with:

```go
	return &Engine{
		Cfg: cfg,
		Adapters: []adapter.Adapter{
			claude.New(home, contentDir).WithProjectRoot(projectRoot),
			opencode.New(home, contentDir).WithProjectRoot(projectRoot),
		},
```

- [ ] **Step 4: Update CLI default content root**

In `internal/cli/plan.go`, replace:

```go
	e, err := engine.Build(cfgPath, home, "content")
```

with:

```go
	e, err := engine.Build(cfgPath, home, "homonto")
```

Make the same replacement in `internal/cli/apply.go`.

- [ ] **Step 5: Update doctor skill checks**

In `internal/engine/status.go`, replace the loop over `e.Cfg.Skills.Own` with:

```go
	for _, entry := range e.Cfg.SkillEntriesForTool("claude") {
		name := entry.Name
		sourceName := name
		if strings.HasPrefix(entry.Resource.Source, "local:") {
			sourceName = strings.TrimPrefix(entry.Resource.Source, "local:")
		}
		p := filepath.Join(e.ContentDir, "skills", sourceName)
		if _, err := os.Stat(p); err != nil {
			out = append(out, fmt.Sprintf("warn: skill %q missing from %s", name, p))
			continue
		}
		dst := filepath.Join(skillpath.Dir("claude", entry.Resource.Scope, e.Home, e.ProjectRoot), name)
		if target, err := os.Readlink(dst); err == nil && target == p {
			out = append(out, fmt.Sprintf("ok: skill %q linked (claude)", name))
		} else {
			out = append(out, fmt.Sprintf("warn: skill %q content present, not linked for claude (run apply)", name))
		}
	}
	for _, entry := range e.Cfg.SkillEntriesForTool("opencode") {
		name := entry.Name
		sourceName := name
		if strings.HasPrefix(entry.Resource.Source, "local:") {
			sourceName = strings.TrimPrefix(entry.Resource.Source, "local:")
		}
		p := filepath.Join(e.ContentDir, "skills", sourceName)
		if _, err := os.Stat(p); err != nil {
			out = append(out, fmt.Sprintf("warn: skill %q missing from %s", name, p))
			continue
		}
		dst := filepath.Join(skillpath.Dir("opencode", entry.Resource.Scope, e.Home, e.ProjectRoot), name)
		if target, err := os.Readlink(dst); err == nil && target == p {
			out = append(out, fmt.Sprintf("ok: skill %q linked (opencode)", name))
		} else {
			out = append(out, fmt.Sprintf("warn: skill %q content present, not linked for opencode (run apply)", name))
		}
	}
```

Add `strings` to `internal/engine/status.go` imports.

- [ ] **Step 6: Update scaffold output**

In `internal/scaffold/scaffold.go`, replace the `homonto.toml` scaffold body with:

```go
"homonto.toml": `# homonto — declarative config for AI coding tools.
# Secrets are referenced, never stored: use ${pass:path} or ${ENV_VAR}.

# [mcps.codegraph]
# command = ["codegraph", "serve", "--mcp"]
# targets = ["claude", "opencode"]   # default: all

# [frameworks.onto]
# source = "builtin:onto"
# scope = "project"

# [skills.graphify]
# source = "local:graphify"
# scope = "project"

# [commands.review]
# source = "builtin:review"
# scope = "user"
# targets = ["opencode"]

# [subagents.architect]
# source = "builtin:architect"
# scope = "project"

# [plugins]
# claude = ["claude-hud@official"]
# opencode = ["@slkiser/opencode-quota"]

# [settings.claude]
# model = "opus"

# [models.claude.architectural]
# model = "opus"
# variant = "max"
# [models.claude.coding]
# model = "sonnet"
# effort = "normal"
# [models.claude.trivial]
# model = "haiku"
# effort = "fast"
`,
```

Replace `content/skills/.gitkeep` with `homonto/skills/.gitkeep` in the `keep` path.

- [ ] **Step 7: Update root dogfood config**

Replace root `homonto.toml` with explicit local skills and model tables:

```toml
[skills.onto]
source = "local:onto"
scope = "project"

[skills.onto-open]
source = "local:onto-open"
scope = "project"

[skills.onto-design]
source = "local:onto-design"
scope = "project"

[skills.onto-build]
source = "local:onto-build"
scope = "project"

[skills.onto-verify]
source = "local:onto-verify"
scope = "project"

[skills.onto-close]
source = "local:onto-close"
scope = "project"

[skills.onto-fix]
source = "local:onto-fix"
scope = "project"

[skills.onto-tweak]
source = "local:onto-tweak"
scope = "project"

[models.claude.architectural]
model = "opus"
variant = "max"

[models.claude.coding]
model = "sonnet"
effort = "normal"

[models.claude.trivial]
model = "haiku"
effort = "fast"

[models.opencode.architectural]
model = "anthropic/claude-opus-4-8"
effort = "high"

[models.opencode.coding]
model = "anthropic/claude-sonnet-4"
effort = "medium"

[models.opencode.trivial]
model = "openai/gpt-5-mini"
variant = "cheap"
```

- [ ] **Step 8: Run engine, CLI, and scaffold tests**

Run: `go test ./internal/engine ./internal/cli ./internal/scaffold -count=1`

Expected: PASS.

- [ ] **Step 9: Commit**

```bash
git add internal/engine internal/cli internal/scaffold homonto.toml
git commit -m "refactor(engine): use explicit resource config"
```

---

### Task 5: Move Local Provider Content From content/ To homonto/

**Files:**
- Move: `content/skills/*` to `homonto/skills/*`
- Delete: old empty `content/` directories if left empty
- Modify: tests or docs that still mention `content/skills`

**Interfaces:**
- Consumes: CLI/engine default local provider root `homonto` from Task 4.
- Produces: repo dogfood content under the approved local provider root.

- [ ] **Step 1: Move skill content with git**

Run:

```bash
mkdir -p homonto/skills
rmdir content/skills content 2>/dev/null || true
```

Expected: all `onto-*` skill directories now live under `homonto/skills/`.

- [ ] **Step 2: Search for old content path references**

Run: `rg "content/skills|content/" .`

Expected: matches remain only in historical archived docs, if any. Current code, tests, README, living specs, and current handoff docs must not instruct users to use `content/` as the local provider root.

- [ ] **Step 3: Update current references**

For every current code/test/doc match outside `docs/changes/archive/`, replace `content/skills` with `homonto/skills` and replace `content/` with `homonto/` when referring to local provider source content.

Do not edit archived change records only to rewrite history.

- [ ] **Step 4: Run dogfood plan smoke**

Run: `go run . plan`

Expected: either a plan showing skill symlink target updates from old `content/skills` paths to new `homonto/skills` paths, or `No changes. Everything up to date.` if local links were already updated.

- [ ] **Step 5: Run package tests affected by paths**

Run: `go test ./internal/... -count=1`

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add homonto content internal docs README.md homonto.toml
git commit -m "chore: move local provider content to homonto"
```

---

### Task 6: Update Living Specs And User-Facing Docs

**Files:**
- Modify: `docs/specs/config-model.md`
- Modify: `docs/specs/tool-adapters.md`
- Modify: `README.md`
- Modify: `docs/NEXT_AGENT.md`
- Modify: `docs/road-to-release.md`
- Modify: `docs/roadmap.md`

**Interfaces:**
- Consumes: implemented explicit resource config and `homonto/` local provider root.
- Produces: living docs that no longer describe the old `[skills] own` and default scope model as current behavior.

- [ ] **Step 1: Update config spec requirements**

In `docs/specs/config-model.md`, replace the old skill install scope requirement with text matching these requirements:

```markdown
### Requirement: Explicit resource declarations

`homonto` SHALL parse frameworks, skills, commands, and subagents as explicit
per-resource tables. Every resource SHALL declare `source` and `scope`. Scope
SHALL be either `user` or `project`; there is no default. Source SHALL be either
`builtin:<name>` or `local:<name>` in the first release.

#### Scenario: Parse explicit resources
- **WHEN** `homonto.toml` declares `[skills.graphify]` with `source = "local:graphify"` and `scope = "project"`
- **THEN** the loader returns a skill resource named `graphify` with local source `graphify` and project scope

#### Scenario: Missing scope is rejected
- **WHEN** a resource omits `scope`
- **THEN** `Load` returns an error naming that resource and the missing scope
```

Add a model routing requirement:

```markdown
### Requirement: Tool-specific model routing

For every enabled target tool, `homonto.toml` SHALL define all three model
levels: `architectural`, `coding`, and `trivial`. Each route SHALL include a
non-empty `model` and at least one of `effort` or `variant`. Homonto SHALL not
validate provider-specific model names or effort values beyond presence.
```

- [ ] **Step 2: Update tool adapter spec**

In `docs/specs/tool-adapters.md`, replace references to `[skills] own` and a single global skill scope with explicit skill resources and per-resource scope. The current behavior must say skills are sourced from `homonto/skills/<local-name>` for `local:<name>` resources.

- [ ] **Step 3: Update README config example**

In `README.md`, replace the `[skills] scope` and `own = [...]` example with explicit tables:

```toml
[skills.graphify]
source = "local:graphify"
scope = "project"

[models.claude.architectural]
model = "opus"
variant = "max"
```

Also replace claims that owned content lives in `content/` with `homonto/`.

- [ ] **Step 4: Update release direction docs**

In `docs/NEXT_AGENT.md`, `docs/road-to-release.md`, and `docs/roadmap.md`, update the current state so they no longer say the release is ready pending only a tag. They must say the release gate has been reopened for the dual-binary Homonto/Onto release and point to `docs/superpowers/specs/2026-07-09-dual-binary-release-design.md`.

- [ ] **Step 5: Run doc drift search**

Run: `rg "\[skills\]|own =|content/skills|release-ready pending the maintainer's tag|state.yaml" README.md docs homonto.toml internal`

Expected: no current docs/code describe `[skills] own` or `content/skills` as the active model. Historical mentions under `docs/changes/archive/` may remain.

- [ ] **Step 6: Run full verification**

Run:

```bash
gofmt -w internal/config internal/adapter internal/engine internal/cli internal/scaffold
go test ./...
go vet ./...
go run . plan
```

Expected:

- `go test ./...` passes.
- `go vet ./...` passes.
- `go run . plan` succeeds and does not fail parsing the root `homonto.toml`.

- [ ] **Step 7: Commit**

```bash
git add README.md docs internal homonto.toml homonto content
git commit -m "docs: document explicit resource config model"
```

---

## Self-Review Notes

- Spec coverage: This plan covers explicit resource tables, required scopes, local provider root `homonto/`, per-tool model routing validation, first-release source restrictions, and moving current skill projection to explicit resources. Catalog expansion, commands/subagents projection, `onto` binary, and Docker release gate are intentionally left to later subsystem plans.
- Placeholder scan: No task contains unresolved placeholder wording or undefined task names. Every code-facing task includes exact tests, code snippets, commands, and expected outcomes.
- Type consistency: The plan consistently uses `config.Resource`, `config.NamedResource`, `config.ModelRoute`, `config.ModelConfig`, `Config.SkillEntriesForTool`, and `Config.EnabledModelTools`.
