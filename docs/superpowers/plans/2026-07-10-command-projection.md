---
change: command-projection
design-doc: docs/superpowers/specs/2026-07-10-command-projection-design.md
base-ref: 70dd84da50e6f04aacc37197c78a14004ca28a4f
---

# Command Projection Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Project single-file catalog commands (builtin + local) into Claude Code and OpenCode command directories, as a thin parallel of the already-shipped skills catalog.

**Architecture:** Reuse the skills machinery end-to-end for single-file commands: the embedded `catalog/commands/<name>.md` area, a `Framework.Commands` table, a factored `expandResources` DFS, single-file `MaterializeCommands`, a `commandpath` package, `config.ExpandedCommandEntriesForTool`, version-gated engine materialization, and `command.<name>` link ops in both adapters. `internal/link` (variadic multi-root) and `internal/state` (`CatalogVersion`) are reused unchanged.

**Tech Stack:** Go 1.x, `github.com/pelletier/go-toml/v2`, `testing/fstest`, standard library `io/fs`/`os`/`path/filepath`.

## Global Constraints

- Module path: `github.com/noviopenworks/homonto`. All internal imports use this prefix.
- The `internal/catalog` package MUST NOT import `internal/config` (config-agnostic, one-way dependency).
- OpenCode command directory is **singular** `command/` (unlike its plural `skills/`). Claude uses `commands/`.
- Commands are **single Markdown files** (`<name>.md`), never directories. Skills remain directories.
- Command materialization is gated on the **same** `state.CatalogVersion` as skills; the version is recorded only after **both** skills and commands materialize successfully.
- `link.managed()` treats an empty-string root as a prefix match for every absolute path. An empty catalog root must NEVER reach any `link.*` call. Every `link.Plan`/`Link`/`Remove`/`IsManaged` call passes `a.managedRoots()...`; `managedRoots()` guards out empty roots.
- Collision namespaces are separate: a skill and a command MAY share a name (`skill.<n>` and `command.<n>` are distinct state keys in distinct tool directories). Command-vs-command name collisions are config errors.
- The placeholder command name for this change is **`example-command`** (used verbatim everywhere below).
- Verification commands: `go test ./... -count=1`, `go vet ./...`, `go build ./...`.
- `.homonto/`, `/.claude/`, `/.opencode/` are gitignored (dogfood artifacts) — never commit them.

---

### Task 1: Catalog commands content + embed

Ships the one placeholder command file and extends the embed directive so `all:commands` is compiled into the binary. The embed directive fails to compile if `catalog/commands/` does not exist, so the file must be created in the same commit.

**Files:**
- Create: `catalog/commands/example-command.md`
- Modify: `catalog/embed.go:8`
- Test: `internal/catalog/embed_test.go`

**Interfaces:**
- Consumes: `embedded "github.com/noviopenworks/homonto/catalog"` exposing `embedded.FS embed.FS`.
- Produces: an embedded file readable at FS path `commands/example-command.md`.

- [x] **Step 1: Write the failing test**

Create `internal/catalog/embed_test.go`:

```go
package catalog

import (
	"io/fs"
	"testing"

	embedded "github.com/noviopenworks/homonto/catalog"
)

func TestEmbedIncludesPlaceholderCommand(t *testing.T) {
	if _, err := fs.Stat(embedded.FS, "commands/example-command.md"); err != nil {
		t.Fatalf("commands/example-command.md not embedded: %v", err)
	}
}
```

- [x] **Step 2: Run test to verify it fails**

Run: `go test ./internal/catalog/ -run TestEmbedIncludesPlaceholderCommand -count=1`
Expected: FAIL (file not embedded, or build error `pattern all:commands: no matching files found` once the directive is added before the file — either failure is acceptable at this step).

- [x] **Step 3: Create the placeholder command file**

Create `catalog/commands/example-command.md`:

```markdown
---
name: example-command
description: Placeholder command shipped to exercise command projection end-to-end; real command content lands in a later change.
---

# Example Command

This is a placeholder command bundled in the homonto catalog. It exists so the
command-projection machinery (materialize, link, doctor) can be dogfooded before
real command content is authored. Replace or remove it in a later change.
```

- [x] **Step 4: Extend the embed directive**

In `catalog/embed.go`, change line 8 from:

```go
//go:embed all:frameworks all:skills version.txt
```

to:

```go
//go:embed all:frameworks all:skills all:commands version.txt
```

- [x] **Step 5: Run test to verify it passes**

Run: `go test ./internal/catalog/ -run TestEmbedIncludesPlaceholderCommand -count=1`
Expected: PASS

- [x] **Step 6: Commit**

```bash
git add catalog/commands/example-command.md catalog/embed.go internal/catalog/embed_test.go
git commit -m "feat(catalog): embed placeholder command and commands area"
```

---

### Task 2: Framework `[commands]` parse, index, and lookup

Adds a `Commands` table to `Framework`/`frameworkTOML`, validates each command path exists in the embedded FS (mirroring skills), builds a global command index, and adds `CommandPath`.

**Files:**
- Modify: `internal/catalog/catalog.go`
- Test: `internal/catalog/catalog_test.go`

**Interfaces:**
- Consumes: `Load(fsys fs.FS)`, existing `Framework`/`Catalog`/`frameworkTOML` structs.
- Produces:
  - `Framework.Commands map[string]string` (command name → `commands/<n>.md`).
  - `Catalog.commands map[string]string` (global index).
  - `func (c *Catalog) CommandPath(name string) (string, bool)`.

- [x] **Step 1: Write the failing test**

Add to `internal/catalog/catalog_test.go`. First extend `fixtureFS()` so the superpowers framework declares a command and its file exists:

```go
// In fixtureFS(), change the superpowers framework.toml block to include a
// [commands] table, and add the command file entry:
//
//   "frameworks/superpowers/framework.toml": {Data: []byte(`name = "superpowers"
// version = "0.1.0"
// description = "sp"
// [skills]
// brainstorming = "skills/brainstorming"
// [commands]
// demo-cmd = "commands/demo-cmd.md"
// `)},
//   "commands/demo-cmd.md": {Data: []byte("d")},

func TestLoadIndexesFrameworkCommands(t *testing.T) {
	c, err := Load(fixtureFS())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	sp, ok := c.Framework("superpowers")
	if !ok {
		t.Fatal("superpowers not indexed")
	}
	if sp.Commands["demo-cmd"] != "commands/demo-cmd.md" {
		t.Fatalf("superpowers commands = %v", sp.Commands)
	}
	if p, ok := c.CommandPath("demo-cmd"); !ok || p != "commands/demo-cmd.md" {
		t.Fatalf("demo-cmd path = %q ok=%v", p, ok)
	}
}

func TestLoadRejectsMissingCommandPath(t *testing.T) {
	m := fixtureFS()
	delete(m, "commands/demo-cmd.md")
	_, err := Load(m)
	if err == nil || !strings.Contains(err.Error(), "commands/demo-cmd.md") {
		t.Fatalf("expected missing-command-path error, got %v", err)
	}
}
```

- [x] **Step 2: Run test to verify it fails**

Run: `go test ./internal/catalog/ -run 'TestLoadIndexesFrameworkCommands|TestLoadRejectsMissingCommandPath' -count=1`
Expected: FAIL (`Commands`/`CommandPath` undefined).

- [x] **Step 3: Implement the parse, index, and lookup**

In `internal/catalog/catalog.go`:

Add the `Commands` field to `Framework`:

```go
type Framework struct {
	Name         string
	Version      string
	Description  string
	Dependencies []string          // framework names
	Skills       map[string]string // skill name -> catalog-relative path ("skills/<n>")
	Commands     map[string]string // command name -> catalog-relative path ("commands/<n>.md")
}
```

Add the `commands` index to `Catalog`:

```go
type Catalog struct {
	fsys       fs.FS
	frameworks map[string]Framework
	skills     map[string]string // skill name -> catalog-relative path (global index)
	commands   map[string]string // command name -> catalog-relative path (global index)
	version    string
}
```

Add the TOML field to `frameworkTOML`:

```go
type frameworkTOML struct {
	Name         string `toml:"name"`
	Version      string `toml:"version"`
	Description  string `toml:"description"`
	Dependencies struct {
		Frameworks []string `toml:"frameworks"`
	} `toml:"dependencies"`
	Skills   map[string]string `toml:"skills"`
	Commands map[string]string `toml:"commands"`
}
```

In `Load`, initialize the index in the `&Catalog{...}` literal:

```go
	c := &Catalog{
		fsys:       fsys,
		frameworks: map[string]Framework{},
		skills:     map[string]string{},
		commands:   map[string]string{},
	}
```

After the existing `for skill, sp := range ft.Skills { ... }` loop (and before the `c.frameworks[dir] = Framework{...}` assignment), add the parallel command loop:

```go
		for command, cp := range ft.Commands {
			if _, err := fs.Stat(fsys, cp); err != nil {
				return nil, fmt.Errorf("catalog: framework %q command %q path %q missing from catalog", dir, command, cp)
			}
			if prev, ok := c.commands[command]; ok && prev != cp {
				return nil, fmt.Errorf("catalog: command %q mapped to both %q and %q", command, prev, cp)
			}
			c.commands[command] = cp
		}
```

Add `Commands: ft.Commands` to the framework assignment:

```go
		c.frameworks[dir] = Framework{
			Name:         ft.Name,
			Version:      ft.Version,
			Description:  ft.Description,
			Dependencies: ft.Dependencies.Frameworks,
			Skills:       ft.Skills,
			Commands:     ft.Commands,
		}
```

Add the lookup at the end of the file (next to `SkillPath`):

```go
// CommandPath returns a command's catalog-relative path ("commands/<n>.md") and
// whether it is known.
func (c *Catalog) CommandPath(name string) (string, bool) {
	p, ok := c.commands[name]
	return p, ok
}
```

- [x] **Step 4: Run test to verify it passes**

Run: `go test ./internal/catalog/ -run 'TestLoadIndexesFrameworkCommands|TestLoadRejectsMissingCommandPath|TestLoad' -count=1`
Expected: PASS (existing `TestLoad*` still pass with the extended fixture).

- [x] **Step 5: Commit**

```bash
git add internal/catalog/catalog.go internal/catalog/catalog_test.go
git commit -m "feat(catalog): parse and index framework [commands] table"
```

---

### Task 3: Factor `expandResources` and add `ExpandCommands`

Refactors the existing three-color cycle-detecting DFS in `Expand` into a private `expandResources` helper parameterized by a resource selector, then adds `ExpandCommands` alongside the delegating `Expand`.

**Files:**
- Modify: `internal/catalog/expand.go`
- Test: `internal/catalog/expand_test.go`

**Interfaces:**
- Consumes: `Catalog.frameworks`, `Framework.Skills`, `Framework.Commands`.
- Produces:
  - `type Expanded struct{ Name, Framework string }`
  - `func (c *Catalog) expandResources(frameworkNames []string, sel func(Framework) map[string]string) ([]Expanded, error)`
  - `type ExpandedCommand struct{ Name, Framework string }`
  - `func (c *Catalog) ExpandCommands(frameworkNames []string) ([]ExpandedCommand, error)`
  - `Expand` unchanged in signature: `func (c *Catalog) Expand(frameworkNames []string) ([]ExpandedSkill, error)`.

- [x] **Step 1: Write the failing test**

Extend `internal/catalog/expand_test.go`. First extend `graphFS` so each framework can also declare commands; add a `commands map[string][]string` parameter:

```go
// Replace graphFS's signature and body so it also writes a [commands] table:
//
// func graphFS(deps map[string][]string, skills, commands map[string][]string) fstest.MapFS {
//   ... after writing [skills], append:
//   if cs := commands[fw]; len(cs) > 0 {
//       b.WriteString("[commands]\n")
//       for _, cmd := range cs {
//           b.WriteString(cmd + " = \"commands/" + cmd + ".md\"\n")
//           m["commands/"+cmd+".md"] = &fstest.MapFile{Data: []byte("x")}
//       }
//   }
// }
//
// Update the two existing callers (TestExpandTransitiveAndDedup,
// TestExpandDetectsCycle) to pass a nil third arg for commands.

func TestExpandCommandsTransitiveAndDedup(t *testing.T) {
	c, err := Load(graphFS(
		map[string][]string{"comet": {"superpowers", "openspec"}},
		map[string][]string{"comet": {"s"}, "superpowers": {"s"}, "openspec": {"s"}},
		map[string][]string{
			"comet":       {"comet-cmd"},
			"superpowers": {"brainstorm-cmd", "shared-cmd"},
			"openspec":    {"openspec-cmd", "shared-cmd"},
		},
	))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	got, err := c.ExpandCommands([]string{"comet"})
	if err != nil {
		t.Fatalf("expand commands: %v", err)
	}
	var names []string
	for _, e := range got {
		names = append(names, e.Name)
	}
	want := []string{"brainstorm-cmd", "comet-cmd", "openspec-cmd", "shared-cmd"}
	if strings.Join(names, ",") != strings.Join(want, ",") {
		t.Fatalf("expanded commands = %v, want %v", names, want)
	}
}

func TestExpandCommandsDetectsCycle(t *testing.T) {
	c, err := Load(graphFS(
		map[string][]string{"a": {"b"}, "b": {"a"}},
		map[string][]string{"a": {"sa"}, "b": {"sb"}},
		map[string][]string{"a": {"ca"}, "b": {"cb"}},
	))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if _, err := c.ExpandCommands([]string{"a"}); err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("expected cycle error, got %v", err)
	}
}
```

- [x] **Step 2: Run test to verify it fails**

Run: `go test ./internal/catalog/ -run TestExpandCommands -count=1`
Expected: FAIL (`ExpandCommands` undefined; `graphFS` arity mismatch until updated).

- [x] **Step 3: Refactor and implement**

Replace the body of `internal/catalog/expand.go` (keeping `ExpandedSkill`) with:

```go
package catalog

import (
	"fmt"
	"sort"
	"strings"
)

// ExpandedSkill is one skill reached by framework expansion, tagged with the
// framework it originated from (for later plan-origin notes).
type ExpandedSkill struct {
	Name      string
	Framework string
}

// ExpandedCommand is one command reached by framework expansion, tagged with
// the framework it originated from.
type ExpandedCommand struct {
	Name      string
	Framework string
}

// Expanded is one resource reached by framework expansion. It backs both
// Expand and ExpandCommands, which differ only in the resource map selected.
type Expanded struct {
	Name      string
	Framework string
}

// expandResources returns the transitive, deduplicated set of resources
// reachable from the given framework names — where sel picks a framework's
// resource map (Skills or Commands) — sorted by name, or an error naming a
// dependency cycle. A resource reachable via two frameworks collapses to one
// entry keyed by its first-seen origin. Cycle detection and dedup live here
// once, shared by Expand and ExpandCommands.
func (c *Catalog) expandResources(frameworkNames []string, sel func(Framework) map[string]string) ([]Expanded, error) {
	const (
		white = 0
		grey  = 1
		black = 2
	)
	color := map[string]int{}
	found := map[string]Expanded{}
	var stack []string

	var visit func(name string) error
	visit = func(name string) error {
		f, ok := c.frameworks[name]
		if !ok {
			return fmt.Errorf("catalog: unknown framework %q", name)
		}
		switch color[name] {
		case grey:
			return fmt.Errorf("catalog: framework dependency cycle: %s", strings.Join(append(stack, name), " -> "))
		case black:
			return nil
		}
		color[name] = grey
		stack = append(stack, name)
		for _, dep := range f.Dependencies {
			if err := visit(dep); err != nil {
				return err
			}
		}
		for res := range sel(f) {
			if _, seen := found[res]; !seen {
				found[res] = Expanded{Name: res, Framework: name}
			}
		}
		stack = stack[:len(stack)-1]
		color[name] = black
		return nil
	}

	for _, n := range frameworkNames {
		if err := visit(n); err != nil {
			return nil, err
		}
	}

	out := make([]Expanded, 0, len(found))
	for _, e := range found {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// Expand returns the transitive, deduplicated set of skills reachable from the
// given framework names, sorted by skill name, or an error naming a dependency
// cycle.
func (c *Catalog) Expand(frameworkNames []string) ([]ExpandedSkill, error) {
	res, err := c.expandResources(frameworkNames, func(f Framework) map[string]string { return f.Skills })
	if err != nil {
		return nil, err
	}
	out := make([]ExpandedSkill, len(res))
	for i, e := range res {
		out[i] = ExpandedSkill{Name: e.Name, Framework: e.Framework}
	}
	return out, nil
}

// ExpandCommands returns the transitive, deduplicated set of commands reachable
// from the given framework names, sorted by command name, or an error naming a
// dependency cycle.
func (c *Catalog) ExpandCommands(frameworkNames []string) ([]ExpandedCommand, error) {
	res, err := c.expandResources(frameworkNames, func(f Framework) map[string]string { return f.Commands })
	if err != nil {
		return nil, err
	}
	out := make([]ExpandedCommand, len(res))
	for i, e := range res {
		out[i] = ExpandedCommand{Name: e.Name, Framework: e.Framework}
	}
	return out, nil
}
```

(The `white` const is now unused but retained for parity/readability; unused constants are legal in Go.)

- [x] **Step 4: Run test to verify it passes**

Run: `go test ./internal/catalog/ -count=1`
Expected: PASS (both new command tests and the unchanged skill `Expand` tests).

- [x] **Step 5: Commit**

```bash
git add internal/catalog/expand.go internal/catalog/expand_test.go
git commit -m "feat(catalog): factor expandResources and add ExpandCommands"
```

---

### Task 4: Single-file command materialization

Adds `MaterializeCommands`, the single-file sibling of `Materialize`: it overwrites `dstRoot/<name>.md` per command. No `RemoveAll` is needed — a single-file overwrite handles upgrades.

**Files:**
- Modify: `internal/catalog/materialize.go`
- Test: `internal/catalog/materialize_test.go`

**Interfaces:**
- Consumes: `Catalog.commands`, `Catalog.fsys`.
- Produces: `func (c *Catalog) MaterializeCommands(dstRoot string, names []string) error`.

- [ ] **Step 1: Write the failing test**

Add to `internal/catalog/materialize_test.go`. Extend `matFS()` to declare a command:

```go
// In matFS(), add a [commands] table to the sp framework.toml and the file:
//
//   "frameworks/sp/framework.toml": {Data: []byte(`name = "sp"
// version = "0.1.0"
// [skills]
// brainstorming = "skills/brainstorming"
// [commands]
// demo-cmd = "commands/demo-cmd.md"
// `)},
//   "commands/demo-cmd.md": {Data: []byte("command body")},

func TestMaterializeCommandsWritesFile(t *testing.T) {
	c, err := Load(matFS())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	dst := t.TempDir()
	if err := c.MaterializeCommands(dst, []string{"demo-cmd"}); err != nil {
		t.Fatalf("materialize commands: %v", err)
	}
	if b, _ := os.ReadFile(filepath.Join(dst, "demo-cmd.md")); string(b) != "command body" {
		t.Fatalf("demo-cmd.md = %q", b)
	}
}

func TestMaterializeCommandsOverwrites(t *testing.T) {
	c, _ := Load(matFS())
	dst := t.TempDir()
	os.WriteFile(filepath.Join(dst, "demo-cmd.md"), []byte("STALE"), 0o644)
	if err := c.MaterializeCommands(dst, []string{"demo-cmd"}); err != nil {
		t.Fatalf("materialize commands: %v", err)
	}
	if b, _ := os.ReadFile(filepath.Join(dst, "demo-cmd.md")); string(b) != "command body" {
		t.Fatalf("stale command not overwritten: %q", b)
	}
}

func TestMaterializeCommandsUnknownErrors(t *testing.T) {
	c, _ := Load(matFS())
	if err := c.MaterializeCommands(t.TempDir(), []string{"nope"}); err == nil {
		t.Fatal("expected error for unknown command")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/catalog/ -run TestMaterializeCommands -count=1`
Expected: FAIL (`MaterializeCommands` undefined).

- [ ] **Step 3: Implement `MaterializeCommands`**

Append to `internal/catalog/materialize.go`:

```go
// MaterializeCommands writes each named builtin command from the embedded FS to
// dstRoot/<name>.md (a single file), replacing any existing file. Unlike
// Materialize (per-skill directories), no RemoveAll is needed — a single-file
// overwrite fully replaces prior content on upgrade. It is the caller's job
// (engine) to gate this on the catalog version.
func (c *Catalog) MaterializeCommands(dstRoot string, names []string) error {
	for _, name := range names {
		cp, ok := c.commands[name]
		if !ok {
			return fmt.Errorf("catalog: unknown command %q", name)
		}
		data, err := fs.ReadFile(c.fsys, cp)
		if err != nil {
			return fmt.Errorf("catalog: read %q: %w", cp, err)
		}
		if err := os.MkdirAll(dstRoot, 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dstRoot, name+".md"), data, 0o644); err != nil {
			return err
		}
	}
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/catalog/ -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/catalog/materialize.go internal/catalog/materialize_test.go
git commit -m "feat(catalog): materialize builtin commands as single files"
```

---

### Task 5: `internal/commandpath` package

New package mirroring `skillpath.Dir`, mapping tool + scope to the command directory. OpenCode uses the singular `command/`. Scope flipping reuses `skillpath.Other` (pure scope logic), so `commandpath` needs only `Dir`.

**Files:**
- Create: `internal/commandpath/commandpath.go`
- Test: `internal/commandpath/commandpath_test.go`

**Interfaces:**
- Produces: `func Dir(tool, scope, home, projectRoot string) string`.

- [ ] **Step 1: Write the failing test**

Create `internal/commandpath/commandpath_test.go`:

```go
package commandpath

import (
	"path/filepath"
	"testing"
)

func TestDir(t *testing.T) {
	home := filepath.Join("/home", "u")
	proj := filepath.Join("/work", "repo")
	cases := []struct {
		tool, scope string
		want        string
	}{
		{"claude", "user", filepath.Join(home, ".claude", "commands")},
		{"claude", "project", filepath.Join(proj, ".claude", "commands")},
		{"opencode", "user", filepath.Join(home, ".config", "opencode", "command")},
		{"opencode", "project", filepath.Join(proj, ".opencode", "command")},
		// Non-"project" scope (empty, unknown) is treated as user.
		{"claude", "", filepath.Join(home, ".claude", "commands")},
		{"opencode", "whatever", filepath.Join(home, ".config", "opencode", "command")},
		// Unknown tool returns "".
		{"nope", "user", ""},
	}
	for _, c := range cases {
		if got := Dir(c.tool, c.scope, home, proj); got != c.want {
			t.Errorf("Dir(%q,%q) = %q; want %q", c.tool, c.scope, got, c.want)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/commandpath/ -count=1`
Expected: FAIL (package/`Dir` does not exist).

- [ ] **Step 3: Implement the package**

Create `internal/commandpath/commandpath.go`:

```go
// Package commandpath is the single source of truth for where each tool's owned
// commands are linked, as a function of the install scope. It parallels
// skillpath; a future change may unify both into a resourcepath.Dir(kind, …).
// Scope flipping (for inactive-scope pruning) reuses skillpath.Other, so this
// package exposes only Dir.
package commandpath

import "path/filepath"

// Dir returns the directory a tool's owned commands are linked into.
//
//	claude   + user     -> <home>/.claude/commands
//	claude   + project  -> <projectRoot>/.claude/commands
//	opencode + user     -> <home>/.config/opencode/command
//	opencode + project  -> <projectRoot>/.opencode/command
//
// OpenCode uses the SINGULAR "command" directory (unlike its plural "skills").
// Any scope other than "project" is treated as "user". An unknown tool
// returns "".
func Dir(tool, scope, home, projectRoot string) string {
	project := scope == "project"
	switch tool {
	case "claude":
		if project {
			return filepath.Join(projectRoot, ".claude", "commands")
		}
		return filepath.Join(home, ".claude", "commands")
	case "opencode":
		if project {
			return filepath.Join(projectRoot, ".opencode", "command")
		}
		return filepath.Join(home, ".config", "opencode", "command")
	}
	return ""
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/commandpath/ -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/commandpath/
git commit -m "feat(commandpath): add command directory mapping"
```

---

### Task 6: Config command expansion

Adds `CommandEntriesForTool` (explicit only) and `ExpandedCommandEntriesForTool` (explicit + framework-expanded commands, scope/targets inherited, collision + cycle propagation), mirroring `ExpandedSkillEntriesForTool` and reusing the `loadedCatalog()` singleton.

**Files:**
- Modify: `internal/config/config.go`
- Test: `internal/config/config_test.go`

**Interfaces:**
- Consumes: `Config.Commands`, `Config.Frameworks`, `entriesForTool`, `sameResource`, `containsString`, `loadedCatalog`, `cat.Catalog.ExpandCommands`.
- Produces:
  - `func (c *Config) CommandEntriesForTool(tool string) []NamedResource`
  - `func (c *Config) ExpandedCommandEntriesForTool(tool string) ([]NamedResource, error)`

**Note on test coverage:** no framework in the *real* embedded catalog declares a `[commands]` table yet (deferred with real content), so the config-level framework-command **inheritance/collision** paths cannot be triggered against the real embed here. Their algorithm is byte-identical to skills (proven by `ExpandedSkill*` tests) and to the catalog `ExpandCommands` tests (Task 3). The config tests below exercise what the real embed supports: explicit command entries, target filtering, skill/command name-share, and that the framework loop safely no-ops when the framework declares no commands.

- [ ] **Step 1: Write the failing test**

Add to `internal/config/config_test.go` (uses `loadTOML` + `validModelsBothTools` helpers already in the file):

```go
func TestExpandedCommandsExplicitAndTargetFilter(t *testing.T) {
	c := loadTOML(t, `
[commands.example-command]
source = "builtin:example-command"
scope = "project"
targets = ["claude"]
`+validModelsBothTools())

	claude, err := c.ExpandedCommandEntriesForTool("claude")
	if err != nil {
		t.Fatalf("expand claude: %v", err)
	}
	if len(claude) != 1 || claude[0].Name != "example-command" {
		t.Fatalf("claude commands = %v", claude)
	}
	if claude[0].Resource.Source != "builtin:example-command" || claude[0].Resource.Scope != "project" {
		t.Fatalf("example-command resource = %+v", claude[0].Resource)
	}
	// targets = ["claude"] only -> opencode gets nothing.
	opencode, err := c.ExpandedCommandEntriesForTool("opencode")
	if err != nil {
		t.Fatalf("expand opencode: %v", err)
	}
	if len(opencode) != 0 {
		t.Fatalf("opencode commands = %v, want none", opencode)
	}
}

// A skill and a command may share a name: separate namespaces, both returned.
func TestSkillAndCommandMayShareName(t *testing.T) {
	c := loadTOML(t, `
[skills.shared]
source = "builtin:shared"
scope = "user"
targets = ["claude"]

[commands.shared]
source = "builtin:shared"
scope = "user"
targets = ["claude"]
`+validModelsBothTools())

	skills, err := c.ExpandedSkillEntriesForTool("claude")
	if err != nil {
		t.Fatalf("skills: %v", err)
	}
	commands, err := c.ExpandedCommandEntriesForTool("claude")
	if err != nil {
		t.Fatalf("commands: %v", err)
	}
	if len(skills) != 1 || skills[0].Name != "shared" {
		t.Fatalf("skills = %v", skills)
	}
	if len(commands) != 1 || commands[0].Name != "shared" {
		t.Fatalf("commands = %v", commands)
	}
}

// The framework loop must not crash or invent commands when the real framework
// declares no [commands] table: only explicit commands survive.
func TestExpandedCommandsFrameworkWithoutCommandsNoOps(t *testing.T) {
	c := loadTOML(t, `
[frameworks.comet]
source = "builtin:comet"
scope = "user"
targets = ["claude"]

[commands.example-command]
source = "builtin:example-command"
scope = "user"
targets = ["claude"]
`+validModelsBothTools())

	got, err := c.ExpandedCommandEntriesForTool("claude")
	if err != nil {
		t.Fatalf("expand: %v", err)
	}
	if len(got) != 1 || got[0].Name != "example-command" {
		t.Fatalf("commands = %v, want only example-command", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/config/ -run 'TestExpandedCommands|TestSkillAndCommandMayShareName' -count=1`
Expected: FAIL (`ExpandedCommandEntriesForTool` undefined).

- [ ] **Step 3: Implement the two methods**

In `internal/config/config.go`, add next to `SkillEntriesForTool`:

```go
func (c *Config) CommandEntriesForTool(tool string) []NamedResource {
	return entriesForTool(c.Commands, tool)
}
```

Add next to `ExpandedSkillEntriesForTool`:

```go
// ExpandedCommandEntriesForTool returns the effective commands for a tool:
// explicit [commands.X] entries plus, for each [frameworks.<fw>]
// source="builtin:<fw>" targeting the tool, its transitively expanded commands.
// Each expanded command inherits the framework declaration's scope and targets.
// A framework command whose name collides with an explicit [commands.X] entry,
// or with another framework's command under a conflicting declaration, is an
// error, as is a dependency cycle (surfaced from catalog.ExpandCommands).
// Collision is command-vs-command only: a command may share a name with a skill.
func (c *Config) ExpandedCommandEntriesForTool(tool string) ([]NamedResource, error) {
	byName := map[string]NamedResource{}
	explicitNames := map[string]bool{}
	for _, e := range c.CommandEntriesForTool(tool) {
		byName[e.Name] = e
		explicitNames[e.Name] = true
	}

	fwNames := make([]string, 0, len(c.Frameworks))
	for name := range c.Frameworks {
		fwNames = append(fwNames, name)
	}
	sort.Strings(fwNames)

	var cl *cat.Catalog
	for _, fwName := range fwNames {
		fwRes := c.Frameworks[fwName]
		if !strings.HasPrefix(fwRes.Source, "builtin:") {
			continue
		}
		if !containsString(fwRes.TargetsOrAll(), tool) {
			continue
		}
		if cl == nil {
			var err error
			if cl, err = loadedCatalog(); err != nil {
				return nil, err
			}
		}
		builtin := strings.TrimPrefix(fwRes.Source, "builtin:")
		expanded, err := cl.ExpandCommands([]string{builtin})
		if err != nil {
			return nil, fmt.Errorf("config: framework %q: %w", fwName, err)
		}
		for _, ec := range expanded {
			if explicitNames[ec.Name] {
				return nil, fmt.Errorf("config: command %q is declared both explicitly in [commands] and by framework %q", ec.Name, fwName)
			}
			nr := NamedResource{
				Name: ec.Name,
				Resource: Resource{
					Source:  "builtin:" + ec.Name,
					Scope:   fwRes.Scope,
					Targets: fwRes.Targets,
				},
			}
			if prev, ok := byName[ec.Name]; ok {
				if !sameResource(prev.Resource, nr.Resource) {
					return nil, fmt.Errorf("config: command %q expanded by multiple frameworks with conflicting scope/targets (framework %q)", ec.Name, fwName)
				}
				continue
			}
			byName[ec.Name] = nr
		}
	}

	out := make([]NamedResource, 0, len(byName))
	for _, nr := range byName {
		out = append(out, nr)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/config/ -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go
git commit -m "feat(config): expand framework and explicit commands per tool"
```

---

### Task 7: Engine command materialization

Extends `materializeCatalog` to collect declared builtin **command** names (across tools) alongside skills, materialize them into a new `CommandCatalogRoot` under the same version gate, and record `CatalogVersion` only after **both** succeed. `Build` computes the command root; `CommandDir()` exposes it for doctor. Adapters are wired with `WithCommandCatalogRoot` in Tasks 8–9 (they emit no command links until then, so materialization here stands alone).

**Files:**
- Modify: `internal/engine/engine.go`
- Test: `internal/engine/materialize_test.go`

**Interfaces:**
- Consumes: `config.ExpandedCommandEntriesForTool`, `catalog.MaterializeCommands`, `state.CatalogVersion*`.
- Produces:
  - `Engine.CommandCatalogRoot string` field.
  - `func (e *Engine) CommandDir() string`
  - local `commandCatalogDir` var in `Build` (used by Tasks 8–9 wiring).
  - `func allCommandFilesExist(root string, names []string) bool`

- [ ] **Step 1: Write the failing test**

Add to `internal/engine/materialize_test.go`. Add a command-declaring config constant near `cometTOML`:

```go
const commandTOML = `
[commands.example-command]
source = "builtin:example-command"
scope = "user"
targets = ["claude"]

[models.claude.architectural]
model = "opus"
variant = "max"
[models.claude.coding]
model = "sonnet"
effort = "n"
[models.claude.trivial]
model = "haiku"
effort = "f"
`

func TestApplyMaterializesBuiltinCommand(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	os.WriteFile(filepath.Join(repo, "homonto.toml"), []byte(commandTOML), 0o644)

	e := buildEngine(t, home, repo)
	sets, _ := e.Plan()
	if err := e.Apply(sets); err != nil {
		t.Fatalf("apply: %v", err)
	}

	got := filepath.Join(repo, ".homonto", "catalog", "commands", "example-command.md")
	if _, err := os.Stat(got); err != nil {
		t.Fatalf("example-command not materialized: %v", err)
	}
	if e.State.CatalogVersionRecorded() == "" {
		t.Fatal("catalog version not recorded after command materialization")
	}
}

func TestApplyRematerializesWhenCommandFileMissing(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	os.WriteFile(filepath.Join(repo, "homonto.toml"), []byte(commandTOML), 0o644)

	e := buildEngine(t, home, repo)
	sets, _ := e.Plan()
	if err := e.Apply(sets); err != nil {
		t.Fatalf("apply: %v", err)
	}
	cmdFile := filepath.Join(e.CommandDir(), "example-command.md")
	if err := os.Remove(cmdFile); err != nil {
		t.Fatal(err)
	}

	e2 := buildEngine(t, home, repo)
	sets2, _ := e2.Plan()
	if err := e2.Apply(sets2); err != nil {
		t.Fatalf("re-apply: %v", err)
	}
	if _, err := os.Stat(cmdFile); err != nil {
		t.Fatalf("command not restored after missing file triggered re-materialization: %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/engine/ -run 'TestApplyMaterializesBuiltinCommand|TestApplyRematerializesWhenCommandFileMissing' -count=1`
Expected: FAIL (`CommandDir` undefined; command file not materialized).

- [ ] **Step 3: Implement the engine changes**

In `internal/engine/engine.go`:

Add the field to `Engine`:

```go
	CatalogRoot        string // materialized builtin catalog root (<stateDir>/catalog/skills)
	CommandCatalogRoot string // materialized builtin command root (<stateDir>/catalog/commands)
```

In `Build`, after `catalogDir := filepath.Join(stateDir, "catalog", "skills")` add:

```go
	commandCatalogDir := filepath.Join(stateDir, "catalog", "commands")
```

In the returned `&Engine{...}` literal, after `CatalogRoot: catalogDir,` add:

```go
		CommandCatalogRoot: commandCatalogDir,
```

(Leave the two adapter constructor lines unchanged here; Tasks 8–9 append `.WithCommandCatalogRoot(commandCatalogDir)`.)

Add the accessor after `CatalogDir`:

```go
// CommandDir returns the materialized builtin command root.
func (e *Engine) CommandDir() string { return e.CommandCatalogRoot }
```

Replace `materializeCatalog` with the combined skills+commands version:

```go
// materializeCatalog extracts the builtin skills and commands the config
// declares into CatalogRoot and CommandCatalogRoot, version-gated: it is a
// no-op when the recorded catalog version matches the embedded one AND every
// skill dir and command file already exists. The version is recorded (and
// state saved) only after BOTH skills and commands materialize, so an
// interrupted extraction re-materializes on the next apply.
func (e *Engine) materializeCatalog() error {
	skillSet := map[string]bool{}
	cmdSet := map[string]bool{}
	for _, tool := range []string{"claude", "opencode"} {
		sEntries, err := e.Cfg.ExpandedSkillEntriesForTool(tool)
		if err != nil {
			return err
		}
		for _, entry := range sEntries {
			if strings.HasPrefix(entry.Resource.Source, "builtin:") {
				skillSet[strings.TrimPrefix(entry.Resource.Source, "builtin:")] = true
			}
		}
		cEntries, err := e.Cfg.ExpandedCommandEntriesForTool(tool)
		if err != nil {
			return err
		}
		for _, entry := range cEntries {
			if strings.HasPrefix(entry.Resource.Source, "builtin:") {
				cmdSet[strings.TrimPrefix(entry.Resource.Source, "builtin:")] = true
			}
		}
	}
	if len(skillSet) == 0 && len(cmdSet) == 0 {
		return nil
	}
	cl, err := catalog.New()
	if err != nil {
		return err
	}
	skillNames := make([]string, 0, len(skillSet))
	for n := range skillSet {
		skillNames = append(skillNames, n)
	}
	sort.Strings(skillNames)
	cmdNames := make([]string, 0, len(cmdSet))
	for n := range cmdSet {
		cmdNames = append(cmdNames, n)
	}
	sort.Strings(cmdNames)

	if e.State.CatalogVersionRecorded() == cl.Version() &&
		allSkillDirsExist(e.CatalogRoot, skillNames) &&
		allCommandFilesExist(e.CommandCatalogRoot, cmdNames) {
		return nil
	}
	if err := cl.Materialize(e.CatalogRoot, skillNames); err != nil {
		return err
	}
	if err := cl.MaterializeCommands(e.CommandCatalogRoot, cmdNames); err != nil {
		return err
	}
	e.State.SetCatalogVersion(cl.Version())
	// Save immediately so a later adapter failure still records the completed
	// materialization.
	return e.State.Save(e.StateDir)
}
```

Add the helper next to `allSkillDirsExist`:

```go
func allCommandFilesExist(root string, names []string) bool {
	for _, n := range names {
		fi, err := os.Stat(filepath.Join(root, n+".md"))
		if err != nil || fi.IsDir() {
			return false
		}
	}
	return true
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/engine/ -count=1`
Expected: PASS (new command tests plus unchanged skill materialization tests).

- [ ] **Step 5: Commit**

```bash
git add internal/engine/engine.go internal/engine/materialize_test.go
git commit -m "feat(engine): version-gated builtin command materialization"
```

---

### Task 8: Claude adapter command projection

Adds `command.<name>` link projection to the Claude adapter, mirroring skill projection exactly (create/update/adopt/prune, scope-switch relocate, ObserveHashes). Wires `WithCommandCatalogRoot` in `engine.Build`'s claude constructor and extends `managedRoots()`/`managedPrefix` for the command root/namespace.

**CRITICAL (learned from the skills change):**
- Every `link.Plan`/`Link`/`Remove`/`IsManaged` call for commands MUST pass `a.managedRoots()...`. The variadic signature compiles even if you forget — a silent bug.
- `managedRoots()` must include `a.commandCatalogRoot` ONLY when non-empty (empty root = prefix match for every path). Never pass an empty root.
- The adopt/no-op path compares the on-disk symlink target against `a.commandSource(entry)`.
- Command dst filenames are `<name>.md`; the state key is `command.<name>` (strip the `.md`). Use `strings.TrimSuffix(filepath.Base(op.Dst), ".md")`.

**Files:**
- Modify: `internal/adapter/claude/claude.go`
- Modify: `internal/adapter/claude/util.go` (`managedPrefix`)
- Modify: `internal/engine/engine.go` (Build claude line)
- Test: `internal/adapter/claude/builtin_test.go`

**Interfaces:**
- Consumes: `commandpath.Dir`, `skillpath.Other`, `config.ExpandedCommandEntriesForTool`, `link.*`, `state.State`.
- Produces:
  - `Adapter.commandCatalogRoot string`, `Adapter.commands []config.NamedResource`
  - `func (a *Adapter) WithCommandCatalogRoot(root string) *Adapter`
  - `func (a *Adapter) commandsDir(scope string) string`
  - `func (a *Adapter) inactiveCommandsDir(scope string) string`
  - `func (a *Adapter) commandSource(entry config.NamedResource) string`
  - `func (a *Adapter) commandLinks() map[string]string`

- [ ] **Step 1: Write the failing test**

Add to `internal/adapter/claude/builtin_test.go`:

```go
func builtinCmdCfg() *config.Config {
	return &config.Config{
		Commands: map[string]config.Resource{
			"example-command": {Source: "builtin:example-command", Scope: "user", Targets: []string{"claude"}},
		},
	}
}

func TestBuiltinCommandLinksToCommandCatalogRoot(t *testing.T) {
	home := t.TempDir()
	cmdRoot := t.TempDir()
	// Simulate materialization: the command file exists under the command root.
	os.WriteFile(filepath.Join(cmdRoot, "example-command.md"), []byte("body"), 0o644)

	a := New(home, t.TempDir()).WithCommandCatalogRoot(cmdRoot)
	st, _ := state.Load(t.TempDir())
	cs, err := a.Plan(builtinCmdCfg(), st)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if err := a.Apply(cs, resolver(), st); err != nil {
		t.Fatalf("apply: %v", err)
	}

	dst := filepath.Join(home, ".claude", "commands", "example-command.md")
	target, err := os.Readlink(dst)
	if err != nil {
		t.Fatalf("command link missing: %v", err)
	}
	if want := filepath.Join(cmdRoot, "example-command.md"); target != want {
		t.Fatalf("link target = %q, want %q", target, want)
	}
	if _, ok := st.Get("claude", "command.example-command"); !ok {
		t.Fatal("command.example-command not recorded in state")
	}
	// Re-plan is a noop for the link.
	cs2, _ := a.Plan(builtinCmdCfg(), st)
	for _, c := range cs2.Changes {
		if c.Key == "command.example-command" && c.Action != "noop" {
			t.Fatalf("re-plan not idempotent: %+v", c)
		}
	}
}

func TestBuiltinCommandPrunedWhenDeDeclared(t *testing.T) {
	home := t.TempDir()
	cmdRoot := t.TempDir()
	os.WriteFile(filepath.Join(cmdRoot, "example-command.md"), []byte("body"), 0o644)
	a := New(home, t.TempDir()).WithCommandCatalogRoot(cmdRoot)
	st, _ := state.Load(t.TempDir())

	cs, _ := a.Plan(builtinCmdCfg(), st)
	if err := a.Apply(cs, resolver(), st); err != nil {
		t.Fatalf("apply: %v", err)
	}
	dst := filepath.Join(home, ".claude", "commands", "example-command.md")

	cs2, err := a.Plan(&config.Config{}, st)
	if err != nil {
		t.Fatalf("plan empty: %v", err)
	}
	if err := a.Apply(cs2, resolver(), st); err != nil {
		t.Fatalf("apply empty: %v", err)
	}
	if _, err := os.Lstat(dst); !os.IsNotExist(err) {
		t.Fatal("de-declared builtin command link not pruned")
	}
}

func TestBuiltinCommandConflictNotClobbered(t *testing.T) {
	home := t.TempDir()
	cmdRoot := t.TempDir()
	os.WriteFile(filepath.Join(cmdRoot, "example-command.md"), []byte("body"), 0o644)
	a := New(home, t.TempDir()).WithCommandCatalogRoot(cmdRoot)
	st, _ := state.Load(t.TempDir())

	dst := filepath.Join(home, ".claude", "commands", "example-command.md")
	os.MkdirAll(filepath.Dir(dst), 0o755)
	os.WriteFile(dst, []byte("user file"), 0o644)

	if _, err := a.Plan(builtinCmdCfg(), st); err == nil {
		t.Fatal("expected conflict for real file at builtin command link dst")
	}
	if b, _ := os.ReadFile(dst); string(b) != "user file" {
		t.Fatal("conflict clobbered the user file")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/adapter/claude/ -run TestBuiltinCommand -count=1`
Expected: FAIL (`WithCommandCatalogRoot`/command projection do not exist).

- [ ] **Step 3: Add fields, constructor, roots, and helpers**

In `internal/adapter/claude/claude.go`:

Add fields to `Adapter`:

```go
type Adapter struct {
	home               string
	content            string
	catalogRoot        string // materialized builtin catalog root (.homonto/catalog/skills)
	commandCatalogRoot string // materialized builtin command root (.homonto/catalog/commands)
	projectRoot        string // directory of homonto.toml; used for project-scope resources
	skills             []config.NamedResource
	commands           []config.NamedResource
}
```

Add the constructor next to `WithCatalogRoot`:

```go
// WithCommandCatalogRoot sets the materialized builtin-command root that
// builtin:<name> commands link from. Mirrors WithCatalogRoot.
func (a *Adapter) WithCommandCatalogRoot(root string) *Adapter {
	a.commandCatalogRoot = root
	return a
}
```

Extend `managedRoots()` to include the command root (non-empty only):

```go
func (a *Adapter) managedRoots() []string {
	roots := []string{a.content}
	if a.catalogRoot != "" {
		roots = append(roots, a.catalogRoot)
	}
	if a.commandCatalogRoot != "" {
		roots = append(roots, a.commandCatalogRoot)
	}
	return roots
}
```

Add the command dir/source/links helpers next to `skillsDir`/`skillSource`/`links` (import `commandpath` — `github.com/noviopenworks/homonto/internal/commandpath`):

```go
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
```

- [ ] **Step 4: Extend `managedPrefix`**

In `internal/adapter/claude/util.go`, add `"command."` to the prefix list:

```go
func managedPrefix(k string) bool {
	for _, p := range []string{"mcp.", "setting.", "plugin.", "skill.", "command."} {
		if strings.HasPrefix(k, p) {
			return true
		}
	}
	return false
}
```

- [ ] **Step 5: Emit command ops in `Plan`**

In `Plan`, after `a.skills = skills` (near the top), add:

```go
	commands, err := c.ExpandedCommandEntriesForTool("claude")
	if err != nil {
		return adapter.ChangeSet{}, err
	}
	a.commands = commands
```

After the skill adopt loop and BEFORE the orphan-delete loop, add the parallel command blocks:

```go
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
```

In the `declared` map population (orphan-delete section), after the skills loop add:

```go
	for _, entry := range a.commands {
		declared["command."+entry.Name] = true
	}
```

(The existing orphan-delete loop over `st.Keys("claude")` now prunes de-declared `command.*` keys automatically, since `managedPrefix` recognizes them.)

- [ ] **Step 6: Handle command ops in `Apply`**

In `Apply`'s adopt branch, after the `if hasPrefix(c.Key, "skill.") { ... }` block add:

```go
			if hasPrefix(c.Key, "command.") {
				st.Set("claude", c.Key, c.New, secret.Hash(c.New))
				continue
			}
```

In the `delete` switch, add a case parallel to `skill.`:

```go
			case hasPrefix(c.Key, "command."):
				name := trim(c.Key, "command.")
				dst := ""
				if e, ok := st.Get("claude", c.Key); ok {
					dst, _ = recordedDst(e.Desired)
				}
				if dst == "" {
					dst = filepath.Join(a.commandsDir("user"), name+".md")
				}
				err = link.Remove(dst, a.managedRoots()...)
```

After the JSON `skill.` skip line (`if hasPrefix(c.Key, "skill.") { continue }`) add:

```go
		if hasPrefix(c.Key, "command.") {
			continue
		}
```

After the skill inactive-prune loop and skill link-creation loop (at the end of `Apply`, before `return nil`), add the command equivalents:

```go
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
	// Fail fast on command link conflicts before creating any link.
	cmdLinks := a.commandLinks()
	if _, err := link.Plan(cmdLinks, a.managedRoots()...); err != nil {
		return err
	}
	for dst, src := range cmdLinks {
		if _, err := link.Link(src, dst, a.managedRoots()...); err != nil {
			return err
		}
		st.Set("claude", "command."+strings.TrimSuffix(filepath.Base(dst), ".md"), dst+" -> "+src, secret.Hash(dst+" -> "+src))
	}
	return nil
```

- [ ] **Step 7: Handle `command.` in `ObserveHashes`**

In `ObserveHashes`, immediately after the `if hasPrefix(key, "skill.") { ... }` block, add:

```go
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
```

- [ ] **Step 8: Wire `WithCommandCatalogRoot` in `engine.Build`**

In `internal/engine/engine.go`, append `.WithCommandCatalogRoot(commandCatalogDir)` to the claude constructor line:

```go
			claude.New(home, contentDir).WithProjectRoot(projectRoot).WithCatalogRoot(catalogDir).WithCommandCatalogRoot(commandCatalogDir),
```

- [ ] **Step 9: Run tests to verify they pass**

Run: `go test ./internal/adapter/claude/ ./internal/engine/ -count=1`
Expected: PASS

- [ ] **Step 10: Commit**

```bash
git add internal/adapter/claude/claude.go internal/adapter/claude/util.go internal/adapter/claude/builtin_test.go internal/engine/engine.go
git commit -m "feat(claude): project builtin and local commands"
```

---

### Task 9: OpenCode adapter command projection

Identical command projection for OpenCode, using `commandpath` (singular `command/`) and OpenCode's `Plan`/`Apply`/`ObserveHashes` structure. Same CRITICAL rules as Task 8 apply (variadic `a.managedRoots()...` on every `link.*`; non-empty command root; adopt compares against `commandSource`; strip `.md` for the state key).

**Files:**
- Modify: `internal/adapter/opencode/opencode.go`
- Modify: `internal/adapter/opencode/util.go` (`managedPrefix`)
- Modify: `internal/engine/engine.go` (Build opencode line)
- Test: `internal/adapter/opencode/builtin_test.go`

**Interfaces:** same set as Task 8, on the OpenCode `*Adapter`.

- [ ] **Step 1: Write the failing test**

Add to `internal/adapter/opencode/builtin_test.go` (note the OpenCode user command dir is `.config/opencode/command`):

```go
func builtinCmdCfg() *config.Config {
	return &config.Config{
		Commands: map[string]config.Resource{
			"example-command": {Source: "builtin:example-command", Scope: "user", Targets: []string{"opencode"}},
		},
	}
}

func TestBuiltinCommandLinksToCommandCatalogRoot(t *testing.T) {
	home := t.TempDir()
	cmdRoot := t.TempDir()
	os.WriteFile(filepath.Join(cmdRoot, "example-command.md"), []byte("body"), 0o644)

	a := New(home, t.TempDir()).WithCommandCatalogRoot(cmdRoot)
	st, _ := state.Load(t.TempDir())
	cs, err := a.Plan(builtinCmdCfg(), st)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if err := a.Apply(cs, resolver(), st); err != nil {
		t.Fatalf("apply: %v", err)
	}

	dst := filepath.Join(home, ".config", "opencode", "command", "example-command.md")
	target, err := os.Readlink(dst)
	if err != nil {
		t.Fatalf("command link missing: %v", err)
	}
	if want := filepath.Join(cmdRoot, "example-command.md"); target != want {
		t.Fatalf("link target = %q, want %q", target, want)
	}
	if _, ok := st.Get("opencode", "command.example-command"); !ok {
		t.Fatal("command.example-command not recorded in state")
	}
	cs2, _ := a.Plan(builtinCmdCfg(), st)
	for _, c := range cs2.Changes {
		if c.Key == "command.example-command" && c.Action != "noop" {
			t.Fatalf("re-plan not idempotent: %+v", c)
		}
	}
}

func TestBuiltinCommandPrunedWhenDeDeclared(t *testing.T) {
	home := t.TempDir()
	cmdRoot := t.TempDir()
	os.WriteFile(filepath.Join(cmdRoot, "example-command.md"), []byte("body"), 0o644)
	a := New(home, t.TempDir()).WithCommandCatalogRoot(cmdRoot)
	st, _ := state.Load(t.TempDir())

	cs, _ := a.Plan(builtinCmdCfg(), st)
	if err := a.Apply(cs, resolver(), st); err != nil {
		t.Fatalf("apply: %v", err)
	}
	dst := filepath.Join(home, ".config", "opencode", "command", "example-command.md")

	cs2, err := a.Plan(&config.Config{}, st)
	if err != nil {
		t.Fatalf("plan empty: %v", err)
	}
	if err := a.Apply(cs2, resolver(), st); err != nil {
		t.Fatalf("apply empty: %v", err)
	}
	if _, err := os.Lstat(dst); !os.IsNotExist(err) {
		t.Fatal("de-declared builtin command link not pruned")
	}
}

func TestBuiltinCommandConflictNotClobbered(t *testing.T) {
	home := t.TempDir()
	cmdRoot := t.TempDir()
	os.WriteFile(filepath.Join(cmdRoot, "example-command.md"), []byte("body"), 0o644)
	a := New(home, t.TempDir()).WithCommandCatalogRoot(cmdRoot)
	st, _ := state.Load(t.TempDir())

	dst := filepath.Join(home, ".config", "opencode", "command", "example-command.md")
	os.MkdirAll(filepath.Dir(dst), 0o755)
	os.WriteFile(dst, []byte("user file"), 0o644)

	if _, err := a.Plan(builtinCmdCfg(), st); err == nil {
		t.Fatal("expected conflict for real file at builtin command link dst")
	}
	if b, _ := os.ReadFile(dst); string(b) != "user file" {
		t.Fatal("conflict clobbered the user file")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/adapter/opencode/ -run TestBuiltinCommand -count=1`
Expected: FAIL (`WithCommandCatalogRoot`/command projection do not exist).

- [ ] **Step 3: Add fields, constructor, roots, and helpers**

In `internal/adapter/opencode/opencode.go`, mirror Task 8 Step 3 with `"opencode"` as the tool. Add `commandCatalogRoot string` and `commands []config.NamedResource` to `Adapter`; add `WithCommandCatalogRoot`; extend `managedRoots()` with the non-empty `a.commandCatalogRoot`; add `commandsDir`, `inactiveCommandsDir`, `commandSource`, `commandLinks` (import `github.com/noviopenworks/homonto/internal/commandpath`):

```go
func (a *Adapter) WithCommandCatalogRoot(root string) *Adapter {
	a.commandCatalogRoot = root
	return a
}

func (a *Adapter) managedRoots() []string {
	roots := []string{a.content}
	if a.catalogRoot != "" {
		roots = append(roots, a.catalogRoot)
	}
	if a.commandCatalogRoot != "" {
		roots = append(roots, a.commandCatalogRoot)
	}
	return roots
}

func (a *Adapter) commandsDir(scope string) string {
	return commandpath.Dir("opencode", scope, a.home, a.projectRoot)
}

func (a *Adapter) inactiveCommandsDir(scope string) string {
	if a.projectRoot == "" {
		return ""
	}
	d := commandpath.Dir("opencode", skillpath.Other(scope), a.home, a.projectRoot)
	if d == a.commandsDir(scope) {
		return ""
	}
	return d
}

func (a *Adapter) commandSource(entry config.NamedResource) string {
	if s := entry.Resource.Source; strings.HasPrefix(s, "builtin:") {
		return filepath.Join(a.commandCatalogRoot, strings.TrimPrefix(s, "builtin:")+".md")
	}
	return filepath.Join(a.content, "commands", localSourceName(entry.Resource.Source, entry.Name)+".md")
}

func (a *Adapter) commandLinks() map[string]string {
	out := map[string]string{}
	for _, entry := range a.commands {
		out[filepath.Join(a.commandsDir(entry.Resource.Scope), entry.Name+".md")] = a.commandSource(entry)
	}
	return out
}
```

- [ ] **Step 4: Extend `managedPrefix`**

In `internal/adapter/opencode/util.go`:

```go
func managedPrefix(k string) bool {
	for _, p := range []string{"mcp.", "setting.", "plugin.", "skill.", "command."} {
		if strings.HasPrefix(k, p) {
			return true
		}
	}
	return false
}
```

- [ ] **Step 5: Emit command ops in `Plan`**

After `a.skills = skills`, add:

```go
	commands, err := c.ExpandedCommandEntriesForTool("opencode")
	if err != nil {
		return adapter.ChangeSet{}, err
	}
	a.commands = commands
```

After the skill adopt loop and before the orphan-delete section, add the same command block as Task 8 Step 5 but with `"opencode"` in the `st.Get(...)` calls:

```go
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
		if e, ok := st.Get("opencode", "command."+entry.Name); ok && e.Applied == secret.Hash(dst+" -> "+src) {
			continue
		}
		cs.Changes = append(cs.Changes, adapter.Change{Action: "adopt", Key: "command." + entry.Name, New: dst + " -> " + src})
	}
```

In the `declared` map population, add:

```go
	for _, entry := range a.commands {
		declared["command."+entry.Name] = true
	}
```

- [ ] **Step 6: Handle command ops in `Apply`**

In the adopt branch, after the skill adopt block:

```go
			if hasPrefix(c.Key, "command.") {
				st.Set("opencode", c.Key, c.New, secret.Hash(c.New))
				continue
			}
```

In the delete switch:

```go
			case hasPrefix(c.Key, "command."):
				name := trim(c.Key, "command.")
				dst := ""
				if e, ok := st.Get("opencode", c.Key); ok {
					dst, _ = recordedDst(e.Desired)
				}
				if dst == "" {
					dst = filepath.Join(a.commandsDir("user"), name+".md")
				}
				err = link.Remove(dst, a.managedRoots()...)
```

After the skill JSON-skip:

```go
		if hasPrefix(c.Key, "command.") {
			continue
		}
```

At the end of `Apply` (after the skill link loop, before `return nil`):

```go
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
	cmdLinks := a.commandLinks()
	if _, err := link.Plan(cmdLinks, a.managedRoots()...); err != nil {
		return err
	}
	for dst, src := range cmdLinks {
		if _, err := link.Link(src, dst, a.managedRoots()...); err != nil {
			return err
		}
		st.Set("opencode", "command."+strings.TrimSuffix(filepath.Base(dst), ".md"), dst+" -> "+src, secret.Hash(dst+" -> "+src))
	}
	return nil
```

- [ ] **Step 7: Handle `command.` in `ObserveHashes`**

In `ObserveHashes`'s switch, add a case after the `skill.` case:

```go
		case hasPrefix(key, "command."):
			e, ok := st.Get("opencode", key)
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
```

- [ ] **Step 8: Wire `WithCommandCatalogRoot` in `engine.Build`**

Append `.WithCommandCatalogRoot(commandCatalogDir)` to the opencode constructor line in `internal/engine/engine.go`:

```go
			opencode.New(home, contentDir).WithProjectRoot(projectRoot).WithCatalogRoot(catalogDir).WithCommandCatalogRoot(commandCatalogDir),
```

- [ ] **Step 9: Run tests to verify they pass**

Run: `go test ./internal/adapter/opencode/ ./internal/engine/ -count=1`
Expected: PASS

- [ ] **Step 10: Commit**

```bash
git add internal/adapter/opencode/opencode.go internal/adapter/opencode/util.go internal/adapter/opencode/builtin_test.go internal/engine/engine.go
git commit -m "feat(opencode): project builtin and local commands"
```

---

### Task 10: Doctor command verification

Extends `Doctor` to verify command content presence + tool-side symlink for each expanded command, mirroring `doctorSkills`.

**Files:**
- Modify: `internal/engine/status.go`
- Test: `internal/engine/status_test.go`

**Interfaces:**
- Consumes: `config.ExpandedCommandEntriesForTool`, `Engine.CommandDir()`, `commandpath.Dir`.
- Produces: `func (e *Engine) doctorCommands(tool string, entries []config.NamedResource) []string`.

- [ ] **Step 1: Write the failing test**

Add to `internal/engine/status_test.go` (mirror the existing doctor skill test's setup pattern; declare the placeholder builtin command, apply, then assert doctor reports it linked):

```go
func TestDoctorReportsLinkedCommand(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	os.WriteFile(filepath.Join(repo, "homonto.toml"), []byte(commandTOML), 0o644)

	e := buildEngine(t, home, repo)
	sets, _ := e.Plan()
	if err := e.Apply(sets); err != nil {
		t.Fatalf("apply: %v", err)
	}

	var found bool
	for _, line := range e.Doctor() {
		if strings.Contains(line, "ok: command \"example-command\" linked (claude)") {
			found = true
		}
	}
	if !found {
		t.Fatalf("doctor did not report example-command linked; got %v", e.Doctor())
	}
}
```

(If `commandTOML` is defined in `materialize_test.go` in the same `engine` package, it is reused directly. `strings` is already imported by `status_test.go`; add it if not.)

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/engine/ -run TestDoctorReportsLinkedCommand -count=1`
Expected: FAIL (doctor does not report commands).

- [ ] **Step 3: Implement `doctorCommands` and wire it into `Doctor`**

In `internal/engine/status.go`, add the `commandpath` import (`github.com/noviopenworks/homonto/internal/commandpath`). In `Doctor`, after the opencode skills block, add:

```go
	claudeCommands, ccerr := e.Cfg.ExpandedCommandEntriesForTool("claude")
	if ccerr != nil {
		out = append(out, fmt.Sprintf("warn: cannot expand claude commands: %v", ccerr))
	} else {
		out = append(out, e.doctorCommands("claude", claudeCommands)...)
	}
	opencodeCommands, ocerr := e.Cfg.ExpandedCommandEntriesForTool("opencode")
	if ocerr != nil {
		out = append(out, fmt.Sprintf("warn: cannot expand opencode commands: %v", ocerr))
	} else {
		out = append(out, e.doctorCommands("opencode", opencodeCommands)...)
	}
```

Add the helper after `doctorSkills`:

```go
// doctorCommands reports, per command, whether its content file is present at
// the right source (builtin: from the materialized command root, local: from
// the content dir) and whether it is linked into the tool's command directory.
func (e *Engine) doctorCommands(tool string, entries []config.NamedResource) []string {
	var out []string
	for _, entry := range entries {
		name := entry.Name
		var p string
		if strings.HasPrefix(entry.Resource.Source, "builtin:") {
			p = filepath.Join(e.CommandDir(), strings.TrimPrefix(entry.Resource.Source, "builtin:")+".md")
		} else {
			sourceName := name
			if strings.HasPrefix(entry.Resource.Source, "local:") {
				sourceName = strings.TrimPrefix(entry.Resource.Source, "local:")
			}
			p = filepath.Join(e.ContentDir, "commands", sourceName+".md")
		}
		if _, err := os.Stat(p); err != nil {
			out = append(out, fmt.Sprintf("warn: command %q missing from %s (run apply)", name, p))
			continue
		}
		dst := filepath.Join(commandpath.Dir(tool, entry.Resource.Scope, e.Home, e.ProjectRoot), name+".md")
		if target, err := os.Readlink(dst); err == nil && target == p {
			out = append(out, fmt.Sprintf("ok: command %q linked (%s)", name, tool))
		} else {
			out = append(out, fmt.Sprintf("warn: command %q content present, not linked for %s (run apply)", name, tool))
		}
	}
	return out
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/engine/ -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/engine/status.go internal/engine/status_test.go
git commit -m "feat(doctor): verify command content and links"
```

---

### Task 11: Dogfood the placeholder command

Declares the placeholder command in the repo's own `homonto.toml` (builtin, project scope) and runs the real binary to confirm materialize + link + status + doctor end-to-end. `.homonto/`, `.claude/`, `.opencode/` are gitignored — only `homonto.toml` is committed.

**Files:**
- Modify: `homonto.toml`

**Interfaces:**
- Consumes: the full pipeline built in Tasks 1–10 via the `homonto` CLI.

- [ ] **Step 1: Declare the command in `homonto.toml`**

Append to `/home/mg/homonto/homonto.toml`:

```toml
[commands.example-command]
source = "builtin:example-command"
scope = "project"
```

(No `targets` → defaults to both claude and opencode; existing `[models.*]` for both tools already satisfy model validation.)

- [ ] **Step 2: Build and apply**

```bash
go build -o /tmp/homonto ./cmd/homonto
cd /home/mg/homonto && /tmp/homonto apply --yes
```

Expected: apply succeeds; command materialized and linked. Verify:

```bash
test -f /home/mg/homonto/.homonto/catalog/commands/example-command.md && echo MATERIALIZED
readlink /home/mg/homonto/.claude/commands/example-command.md
readlink /home/mg/homonto/.opencode/command/example-command.md
```

Expected: `MATERIALIZED`; both readlinks print `.../.homonto/catalog/commands/example-command.md`.

(If the CLI subcommand/flag names differ from `apply --yes`, use the repo's actual apply invocation — check `cmd/homonto` help. The observable outcomes above are what matters.)

- [ ] **Step 3: Status and doctor**

```bash
cd /home/mg/homonto && /tmp/homonto status && /tmp/homonto doctor
```

Expected: `status` reports no drift and no pending for the command; `doctor` prints `ok: command "example-command" linked (claude)` and `... (opencode)`.

- [ ] **Step 4: Commit (config only)**

```bash
git add homonto.toml
git commit -m "chore: dogfood placeholder command projection"
```

---

### Task 12: Regression and docs

Full regression sweep, stale-doc check, and roadmap status update for v1.1.

**Files:**
- Modify: `docs/roadmap.md`

- [ ] **Step 1: Full regression**

```bash
cd /home/mg/homonto && go test ./... -count=1 && go vet ./... && go build ./...
```

Expected: all pass, no vet findings, clean build.

- [ ] **Step 2: Stale-doc grep**

```bash
cd /home/mg/homonto && grep -rn -i "command" docs/roadmap.md | grep -i "not.*implement\|unimplement\|deferred\|todo\|future" || echo "no stale command claims"
```

Review hits: no doc should now claim command projection is unimplemented. The v1.1 scope line "Projection for skills, commands, and subagents..." remains accurate (subagents still pending; commands now landed as machinery + placeholder).

- [ ] **Step 3: Update roadmap v1.1 status**

In `docs/roadmap.md`, under the v1.1 section, add a status note reflecting that command projection machinery has landed with a placeholder while real command content is deferred. For example, append to the v1.1 Scope area:

```markdown
- Status: skill projection and single-file command projection (builtin + local,
  both tools) are implemented and dogfooded via a placeholder command; real
  bundled command content and framework-declared `[commands]` tables are
  deferred to a later change. Subagent projection remains pending.
```

(Match the surrounding doc's exact heading/format; keep the claim precise — machinery landed, content deferred.)

- [ ] **Step 4: Commit**

```bash
git add docs/roadmap.md
git commit -m "docs: mark command projection machinery landed in v1.1"
```

---

## Self-Review

**Spec coverage** (delta specs + design doc §1–§11):

- command-projection §"Builtin/local source resolution" → Task 8/9 `commandSource` (builtin→`.homonto/catalog/commands/<n>.md`, local→`homonto/commands/<n>.md`); scope required is enforced by existing `validateResources` (unchanged). ✓
- command-projection §"Single-file materialization" (version-gated, record after success) → Task 4 + Task 7. ✓
- command-projection §"Projection into tool command dirs" (claude `commands/`, opencode user `command/` / project `.opencode/command`; create/update/no-op; record; prune only managed symlink; conflict never clobbered) → Task 5 (paths) + Task 8/9 (link ops, prune, conflict). ✓
- command-projection §"Framework command expansion" (inherit scope/targets, transitive, dedup, explicit collision) → Task 3 (catalog) + Task 6 (config). ✓
- command-projection §"Doctor verification" → Task 10. ✓
- command-projection §"Placeholder fixture command" (exactly one) → Task 1 + Task 11. ✓
- config-model §"Local provider content root" (local commands from `homonto/commands/<n>.md`, builtin from materialized root) → Task 8/9 `commandSource`. ✓
- framework-expansion §"Framework metadata format" ([commands] table, `commands/<n>.md` paths) → Task 2. ✓
- Design §2 `commandpath.Dir` → Task 5. ✓  §3 `CommandEntriesForTool` + `ExpandedCommandEntriesForTool` → Task 6. ✓  §4 engine (`materializeCatalog`, `CommandDir`, version gate over both) → Task 7. ✓  §5 adapters (`commandCatalogRoot`, `WithCommandCatalogRoot`, `commandsDir`, `commandSource`, `managedRoots` = {content, catalogRoot, commandCatalogRoot}, `command.<n>` state) → Task 8/9. ✓  §9 edge cases (interrupted materialization, empty command set, skill/command name-share) → covered by version-gate save ordering (Task 7), `len(...)==0` guard (Task 7), separate state keys/dirs (Task 8/9) + name-share test (Task 6). ✓  §11 no spec patches. ✓

**Placeholder scan:** No "TBD"/"handle edge cases"/"similar to Task N" — every code step shows concrete code; the one deliberate deferral (config-level framework-command inheritance/collision tests) is explained with its coverage rationale, not left as a gap. The roadmap wording in Task 12 Step 3 is intentionally match-the-surrounding-format since the exact heading text is environmental.

**Type consistency:** `expandResources`/`Expanded`/`ExpandCommands`/`ExpandedCommand` (Task 3) consumed by `ExpandedCommandEntriesForTool` (Task 6). `MaterializeCommands(dstRoot, names)` (Task 4) called by engine (Task 7) with `e.CommandCatalogRoot`. `WithCommandCatalogRoot` (Task 8/9) wired with `commandCatalogDir` (Task 7). `commandpath.Dir` (Task 5) used by adapters (Task 8/9) and doctor (Task 10). State key `command.<name>` (no `.md`) consistent across Plan/Apply/ObserveHashes/prune and `managedPrefix`. `commandSource` filename `<n>.md` consistent between adapter and doctor.

## Execution Handoff

**Plan complete and saved to `docs/superpowers/plans/2026-07-10-command-projection.md`. Two execution options:**

**1. Subagent-Driven (recommended)** — I dispatch a fresh subagent per task, review between tasks, fast iteration.

**2. Inline Execution** — Execute tasks in this session using executing-plans, batch execution with checkpoints.

**Which approach?**
