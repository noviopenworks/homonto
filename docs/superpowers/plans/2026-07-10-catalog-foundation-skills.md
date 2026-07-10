---
change: catalog-foundation-skills
design-doc: docs/superpowers/specs/2026-07-10-catalog-foundation-skills-design.md
base-ref: bc85fa2e4de8b03447b73ef040a8b60edb04c627
---

# Catalog Foundation (Skills) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bundle a `go:embed`ded catalog of frameworks and skills into the homonto binary, expand `[frameworks.X]` declarations into their constituent builtin skills, materialize that content to `.homonto/catalog/skills/`, and symlink it into each tool the same way local skills already link.

**Architecture:** A new root `catalog/` package embeds framework metadata (`framework.toml`) and skill content and exposes only an `embed.FS`. A new `internal/catalog` package holds all logic: `Load` (parse + validate), `Expand` (transitive framework→skill graph with cycle detection), and `Materialize` (embedded FS → on-disk cache). `internal/config` gains `ExpandedSkillEntriesForTool` (explicit skills plus framework expansion, with collision/cycle errors). `internal/link` generalizes its single managed-root check to a variadic set of roots so links into `.homonto/catalog/skills/` are treated as "ours". `internal/state` records the catalog version. The engine materializes builtin skills (version-gated) before adapters link them. Both adapters resolve `builtin:<name>` sources to the materialized catalog root.

**Tech Stack:** Go 1.23 (module `github.com/noviopenworks/homonto`), `github.com/pelletier/go-toml/v2` for TOML, standard-library `embed`/`io/fs`/`testing/fstest`. No new dependencies.

## Global Constraints

- Module path is `github.com/noviopenworks/homonto`; import the root catalog package as `github.com/noviopenworks/homonto/catalog` and the logic package as `github.com/noviopenworks/homonto/internal/catalog`.
- **Layering rule (prevents an import cycle):** `internal/catalog` MUST NOT import `internal/config`. The root `catalog` package imports nothing but `embed`. `internal/config` imports `internal/catalog`; `internal/engine` imports config + adapters + `internal/catalog`.
- Catalog version string lives in `catalog/version.txt`, initial value `0.1.0`, read trimmed.
- Embed directive is exactly `//go:embed all:frameworks all:skills version.txt` — the `all:` prefix is required so skill `references/` (and any `_`/dot files) are not silently dropped.
- Materialized cache lives at `<homonto.toml dir>/.homonto/catalog/skills/<name>/`. `.homonto/` is already gitignored (`internal/scaffold/scaffold.go` writes `/.homonto/`), which covers the catalog cache — no scaffold change is required, only verification.
- Directories are created 0755, files written 0644.
- The four first-release frameworks are `onto`, `comet`, `superpowers`, `openspec`. `comet` depends on `["superpowers", "openspec"]`; the other three have no dependencies.
- TDD is mandatory: for every logic-bearing task write the failing test first, watch it fail, then implement. Commit after each task (one commit per task). End every commit message with:
  `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`
- Full regression at the end: `go test ./... -count=1`, `go vet ./...`, `go build ./...`.

---

### Task 1: Catalog content + embed package

Maps tasks.md 1.1, 1.2, 1.3, 1.4. Content-only task (no unit test); its deliverable is a compiling embed package plus a metadata/content consistency check. Everything logic-bearing comes in later tasks.

**Files:**
- Create: `catalog/version.txt`
- Create: `catalog/frameworks/onto/framework.toml`
- Create: `catalog/frameworks/comet/framework.toml`
- Create: `catalog/frameworks/superpowers/framework.toml`
- Create: `catalog/frameworks/openspec/framework.toml`
- Create: `catalog/skills/<name>/…` (copied from `homonto/skills/<name>/`)
- Create: `catalog/embed.go`

**Interfaces:**
- Produces: `package catalog` (root) exporting `var FS embed.FS`. Consumed by `internal/catalog` in Task 2.

**Skill→framework mapping** (all 39 dirs under `homonto/skills/` map to exactly one framework):
- `onto` (8): onto, onto-build, onto-close, onto-design, onto-fix, onto-open, onto-tweak, onto-verify
- `comet` (8): comet, comet-open, comet-design, comet-build, comet-verify, comet-archive, comet-hotfix, comet-tweak
- `superpowers` (12): brainstorming, writing-plans, executing-plans, subagent-driven-development, using-git-worktrees, test-driven-development, systematic-debugging, verification-before-completion, finishing-a-development-branch, requesting-code-review, receiving-code-review, dispatching-parallel-agents
- `openspec` (11): openspec-apply-change, openspec-archive-change, openspec-bulk-archive-change, openspec-continue-change, openspec-explore, openspec-ff-change, openspec-new-change, openspec-onboard, openspec-propose, openspec-sync-specs, openspec-verify-change

- [x] **Step 1: Create the version file**

Create `catalog/version.txt` with exactly one line:

```
0.1.0
```

- [x] **Step 2: Create the four framework.toml files**

`catalog/frameworks/onto/framework.toml`:

```toml
name = "onto"
version = "0.1.0"
description = "Onto self-contained development workflow"

[dependencies]
frameworks = []

[skills]
onto = "skills/onto"
onto-build = "skills/onto-build"
onto-close = "skills/onto-close"
onto-design = "skills/onto-design"
onto-fix = "skills/onto-fix"
onto-open = "skills/onto-open"
onto-tweak = "skills/onto-tweak"
onto-verify = "skills/onto-verify"
```

`catalog/frameworks/superpowers/framework.toml`:

```toml
name = "superpowers"
version = "0.1.0"
description = "Superpowers development skills"

[dependencies]
frameworks = []

[skills]
brainstorming = "skills/brainstorming"
writing-plans = "skills/writing-plans"
executing-plans = "skills/executing-plans"
subagent-driven-development = "skills/subagent-driven-development"
using-git-worktrees = "skills/using-git-worktrees"
test-driven-development = "skills/test-driven-development"
systematic-debugging = "skills/systematic-debugging"
verification-before-completion = "skills/verification-before-completion"
finishing-a-development-branch = "skills/finishing-a-development-branch"
requesting-code-review = "skills/requesting-code-review"
receiving-code-review = "skills/receiving-code-review"
dispatching-parallel-agents = "skills/dispatching-parallel-agents"
```

`catalog/frameworks/openspec/framework.toml`:

```toml
name = "openspec"
version = "0.1.0"
description = "OpenSpec change-management skills"

[dependencies]
frameworks = []

[skills]
openspec-apply-change = "skills/openspec-apply-change"
openspec-archive-change = "skills/openspec-archive-change"
openspec-bulk-archive-change = "skills/openspec-bulk-archive-change"
openspec-continue-change = "skills/openspec-continue-change"
openspec-explore = "skills/openspec-explore"
openspec-ff-change = "skills/openspec-ff-change"
openspec-new-change = "skills/openspec-new-change"
openspec-onboard = "skills/openspec-onboard"
openspec-propose = "skills/openspec-propose"
openspec-sync-specs = "skills/openspec-sync-specs"
openspec-verify-change = "skills/openspec-verify-change"
```

`catalog/frameworks/comet/framework.toml`:

```toml
name = "comet"
version = "0.1.0"
description = "Comet five-phase OpenSpec workflow"

[dependencies]
frameworks = ["superpowers", "openspec"]

[skills]
comet = "skills/comet"
comet-open = "skills/comet-open"
comet-design = "skills/comet-design"
comet-build = "skills/comet-build"
comet-verify = "skills/comet-verify"
comet-archive = "skills/comet-archive"
comet-hotfix = "skills/comet-hotfix"
comet-tweak = "skills/comet-tweak"
```

- [x] **Step 3: Copy all skill content from `homonto/skills/` into `catalog/skills/`**

Run (copies every skill directory, including nested `references/`, `scripts/`, dotfiles):

```bash
mkdir -p catalog/skills
cp -R homonto/skills/. catalog/skills/
```

Verify the copy carried nested content (spot-check a skill that has `references/`):

```bash
ls catalog/skills/comet/ && ls catalog/skills/onto/
```

Expected: files/subdirectories present (e.g. `SKILL.md`, `references/`, `scripts/`).

- [x] **Step 4: Create the embed package**

Create `catalog/embed.go`:

```go
// Package catalog embeds the bundled framework metadata and skill content.
// It exposes only the embedded filesystem; all logic lives in
// github.com/noviopenworks/homonto/internal/catalog.
package catalog

import "embed"

//go:embed all:frameworks all:skills version.txt
var FS embed.FS
```

- [x] **Step 5: Verify the embed compiles and metadata matches content**

Run:

```bash
go build ./...
```

Expected: builds with no error (a missing embed target would fail the build).

Then verify every `framework.toml` skill path exists on disk:

```bash
for f in catalog/frameworks/*/framework.toml; do
  grep -oE '"skills/[^"]+"' "$f" | tr -d '"' | while read p; do
    [ -d "catalog/$p" ] || echo "MISSING: $f -> catalog/$p"
  done
done
echo "consistency check done"
```

Expected: prints only `consistency check done` (no `MISSING:` lines).

- [x] **Step 6: Commit**

```bash
git add catalog/
git commit -m "$(cat <<'EOF'
feat(catalog): embed first-release frameworks and skills

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 2: `internal/catalog` — Load, types, version, skill-path index

Maps tasks.md 2.1 and the parse/validation portion of 2.5.

**Files:**
- Create: `internal/catalog/catalog.go`
- Test: `internal/catalog/catalog_test.go`

**Interfaces:**
- Consumes: root `catalog.FS` (Task 1).
- Produces:
  - `type Framework struct { Name, Version, Description string; Dependencies []string; Skills map[string]string }`
  - `type Catalog struct { … }` (unexported fields)
  - `func New() (*Catalog, error)` — loads from the embedded `catalog.FS`
  - `func Load(fsys fs.FS) (*Catalog, error)`
  - `func (c *Catalog) Version() string`
  - `func (c *Catalog) Framework(name string) (Framework, bool)`
  - `func (c *Catalog) SkillPath(name string) (string, bool)` — skill name → catalog-relative path

- [x] **Step 1: Write the failing test**

Create `internal/catalog/catalog_test.go`:

```go
package catalog

import (
	"strings"
	"testing"
	"testing/fstest"
)

func fixtureFS() fstest.MapFS {
	return fstest.MapFS{
		"version.txt":                        {Data: []byte("0.1.0\n")},
		"frameworks/superpowers/framework.toml": {Data: []byte(`name = "superpowers"
version = "0.1.0"
description = "sp"
[skills]
brainstorming = "skills/brainstorming"
`)},
		"frameworks/comet/framework.toml": {Data: []byte(`name = "comet"
version = "0.1.0"
description = "cm"
[dependencies]
frameworks = ["superpowers"]
[skills]
comet = "skills/comet"
`)},
		"skills/brainstorming/SKILL.md": {Data: []byte("b")},
		"skills/comet/SKILL.md":         {Data: []byte("c")},
	}
}

func TestLoadIndexesFrameworksAndVersion(t *testing.T) {
	c, err := Load(fixtureFS())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if c.Version() != "0.1.0" {
		t.Fatalf("version = %q", c.Version())
	}
	cm, ok := c.Framework("comet")
	if !ok {
		t.Fatal("comet not indexed")
	}
	if len(cm.Dependencies) != 1 || cm.Dependencies[0] != "superpowers" {
		t.Fatalf("comet deps = %v", cm.Dependencies)
	}
	if p, ok := c.SkillPath("brainstorming"); !ok || p != "skills/brainstorming" {
		t.Fatalf("brainstorming path = %q ok=%v", p, ok)
	}
}

func TestLoadRejectsMissingSkillPath(t *testing.T) {
	m := fixtureFS()
	delete(m, "skills/comet/SKILL.md") // now skills/comet has no entries -> path absent
	_, err := Load(m)
	if err == nil || !strings.Contains(err.Error(), "skills/comet") {
		t.Fatalf("expected missing-skill-path error, got %v", err)
	}
}

func TestLoadRejectsNameDirMismatch(t *testing.T) {
	m := fixtureFS()
	m["frameworks/comet/framework.toml"] = &fstest.MapFile{Data: []byte(`name = "wrong"
version = "0.1.0"
[skills]
comet = "skills/comet"
`)}
	_, err := Load(m)
	if err == nil || !strings.Contains(err.Error(), "name") {
		t.Fatalf("expected name/dir mismatch error, got %v", err)
	}
}
```

- [x] **Step 2: Run test to verify it fails**

Run: `go test ./internal/catalog/ -run TestLoad -v`
Expected: FAIL / build error — `Load`, `Catalog`, `Framework`, `Version`, `SkillPath` undefined.

- [x] **Step 3: Write the implementation**

Create `internal/catalog/catalog.go`:

```go
// Package catalog loads and expands the embedded framework/skill catalog.
// It is config-agnostic: it MUST NOT import internal/config.
package catalog

import (
	"fmt"
	"io/fs"
	"path"
	"strings"

	embedded "github.com/noviopenworks/homonto/catalog"
	toml "github.com/pelletier/go-toml/v2"
)

// Framework is one catalog framework's parsed metadata.
type Framework struct {
	Name         string
	Version      string
	Description  string
	Dependencies []string          // framework names
	Skills       map[string]string // skill name -> catalog-relative path ("skills/<n>")
}

// Catalog is the loaded, indexed catalog.
type Catalog struct {
	fsys       fs.FS
	frameworks map[string]Framework
	skills     map[string]string // skill name -> catalog-relative path (global index)
	version    string
}

type frameworkTOML struct {
	Name         string `toml:"name"`
	Version      string `toml:"version"`
	Description  string `toml:"description"`
	Dependencies struct {
		Frameworks []string `toml:"frameworks"`
	} `toml:"dependencies"`
	Skills map[string]string `toml:"skills"`
}

// New loads the production catalog from the embedded filesystem.
func New() (*Catalog, error) { return Load(embedded.FS) }

// Load parses every frameworks/<name>/framework.toml in fsys, validates that
// each declared skill path exists and that a framework's name equals its
// directory, and reads version.txt (trimmed).
func Load(fsys fs.FS) (*Catalog, error) {
	c := &Catalog{
		fsys:       fsys,
		frameworks: map[string]Framework{},
		skills:     map[string]string{},
	}
	vb, err := fs.ReadFile(fsys, "version.txt")
	if err != nil {
		return nil, fmt.Errorf("catalog: read version.txt: %w", err)
	}
	c.version = strings.TrimSpace(string(vb))

	dirs, err := fs.ReadDir(fsys, "frameworks")
	if err != nil {
		return nil, fmt.Errorf("catalog: read frameworks: %w", err)
	}
	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}
		dir := d.Name()
		tp := path.Join("frameworks", dir, "framework.toml")
		b, err := fs.ReadFile(fsys, tp)
		if err != nil {
			return nil, fmt.Errorf("catalog: read %s: %w", tp, err)
		}
		var ft frameworkTOML
		if err := toml.Unmarshal(b, &ft); err != nil {
			return nil, fmt.Errorf("catalog: parse %s: %w", tp, err)
		}
		if ft.Name != dir {
			return nil, fmt.Errorf("catalog: framework %q declares name %q; name must equal directory", dir, ft.Name)
		}
		for skill, sp := range ft.Skills {
			if _, err := fs.Stat(fsys, sp); err != nil {
				return nil, fmt.Errorf("catalog: framework %q skill %q path %q missing from catalog", dir, skill, sp)
			}
			if prev, ok := c.skills[skill]; ok && prev != sp {
				return nil, fmt.Errorf("catalog: skill %q mapped to both %q and %q", skill, prev, sp)
			}
			c.skills[skill] = sp
		}
		c.frameworks[dir] = Framework{
			Name:         ft.Name,
			Version:      ft.Version,
			Description:  ft.Description,
			Dependencies: ft.Dependencies.Frameworks,
			Skills:       ft.Skills,
		}
	}
	return c, nil
}

// Version returns the catalog version string from version.txt.
func (c *Catalog) Version() string { return c.version }

// Framework returns the indexed framework and whether it exists.
func (c *Catalog) Framework(name string) (Framework, bool) {
	f, ok := c.frameworks[name]
	return f, ok
}

// SkillPath returns a skill's catalog-relative path ("skills/<n>") and whether
// it is known.
func (c *Catalog) SkillPath(name string) (string, bool) {
	p, ok := c.skills[name]
	return p, ok
}
```

- [x] **Step 4: Run test to verify it passes**

Run: `go test ./internal/catalog/ -run TestLoad -v`
Expected: PASS (all three tests). The real embedded catalog is also loadable — confirm with `go test ./internal/catalog/ -run TestLoad -count=1`.

- [x] **Step 5: Commit**

```bash
git add internal/catalog/catalog.go internal/catalog/catalog_test.go
git commit -m "feat(catalog): load and index frameworks from embedded FS

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 3: `internal/catalog` — transitive expansion + cycle detection

Maps tasks.md 2.2 and the expansion/cycle portion of 2.5.

**Files:**
- Create: `internal/catalog/expand.go`
- Test: `internal/catalog/expand_test.go`

**Interfaces:**
- Consumes: `Catalog.frameworks` (Task 2).
- Produces:
  - `type ExpandedSkill struct { Name, Framework string }`
  - `func (c *Catalog) Expand(frameworkNames []string) ([]ExpandedSkill, error)` — transitive, deduplicated, sorted by `Name`; returns an error naming a dependency cycle.

- [x] **Step 1: Write the failing test**

Create `internal/catalog/expand_test.go`:

```go
package catalog

import (
	"strings"
	"testing"
	"testing/fstest"
)

// graphFS builds a catalog whose skill content always exists, so Load passes and
// tests can focus on the dependency graph. deps maps framework -> dep names;
// skills maps framework -> its own skill names.
func graphFS(deps map[string][]string, skills map[string][]string) fstest.MapFS {
	m := fstest.MapFS{"version.txt": {Data: []byte("0.1.0")}}
	for fw, sk := range skills {
		var b strings.Builder
		b.WriteString("name = \"" + fw + "\"\nversion = \"0.1.0\"\n")
		if d := deps[fw]; len(d) > 0 {
			b.WriteString("[dependencies]\nframeworks = [")
			for i, dep := range d {
				if i > 0 {
					b.WriteString(", ")
				}
				b.WriteString("\"" + dep + "\"")
			}
			b.WriteString("]\n")
		}
		b.WriteString("[skills]\n")
		for _, s := range sk {
			b.WriteString(s + " = \"skills/" + s + "\"\n")
			m["skills/"+s+"/SKILL.md"] = &fstest.MapFile{Data: []byte("x")}
		}
		m["frameworks/"+fw+"/framework.toml"] = &fstest.MapFile{Data: []byte(b.String())}
	}
	return m
}

func TestExpandTransitiveAndDedup(t *testing.T) {
	// comet -> superpowers, openspec; superpowers and openspec share "shared".
	c, err := Load(graphFS(
		map[string][]string{"comet": {"superpowers", "openspec"}},
		map[string][]string{
			"comet":       {"comet-open"},
			"superpowers": {"brainstorming", "shared"},
			"openspec":    {"openspec-new", "shared"},
		},
	))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	got, err := c.Expand([]string{"comet"})
	if err != nil {
		t.Fatalf("expand: %v", err)
	}
	var names []string
	for _, e := range got {
		names = append(names, e.Name)
	}
	want := []string{"brainstorming", "comet-open", "openspec-new", "shared"}
	if strings.Join(names, ",") != strings.Join(want, ",") {
		t.Fatalf("expanded (sorted, deduped) = %v, want %v", names, want)
	}
}

func TestExpandDetectsCycle(t *testing.T) {
	c, err := Load(graphFS(
		map[string][]string{"a": {"b"}, "b": {"a"}},
		map[string][]string{"a": {"sa"}, "b": {"sb"}},
	))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	_, err = c.Expand([]string{"a"})
	if err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("expected cycle error, got %v", err)
	}
	if !strings.Contains(err.Error(), "a") || !strings.Contains(err.Error(), "b") {
		t.Fatalf("cycle error should name the chain, got %v", err)
	}
}
```

- [x] **Step 2: Run test to verify it fails**

Run: `go test ./internal/catalog/ -run TestExpand -v`
Expected: FAIL / build error — `Expand`, `ExpandedSkill` undefined.

- [x] **Step 3: Write the implementation**

Create `internal/catalog/expand.go`:

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

// Expand returns the transitive, deduplicated set of skills reachable from the
// given framework names, sorted by skill name, or an error naming a dependency
// cycle. A skill reachable via two frameworks collapses to one entry keyed by
// its first-seen origin.
func (c *Catalog) Expand(frameworkNames []string) ([]ExpandedSkill, error) {
	const (
		white = 0
		grey  = 1
		black = 2
	)
	color := map[string]int{}
	skills := map[string]ExpandedSkill{}
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
		for skill := range f.Skills {
			if _, seen := skills[skill]; !seen {
				skills[skill] = ExpandedSkill{Name: skill, Framework: name}
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

	out := make([]ExpandedSkill, 0, len(skills))
	for _, s := range skills {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}
```

- [x] **Step 4: Run test to verify it passes**

Run: `go test ./internal/catalog/ -run TestExpand -v`
Expected: PASS.

- [x] **Step 5: Commit**

```bash
git add internal/catalog/expand.go internal/catalog/expand_test.go
git commit -m "feat(catalog): transitive framework expansion with cycle detection

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 4: `internal/catalog` — materialization

Maps tasks.md 2.3 and the materialization portion of 2.5.

**Files:**
- Create: `internal/catalog/materialize.go`
- Test: `internal/catalog/materialize_test.go`

**Interfaces:**
- Consumes: `Catalog.SkillPath` + `Catalog.fsys` (Task 2).
- Produces: `func (c *Catalog) Materialize(dstRoot string, skillNames []string) error` — extracts each named skill's sub-FS into `dstRoot/<name>/`, removing any existing per-skill dir first so an upgrade cannot leave stale files.

- [x] **Step 1: Write the failing test**

Create `internal/catalog/materialize_test.go`:

```go
package catalog

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func matFS() fstest.MapFS {
	return fstest.MapFS{
		"version.txt": {Data: []byte("0.1.0")},
		"frameworks/sp/framework.toml": {Data: []byte(`name = "sp"
version = "0.1.0"
[skills]
brainstorming = "skills/brainstorming"
`)},
		"skills/brainstorming/SKILL.md":            {Data: []byte("top")},
		"skills/brainstorming/references/notes.md": {Data: []byte("nested")},
	}
}

func TestMaterializeWritesNestedContent(t *testing.T) {
	c, err := Load(matFS())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	dst := t.TempDir()
	if err := c.Materialize(dst, []string{"brainstorming"}); err != nil {
		t.Fatalf("materialize: %v", err)
	}
	if b, _ := os.ReadFile(filepath.Join(dst, "brainstorming", "SKILL.md")); string(b) != "top" {
		t.Fatalf("SKILL.md = %q", b)
	}
	if b, _ := os.ReadFile(filepath.Join(dst, "brainstorming", "references", "notes.md")); string(b) != "nested" {
		t.Fatalf("nested references/notes.md = %q", b)
	}
}

func TestMaterializeRemovesStaleOnUpgrade(t *testing.T) {
	c, err := Load(matFS())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	dst := t.TempDir()
	// Pre-seed a stale file that the new content does not include.
	os.MkdirAll(filepath.Join(dst, "brainstorming"), 0o755)
	os.WriteFile(filepath.Join(dst, "brainstorming", "STALE.md"), []byte("old"), 0o644)

	if err := c.Materialize(dst, []string{"brainstorming"}); err != nil {
		t.Fatalf("materialize: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dst, "brainstorming", "STALE.md")); !os.IsNotExist(err) {
		t.Fatal("stale file survived materialization")
	}
}

func TestMaterializeUnknownSkillErrors(t *testing.T) {
	c, _ := Load(matFS())
	if err := c.Materialize(t.TempDir(), []string{"nope"}); err == nil {
		t.Fatal("expected error for unknown skill")
	}
}
```

- [x] **Step 2: Run test to verify it fails**

Run: `go test ./internal/catalog/ -run TestMaterialize -v`
Expected: FAIL / build error — `Materialize` undefined.

- [x] **Step 3: Write the implementation**

Create `internal/catalog/materialize.go`:

```go
package catalog

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Materialize extracts each named builtin skill from the embedded FS into
// dstRoot/<name>/, removing any existing per-skill directory first so a stale
// file from a previous version cannot survive an upgrade. It is the caller's
// job (engine) to gate this on the catalog version.
func (c *Catalog) Materialize(dstRoot string, skillNames []string) error {
	for _, name := range skillNames {
		sp, ok := c.skills[name]
		if !ok {
			return fmt.Errorf("catalog: unknown skill %q", name)
		}
		sub, err := fs.Sub(c.fsys, sp)
		if err != nil {
			return fmt.Errorf("catalog: sub %q: %w", sp, err)
		}
		dstDir := filepath.Join(dstRoot, name)
		if err := os.RemoveAll(dstDir); err != nil {
			return err
		}
		err = fs.WalkDir(sub, ".", func(p string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			target := filepath.Join(dstDir, filepath.FromSlash(p))
			if d.IsDir() {
				return os.MkdirAll(target, 0o755)
			}
			data, err := fs.ReadFile(sub, p)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			return os.WriteFile(target, data, 0o644)
		})
		if err != nil {
			return err
		}
	}
	return nil
}
```

- [x] **Step 4: Run test to verify it passes**

Run: `go test ./internal/catalog/ -count=1 -v`
Expected: PASS (whole package, including Task 2 and 3 tests).

- [x] **Step 5: Commit**

```bash
git add internal/catalog/materialize.go internal/catalog/materialize_test.go
git commit -m "feat(catalog): materialize builtin skills from embedded FS

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 5: `internal/state` — catalog version slot

Maps tasks.md 4.2 (state portion; the engine gating that consumes it lands in Task 10).

**Files:**
- Modify: `internal/state/state.go`
- Test: `internal/state/state_test.go` (add cases)

**Interfaces:**
- Produces:
  - `State.CatalogVersion string` (JSON `catalogVersion,omitempty`)
  - `func (s *State) CatalogVersionRecorded() string`
  - `func (s *State) SetCatalogVersion(v string)`

- [x] **Step 1: Write the failing test**

Add to `internal/state/state_test.go`:

```go
func TestCatalogVersionRoundTrips(t *testing.T) {
	dir := t.TempDir()
	s, _ := Load(dir)
	if s.CatalogVersionRecorded() != "" {
		t.Fatal("fresh state should record no catalog version")
	}
	s.SetCatalogVersion("0.1.0")
	if err := s.Save(dir); err != nil {
		t.Fatal(err)
	}
	got, _ := Load(dir)
	if got.CatalogVersionRecorded() != "0.1.0" {
		t.Fatalf("reloaded catalog version = %q", got.CatalogVersionRecorded())
	}
}

func TestCatalogVersionOmittedWhenEmpty(t *testing.T) {
	dir := t.TempDir()
	s, _ := Load(dir)
	s.Set("claude", "mcp.a", "x", "h") // some content so the file is written
	if err := s.Save(dir); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(filepath.Join(dir, "state.json"))
	if strings.Contains(string(raw), "catalogVersion") {
		t.Fatalf("empty catalog version must be omitted, got %s", raw)
	}
}
```

- [x] **Step 2: Run test to verify it fails**

Run: `go test ./internal/state/ -run TestCatalogVersion -v`
Expected: FAIL / build error — `CatalogVersionRecorded`, `SetCatalogVersion` undefined.

- [x] **Step 3: Write the implementation**

In `internal/state/state.go`, change the `State` struct and add accessors. Replace:

```go
// State is the last-applied snapshot, keyed tool -> managed key -> Entry.
type State struct {
	Managed map[string]map[string]Entry `json:"managed"`
}
```

with:

```go
// State is the last-applied snapshot, keyed tool -> managed key -> Entry.
// CatalogVersion is the embedded-catalog version last successfully materialized;
// it is global (not per-tool) and omitted when empty so pre-catalog state.json
// files stay backward-compatible (absent = "force materialize").
type State struct {
	Managed        map[string]map[string]Entry `json:"managed"`
	CatalogVersion string                      `json:"catalogVersion,omitempty"`
}

// CatalogVersionRecorded returns the catalog version last materialized, or "".
func (s *State) CatalogVersionRecorded() string { return s.CatalogVersion }

// SetCatalogVersion records the catalog version after a successful materialize.
func (s *State) SetCatalogVersion(v string) { s.CatalogVersion = v }
```

- [x] **Step 4: Run test to verify it passes**

Run: `go test ./internal/state/ -count=1 -v`
Expected: PASS (new and existing tests).

- [x] **Step 5: Commit**

```bash
git add internal/state/state.go internal/state/state_test.go
git commit -m "feat(state): record catalog version for materialization gating

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 6: `internal/link` — managed roots as a variadic set

Maps tasks.md 5.3.

**Files:**
- Modify: `internal/link/linker.go`
- Test: `internal/link/linker_test.go` (add a multi-root case)

**Interfaces:**
- Produces (signatures change from single `contentRoot string` to variadic `roots ...string`; all existing single-root call sites stay source-compatible):
  - `func Link(src, dst string, roots ...string) (bool, error)`
  - `func Remove(dst string, roots ...string) error`
  - `func IsManaged(dst string, roots ...string) bool`
  - `func Plan(srcs map[string]string, roots ...string) ([]Op, error)`
  - `managed(target string, roots ...string) bool` — true if `target` is under ANY root.

- [x] **Step 1: Write the failing test**

Add to `internal/link/linker_test.go`:

```go
// TestLinkManagedAcrossMultipleRoots: a symlink pointing into the SECOND managed
// root (e.g. the materialized catalog) must be treated as ours and relinkable,
// while a symlink pointing outside every root is still a conflict.
func TestLinkManagedAcrossMultipleRoots(t *testing.T) {
	dir := t.TempDir()
	local := filepath.Join(dir, "content")
	catalog := filepath.Join(dir, "catalog")
	src := filepath.Join(catalog, "brainstorming")
	stale := filepath.Join(catalog, "brainstorming-old")
	os.MkdirAll(src, 0o755)
	os.MkdirAll(stale, 0o755)
	dst := filepath.Join(dir, "claude", "skills", "brainstorming")
	os.MkdirAll(filepath.Dir(dst), 0o755)
	if err := os.Symlink(stale, dst); err != nil {
		t.Fatal(err)
	}

	// A symlink under the catalog root is ours -> relinked in place.
	changed, err := Link(src, dst, local, catalog)
	if err != nil || !changed {
		t.Fatalf("relink under second root: changed=%v err=%v", changed, err)
	}
	if got, _ := os.Readlink(dst); got != src {
		t.Fatalf("symlink points to %q, want %q", got, src)
	}

	// IsManaged and Remove also honor the second root.
	if !IsManaged(dst, local, catalog) {
		t.Fatal("IsManaged should be true for a link under the catalog root")
	}
	if err := Remove(dst, local, catalog); err != nil {
		t.Fatalf("Remove under second root: %v", err)
	}

	// A link outside every root is still a conflict.
	foreign := filepath.Join(dir, "elsewhere")
	os.MkdirAll(foreign, 0o755)
	os.Symlink(foreign, dst)
	if _, err := Link(src, dst, local, catalog); err == nil || !strings.Contains(err.Error(), "conflict") {
		t.Fatalf("foreign link must still be a conflict, got %v", err)
	}
}
```

- [x] **Step 2: Run test to verify it fails**

Run: `go test ./internal/link/ -run TestLinkManagedAcrossMultipleRoots -v`
Expected: FAIL to compile — `Link`/`IsManaged`/`Remove` take one root today (they still compile with one arg, but `Link(src, dst, local, catalog)` is too many args → build error).

- [x] **Step 3: Write the implementation**

In `internal/link/linker.go`:

Replace `managed`:

```go
// managed reports whether target points inside ANY of the content roots homonto
// owns. A symlink pointing into one of them is ours (relinkable/prunable); a
// symlink pointing outside every root is user-owned and must never be touched.
func managed(target string, roots ...string) bool {
	for _, root := range roots {
		if strings.HasPrefix(target, root+string(os.PathSeparator)) {
			return true
		}
	}
	return false
}
```

Change `Link`'s signature and its two `managed`/error sites:

```go
func Link(src, dst string, roots ...string) (bool, error) {
```
- `if !managed(cur, contentRoot) {` → `if !managed(cur, roots...) {`
- error message: `..., outside managed content %s; not changing", dst, cur, contentRoot)` → `..., outside managed content %s; not changing", dst, cur, strings.Join(roots, ", "))`

Change `Remove`:

```go
func Remove(dst string, roots ...string) error {
```
- `if !managed(target, contentRoot) {` → `if !managed(target, roots...) {`
- error message `contentRoot` → `strings.Join(roots, ", ")`

Change `IsManaged`:

```go
func IsManaged(dst string, roots ...string) bool {
```
- final `return managed(target, contentRoot)` → `return managed(target, roots...)`

Change `Plan`:

```go
func Plan(srcs map[string]string, roots ...string) ([]Op, error) {
```
- `if !managed(cur, contentRoot) {` → `if !managed(cur, roots...) {`
- error message `contentRoot` → `strings.Join(roots, ", ")`

- [x] **Step 4: Run test to verify it passes**

Run: `go test ./internal/link/ -count=1 -v`
Expected: PASS (new multi-root test and all existing single-root tests — variadic keeps `Plan(m, content)` etc. valid).

- [x] **Step 5: Commit**

```bash
git add internal/link/linker.go internal/link/linker_test.go
git commit -m "feat(link): treat multiple managed roots as ours

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 7: `internal/config` — `ExpandedSkillEntriesForTool`

Maps tasks.md 3.1, 3.2, 3.3, 3.4.

**Files:**
- Modify: `internal/config/config.go`
- Test: `internal/config/config_test.go` (add cases)

**Interfaces:**
- Consumes: `internal/catalog` `New`, `Expand`, `ExpandedSkill` (Tasks 2–3).
- Produces: `func (c *Config) ExpandedSkillEntriesForTool(tool string) ([]NamedResource, error)` — explicit `[skills.X]` entries plus, for each `[frameworks.<fw>] source = "builtin:<fw>"` targeting `tool`, its transitively expanded skills as `NamedResource{Name: skill, Resource:{Source:"builtin:"+skill, Scope: fwScope, Targets: fwTargets}}`. Collision (explicit-vs-framework, or framework-vs-framework with a conflicting declaration) and dependency cycles are returned as errors.

Note (design §4): `config.Load` stays a pure parse+validate; collision/cycle errors surface from `ExpandedSkillEntriesForTool`, which `plan`/`apply` and the adapters call — so a bad framework graph is reported cleanly at plan/apply time.

- [ ] **Step 1: Write the failing test**

Add to `internal/config/config_test.go`. These drive the REAL embedded catalog (comet → superpowers + openspec), so they assert against known bundled skills:

```go
func loadTOML(t *testing.T, body string) *Config {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "homonto.toml")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := Load(p)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	return c
}

func TestExpandedSkillsIncludeFrameworkAndDeps(t *testing.T) {
	c := loadTOML(t, `
[frameworks.comet]
source = "builtin:comet"
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
`)
	got, err := c.ExpandedSkillEntriesForTool("claude")
	if err != nil {
		t.Fatalf("expand: %v", err)
	}
	byName := map[string]NamedResource{}
	for _, e := range got {
		byName[e.Name] = e
	}
	// A comet skill, a superpowers dep skill, and an openspec dep skill.
	for _, want := range []string{"comet-open", "brainstorming", "openspec-explore"} {
		e, ok := byName[want]
		if !ok {
			t.Fatalf("expanded set missing %q; got %v", want, keysOf(byName))
		}
		if e.Resource.Source != "builtin:"+want {
			t.Fatalf("%q source = %q", want, e.Resource.Source)
		}
		// Inherits the framework declaration's scope and targets (Spec Patch #1).
		if e.Resource.Scope != "user" || len(e.Resource.Targets) != 1 || e.Resource.Targets[0] != "claude" {
			t.Fatalf("%q did not inherit scope/targets: %+v", want, e.Resource)
		}
	}
}

func keysOf(m map[string]NamedResource) []string {
	var ks []string
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}

func TestExpandedSkillsTargetFiltering(t *testing.T) {
	c := loadTOML(t, `
[frameworks.comet]
source = "builtin:comet"
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
`)
	got, err := c.ExpandedSkillEntriesForTool("opencode")
	if err != nil {
		t.Fatalf("expand: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("comet targets claude only; opencode should get no skills, got %v", got)
	}
}

func TestExpandedSkillsCollisionWithExplicit(t *testing.T) {
	c := loadTOML(t, `
[frameworks.comet]
source = "builtin:comet"
scope = "user"
targets = ["claude"]

[skills.comet-open]
source = "builtin:comet-open"
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
`)
	_, err := c.ExpandedSkillEntriesForTool("claude")
	if err == nil || !strings.Contains(err.Error(), "comet-open") {
		t.Fatalf("expected collision error naming comet-open, got %v", err)
	}
}
```

(`os`, `path/filepath`, `strings` are already imported in config_test.go.)

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/config/ -run TestExpandedSkills -v`
Expected: FAIL / build error — `ExpandedSkillEntriesForTool` undefined.

- [ ] **Step 3: Write the implementation**

In `internal/config/config.go`, add imports `"slices"`, `"sync"`, and `cat "github.com/noviopenworks/homonto/internal/catalog"`. Then add:

```go
var (
	catalogOnce sync.Once
	catalogInst *cat.Catalog
	catalogErr  error
)

// loadedCatalog lazily builds the singleton embedded catalog (cheap to index).
func loadedCatalog() (*cat.Catalog, error) {
	catalogOnce.Do(func() { catalogInst, catalogErr = cat.New() })
	return catalogInst, catalogErr
}

func sameResource(a, b Resource) bool {
	return a.Source == b.Source && a.Scope == b.Scope && slices.Equal(a.Targets, b.Targets)
}

// ExpandedSkillEntriesForTool returns the effective skills for a tool: explicit
// [skills.X] entries plus, for each [frameworks.<fw>] source="builtin:<fw>"
// targeting the tool, its transitively expanded skills. Each expanded skill
// inherits the framework declaration's scope and targets. A framework skill
// whose name collides with an explicit [skills.X] entry, or with another
// framework's skill under a conflicting declaration, is an error, as is a
// dependency cycle (surfaced from catalog.Expand).
func (c *Config) ExpandedSkillEntriesForTool(tool string) ([]NamedResource, error) {
	byName := map[string]NamedResource{}
	explicitNames := map[string]bool{}
	for _, e := range c.SkillEntriesForTool(tool) {
		byName[e.Name] = e
		explicitNames[e.Name] = true
	}

	// Deterministic framework iteration order for stable error messages.
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
		expanded, err := cl.Expand([]string{builtin})
		if err != nil {
			return nil, fmt.Errorf("config: framework %q: %w", fwName, err)
		}
		for _, es := range expanded {
			if explicitNames[es.Name] {
				return nil, fmt.Errorf("config: skill %q is declared both explicitly in [skills] and by framework %q", es.Name, fwName)
			}
			nr := NamedResource{
				Name: es.Name,
				Resource: Resource{
					Source:  "builtin:" + es.Name,
					Scope:   fwRes.Scope,
					Targets: fwRes.Targets,
				},
			}
			if prev, ok := byName[es.Name]; ok {
				if !sameResource(prev.Resource, nr.Resource) {
					return nil, fmt.Errorf("config: skill %q expanded by multiple frameworks with conflicting scope/targets (framework %q)", es.Name, fwName)
				}
				continue
			}
			byName[es.Name] = nr
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

Run: `go test ./internal/config/ -count=1 -v`
Expected: PASS (new expansion tests and all existing config tests).

- [ ] **Step 5: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go
git commit -m "feat(config): expand builtin frameworks into effective skills

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 8: `claude` adapter — builtin source resolution + catalog root

Maps tasks.md 5.1 and the claude portion of 5.5.

**Files:**
- Modify: `internal/adapter/claude/claude.go`
- Test: `internal/adapter/claude/builtin_test.go` (new)

**Interfaces:**
- Consumes: `config.ExpandedSkillEntriesForTool` (Task 7); variadic `link.*` (Task 6).
- Produces:
  - `Adapter.catalogRoot string` field
  - `func (a *Adapter) WithCatalogRoot(catalogRoot string) *Adapter`
  - `func (a *Adapter) skillSource(entry config.NamedResource) string` — `builtin:<n>` → `filepath.Join(a.catalogRoot, <n>)`, else the existing local path.

- [ ] **Step 1: Write the failing test**

Create `internal/adapter/claude/builtin_test.go`:

```go
package claude

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/noviopenworks/homonto/internal/config"
	"github.com/noviopenworks/homonto/internal/secret"
	"github.com/noviopenworks/homonto/internal/state"
)

// builtinCfg declares one builtin skill directly (explicit [skills] passes
// through ExpandedSkillEntriesForTool unchanged; no framework expansion needed).
func builtinCfg() *config.Config {
	return &config.Config{
		Skills: map[string]config.Resource{
			"brainstorming": {Source: "builtin:brainstorming", Scope: "user", Targets: []string{"claude"}},
		},
	}
}

func resolver() *secret.Resolver {
	return &secret.Resolver{Getenv: os.Getenv, Pass: func(string) (string, error) { return "", nil }}
}

func TestBuiltinSkillLinksToCatalogRoot(t *testing.T) {
	home := t.TempDir()
	catalogRoot := t.TempDir()
	// Simulate materialization: the skill dir exists under the catalog root.
	os.MkdirAll(filepath.Join(catalogRoot, "brainstorming"), 0o755)

	a := New(home, t.TempDir()).WithCatalogRoot(catalogRoot)
	st, _ := state.Load(t.TempDir())
	cs, err := a.Plan(builtinCfg(), st)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if err := a.Apply(cs, resolver(), st); err != nil {
		t.Fatalf("apply: %v", err)
	}

	dst := filepath.Join(home, ".claude", "skills", "brainstorming")
	target, err := os.Readlink(dst)
	if err != nil {
		t.Fatalf("skill link missing: %v", err)
	}
	if want := filepath.Join(catalogRoot, "brainstorming"); target != want {
		t.Fatalf("link target = %q, want %q", target, want)
	}
	if _, ok := st.Get("claude", "skill.brainstorming"); !ok {
		t.Fatal("skill.brainstorming not recorded in state")
	}

	// Re-plan is a noop for the link.
	cs2, _ := a.Plan(builtinCfg(), st)
	for _, c := range cs2.Changes {
		if c.Key == "skill.brainstorming" && c.Action != "noop" {
			t.Fatalf("re-plan not idempotent: %+v", c)
		}
	}
}

func TestBuiltinSkillPrunedWhenDeDeclared(t *testing.T) {
	home := t.TempDir()
	catalogRoot := t.TempDir()
	os.MkdirAll(filepath.Join(catalogRoot, "brainstorming"), 0o755)
	a := New(home, t.TempDir()).WithCatalogRoot(catalogRoot)
	st, _ := state.Load(t.TempDir())

	cs, _ := a.Plan(builtinCfg(), st)
	if err := a.Apply(cs, resolver(), st); err != nil {
		t.Fatalf("apply: %v", err)
	}
	dst := filepath.Join(home, ".claude", "skills", "brainstorming")

	// Skill removed from config -> delete plan -> link pruned (managed under catalogRoot).
	empty := &config.Config{}
	cs2, err := a.Plan(empty, st)
	if err != nil {
		t.Fatalf("plan empty: %v", err)
	}
	if err := a.Apply(cs2, resolver(), st); err != nil {
		t.Fatalf("apply empty: %v", err)
	}
	if _, err := os.Lstat(dst); !os.IsNotExist(err) {
		t.Fatal("de-declared builtin skill link not pruned")
	}
}

func TestBuiltinSkillConflictNotClobbered(t *testing.T) {
	home := t.TempDir()
	catalogRoot := t.TempDir()
	os.MkdirAll(filepath.Join(catalogRoot, "brainstorming"), 0o755)
	a := New(home, t.TempDir()).WithCatalogRoot(catalogRoot)
	st, _ := state.Load(t.TempDir())

	dst := filepath.Join(home, ".claude", "skills", "brainstorming")
	os.MkdirAll(filepath.Dir(dst), 0o755)
	os.WriteFile(dst, []byte("user file"), 0o644) // a real file, not our link

	_, err := a.Plan(builtinCfg(), st)
	if err == nil {
		t.Fatal("expected conflict for real file at builtin skill link dst")
	}
	if b, _ := os.ReadFile(dst); string(b) != "user file" {
		t.Fatal("conflict clobbered the user file")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/adapter/claude/ -run TestBuiltin -v`
Expected: FAIL / build error — `WithCatalogRoot` undefined, and links target `<content>/skills/brainstorming` not the catalog root.

- [ ] **Step 3: Write the implementation**

In `internal/adapter/claude/claude.go`:

Add the field to the struct (after `content`):

```go
type Adapter struct {
	home        string
	content     string
	catalogRoot string // materialized builtin catalog root (.homonto/catalog/skills)
	projectRoot string // directory of homonto.toml; used for project-scope resources
	skills      []config.NamedResource
}
```

Add the builder + resolver near `WithProjectRoot`:

```go
// WithCatalogRoot sets the materialized builtin-catalog root that builtin:<name>
// skills link from. Mirrors WithProjectRoot.
func (a *Adapter) WithCatalogRoot(catalogRoot string) *Adapter {
	a.catalogRoot = catalogRoot
	return a
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
```

In `links()`, replace the body's source join:

```go
func (a *Adapter) links() map[string]string {
	out := map[string]string{}
	for _, entry := range a.skills {
		out[filepath.Join(a.skillsDir(entry.Resource.Scope), entry.Name)] = a.skillSource(entry)
	}
	return out
}
```

In `Plan`, change the skills source line and switch to expanded entries with error propagation. Replace:

```go
	a.skills = c.SkillEntriesForTool("claude")
```

with:

```go
	skills, err := c.ExpandedSkillEntriesForTool("claude")
	if err != nil {
		return adapter.ChangeSet{}, err
	}
	a.skills = skills
```

(There is already a `cur, err :=` a few lines down; because `err` is now declared above, change that line from `cur, err := a.current()` to `cur, err = a.current()`.)

In `Plan`, the adopt loop source line — replace:

```go
		src := filepath.Join(a.content, "skills", localSourceName(entry.Resource.Source, name))
```

with:

```go
		src := a.skillSource(entry)
```

In `Plan`, the relocate check — replace:

```go
		if op.Cur == "" && inactive != "" && link.IsManaged(filepath.Join(inactive, name), a.content) {
```

with:

```go
		if op.Cur == "" && inactive != "" && link.IsManaged(filepath.Join(inactive, name), a.content, a.catalogRoot) {
```

In `Plan`, the link.Plan call — replace `link.Plan(a.links(), a.content)` with `link.Plan(a.links(), a.content, a.catalogRoot)`.

In `Apply`, the delete-skill `link.Remove` — replace `err = link.Remove(dst, a.content)` with `err = link.Remove(dst, a.content, a.catalogRoot)`.

In `Apply`, the fail-fast `link.Plan(links, a.content)` — replace with `link.Plan(links, a.content, a.catalogRoot)`.

In `Apply`, the inactive-prune block — replace:

```go
		if link.IsManaged(old, a.content) {
			if err := link.Remove(old, a.content); err != nil {
```

with:

```go
		if link.IsManaged(old, a.content, a.catalogRoot) {
			if err := link.Remove(old, a.content, a.catalogRoot); err != nil {
```

In `Apply`, the final relink loop — replace `if _, err := link.Link(src, dst, a.content); err != nil {` with `if _, err := link.Link(src, dst, a.content, a.catalogRoot); err != nil {`.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/adapter/claude/ -count=1 -v`
Expected: PASS (new builtin tests and all existing claude tests — existing tests use `local:` sources, unaffected by `skillSource`).

- [ ] **Step 5: Commit**

```bash
git add internal/adapter/claude/claude.go internal/adapter/claude/builtin_test.go
git commit -m "feat(claude): resolve builtin skills from materialized catalog

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 9: `opencode` adapter — builtin source resolution + catalog root

Maps tasks.md 5.2 and the opencode portion of 5.5. Mirrors Task 8 exactly.

**Files:**
- Modify: `internal/adapter/opencode/opencode.go`
- Test: `internal/adapter/opencode/builtin_test.go` (new)

**Interfaces:** identical shape to Task 8 (`Adapter.catalogRoot`, `WithCatalogRoot`, `skillSource`).

- [ ] **Step 1: Write the failing test**

Create `internal/adapter/opencode/builtin_test.go` — same as Task 8's file but with `package opencode` and the opencode user skills dir (`<home>/.config/opencode/skills/brainstorming`):

```go
package opencode

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/noviopenworks/homonto/internal/config"
	"github.com/noviopenworks/homonto/internal/secret"
	"github.com/noviopenworks/homonto/internal/state"
)

func builtinCfg() *config.Config {
	return &config.Config{
		Skills: map[string]config.Resource{
			"brainstorming": {Source: "builtin:brainstorming", Scope: "user", Targets: []string{"opencode"}},
		},
	}
}

func resolver() *secret.Resolver {
	return &secret.Resolver{Getenv: os.Getenv, Pass: func(string) (string, error) { return "", nil }}
}

func TestBuiltinSkillLinksToCatalogRoot(t *testing.T) {
	home := t.TempDir()
	catalogRoot := t.TempDir()
	os.MkdirAll(filepath.Join(catalogRoot, "brainstorming"), 0o755)
	a := New(home, t.TempDir()).WithCatalogRoot(catalogRoot)
	st, _ := state.Load(t.TempDir())

	cs, err := a.Plan(builtinCfg(), st)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if err := a.Apply(cs, resolver(), st); err != nil {
		t.Fatalf("apply: %v", err)
	}
	dst := filepath.Join(home, ".config", "opencode", "skills", "brainstorming")
	target, err := os.Readlink(dst)
	if err != nil {
		t.Fatalf("skill link missing: %v", err)
	}
	if want := filepath.Join(catalogRoot, "brainstorming"); target != want {
		t.Fatalf("link target = %q, want %q", target, want)
	}
	if _, ok := st.Get("opencode", "skill.brainstorming"); !ok {
		t.Fatal("skill.brainstorming not recorded")
	}
	cs2, _ := a.Plan(builtinCfg(), st)
	for _, c := range cs2.Changes {
		if c.Key == "skill.brainstorming" && c.Action != "noop" {
			t.Fatalf("re-plan not idempotent: %+v", c)
		}
	}
}

func TestBuiltinSkillPrunedWhenDeDeclared(t *testing.T) {
	home := t.TempDir()
	catalogRoot := t.TempDir()
	os.MkdirAll(filepath.Join(catalogRoot, "brainstorming"), 0o755)
	a := New(home, t.TempDir()).WithCatalogRoot(catalogRoot)
	st, _ := state.Load(t.TempDir())
	cs, _ := a.Plan(builtinCfg(), st)
	if err := a.Apply(cs, resolver(), st); err != nil {
		t.Fatalf("apply: %v", err)
	}
	dst := filepath.Join(home, ".config", "opencode", "skills", "brainstorming")
	cs2, err := a.Plan(&config.Config{}, st)
	if err != nil {
		t.Fatalf("plan empty: %v", err)
	}
	if err := a.Apply(cs2, resolver(), st); err != nil {
		t.Fatalf("apply empty: %v", err)
	}
	if _, err := os.Lstat(dst); !os.IsNotExist(err) {
		t.Fatal("de-declared builtin skill link not pruned")
	}
}

func TestBuiltinSkillConflictNotClobbered(t *testing.T) {
	home := t.TempDir()
	catalogRoot := t.TempDir()
	os.MkdirAll(filepath.Join(catalogRoot, "brainstorming"), 0o755)
	a := New(home, t.TempDir()).WithCatalogRoot(catalogRoot)
	st, _ := state.Load(t.TempDir())
	dst := filepath.Join(home, ".config", "opencode", "skills", "brainstorming")
	os.MkdirAll(filepath.Dir(dst), 0o755)
	os.WriteFile(dst, []byte("user file"), 0o644)
	if _, err := a.Plan(builtinCfg(), st); err == nil {
		t.Fatal("expected conflict for real file at builtin skill link dst")
	}
	if b, _ := os.ReadFile(dst); string(b) != "user file" {
		t.Fatal("conflict clobbered the user file")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/adapter/opencode/ -run TestBuiltin -v`
Expected: FAIL / build error — `WithCatalogRoot` undefined.

- [ ] **Step 3: Write the implementation**

Apply the same edits as Task 8, to `internal/adapter/opencode/opencode.go`:

Add `catalogRoot string` to the `Adapter` struct (after `content`). Add `WithCatalogRoot` and `skillSource` (identical bodies to Task 8). In `Plan`, replace:

```go
	a.skills = c.SkillEntriesForTool("opencode")
```

with:

```go
	skills, err := c.ExpandedSkillEntriesForTool("opencode")
	if err != nil {
		return adapter.ChangeSet{}, err
	}
	a.skills = skills
```

(There is already a `doc, err := readStandardized(...)` right below; change it to `doc, err = readStandardized(...)`.)

In `links()`, use `a.skillSource(entry)` for the value. In the adopt loop, replace `src := filepath.Join(a.content, "skills", localSourceName(entry.Resource.Source, name))` with `src := a.skillSource(entry)`. In the relocate check, `link.IsManaged(filepath.Join(inactive, name), a.content)` → add `, a.catalogRoot`. Every `link.Plan(a.links(), a.content)`, `link.Plan(links, a.content)`, `link.Remove(dst, a.content)`, `link.IsManaged(old, a.content)`, `link.Remove(old, a.content)`, and `link.Link(src, dst, a.content)` → append `, a.catalogRoot`.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/adapter/opencode/ -count=1 -v`
Expected: PASS (new builtin tests and all existing opencode tests).

- [ ] **Step 5: Commit**

```bash
git add internal/adapter/opencode/opencode.go internal/adapter/opencode/builtin_test.go
git commit -m "feat(opencode): resolve builtin skills from materialized catalog

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 10: `internal/engine` — materialization orchestration

Maps tasks.md 4.1, 4.2, 4.3.

**Files:**
- Modify: `internal/engine/engine.go`
- Test: `internal/engine/materialize_test.go` (new)

**Interfaces:**
- Consumes: `catalog.New`/`Materialize`/`Version` (Tasks 2,4); `state.CatalogVersionRecorded`/`SetCatalogVersion` (Task 5); adapter `WithCatalogRoot` (Tasks 8,9); `config.ExpandedSkillEntriesForTool` (Task 7).
- Produces:
  - `Engine.CatalogDir string` (= `<stateDir>/catalog/skills`)
  - Both adapters wired with `.WithCatalogRoot(catalogDir)` in `Build`.
  - `func (e *Engine) materializeCatalog() error`, called at the top of `Apply` before the adapter loop.

- [ ] **Step 1: Write the failing test**

Create `internal/engine/materialize_test.go`. It drives the REAL embedded catalog through a `[frameworks.comet]` config and asserts the materialized cache appears, is version-gated, and re-materializes when the recorded version is stale:

```go
package engine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/noviopenworks/homonto/internal/secret"
)

const cometTOML = `
[frameworks.comet]
source = "builtin:comet"
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

func buildEngine(t *testing.T, home, repo string) *Engine {
	t.Helper()
	e, err := Build(filepath.Join(repo, "homonto.toml"), home, filepath.Join(repo, "content"))
	if err != nil {
		t.Fatal(err)
	}
	e.Resolver = &secret.Resolver{Getenv: os.Getenv, Pass: func(string) (string, error) { return "", nil }}
	return e
}

func TestApplyMaterializesBuiltinSkills(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	os.WriteFile(filepath.Join(repo, "homonto.toml"), []byte(cometTOML), 0o644)

	e := buildEngine(t, home, repo)
	sets, _ := e.Plan()
	if err := e.Apply(sets); err != nil {
		t.Fatalf("apply: %v", err)
	}

	// A known comet skill materialized under .homonto/catalog/skills/.
	got := filepath.Join(repo, ".homonto", "catalog", "skills", "comet-open", "SKILL.md")
	if _, err := os.Stat(got); err != nil {
		t.Fatalf("comet-open not materialized: %v", err)
	}
	// State recorded the catalog version.
	if e.State.CatalogVersionRecorded() == "" {
		t.Fatal("catalog version not recorded after materialization")
	}
	// A dependency skill (superpowers) also materialized.
	if _, err := os.Stat(filepath.Join(repo, ".homonto", "catalog", "skills", "brainstorming")); err != nil {
		t.Fatalf("dependency skill brainstorming not materialized: %v", err)
	}
}

func TestApplyRematerializesWhenVersionStale(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	os.WriteFile(filepath.Join(repo, "homonto.toml"), []byte(cometTOML), 0o644)

	e := buildEngine(t, home, repo)
	sets, _ := e.Plan()
	if err := e.Apply(sets); err != nil {
		t.Fatalf("apply: %v", err)
	}
	skillFile := filepath.Join(repo, ".homonto", "catalog", "skills", "comet-open", "SKILL.md")

	// Simulate a partial/stale cache: corrupt content + wipe the recorded version.
	os.WriteFile(skillFile, []byte("STALE"), 0o644)
	e.State.SetCatalogVersion("")
	if err := e.State.Save(e.StateDir); err != nil {
		t.Fatal(err)
	}

	e2 := buildEngine(t, home, repo)
	sets2, _ := e2.Plan()
	if err := e2.Apply(sets2); err != nil {
		t.Fatalf("re-apply: %v", err)
	}
	if b, _ := os.ReadFile(skillFile); string(b) == "STALE" {
		t.Fatal("stale content not refreshed when recorded version was empty")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/engine/ -run TestApplyMaterializes -v`
Expected: FAIL — `.homonto/catalog/skills/comet-open` is not created (no materialization step yet); `CatalogVersionRecorded` empty.

- [ ] **Step 3: Write the implementation**

In `internal/engine/engine.go`:

Add imports `"os"`, `"sort"`, `"strings"`, and `"github.com/noviopenworks/homonto/internal/catalog"`.

Add the field to `Engine` (after `ContentDir`):

```go
	ContentDir  string
	CatalogDir  string // materialized builtin catalog root (<stateDir>/catalog/skills)
```

In `Build`, after `stateDir := filepath.Join(...)`, compute the catalog dir and wire the adapters. Replace the `Adapters:` slice and add `CatalogDir` to the returned struct:

```go
	stateDir := filepath.Join(filepath.Dir(configPath), ".homonto")
	catalogDir := filepath.Join(stateDir, "catalog", "skills")
	st, err := state.Load(stateDir)
	if err != nil {
		return nil, err
	}
	return &Engine{
		Cfg: cfg,
		Adapters: []adapter.Adapter{
			claude.New(home, contentDir).WithProjectRoot(projectRoot).WithCatalogRoot(catalogDir),
			opencode.New(home, contentDir).WithProjectRoot(projectRoot).WithCatalogRoot(catalogDir),
		},
		State:       st,
		StateDir:    stateDir,
		ContentDir:  contentDir,
		CatalogDir:  catalogDir,
		Home:        home,
		ProjectRoot: projectRoot,
		Resolver:    secret.NewResolver(),
	}, nil
```

In `Apply`, add the materialization call after the secret pre-resolve loop and before the `byName` adapter loop:

```go
	// Materialize builtin skills before any adapter links them, so no symlink is
	// created ahead of its target.
	if err := e.materializeCatalog(); err != nil {
		return err
	}
```

Add the method at the end of the file:

```go
// materializeCatalog extracts the builtin skills the config declares into
// CatalogDir, version-gated: it is a no-op when the recorded catalog version
// matches the embedded one and every skill dir already exists. The version is
// recorded (and state saved) only after a full successful materialization, so an
// interrupted extraction re-materializes on the next apply.
func (e *Engine) materializeCatalog() error {
	names := map[string]bool{}
	for _, tool := range []string{"claude", "opencode"} {
		entries, err := e.Cfg.ExpandedSkillEntriesForTool(tool)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if strings.HasPrefix(entry.Resource.Source, "builtin:") {
				names[strings.TrimPrefix(entry.Resource.Source, "builtin:")] = true
			}
		}
	}
	if len(names) == 0 {
		return nil
	}
	cl, err := catalog.New()
	if err != nil {
		return err
	}
	skillNames := make([]string, 0, len(names))
	for n := range names {
		skillNames = append(skillNames, n)
	}
	sort.Strings(skillNames)

	if e.State.CatalogVersionRecorded() == cl.Version() && allSkillDirsExist(e.CatalogDir, skillNames) {
		return nil
	}
	if err := cl.Materialize(e.CatalogDir, skillNames); err != nil {
		return err
	}
	e.State.SetCatalogVersion(cl.Version())
	// Save immediately so a later adapter failure still records the completed
	// materialization.
	return e.State.Save(e.StateDir)
}

func allSkillDirsExist(root string, names []string) bool {
	for _, n := range names {
		fi, err := os.Stat(filepath.Join(root, n))
		if err != nil || !fi.IsDir() {
			return false
		}
	}
	return true
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/engine/ -count=1 -v`
Expected: PASS (new materialization tests and all existing engine tests).

- [ ] **Step 5: Commit**

```bash
git add internal/engine/engine.go internal/engine/materialize_test.go
git commit -m "feat(engine): materialize builtin catalog before linking

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 11: `doctor` — check builtin skills at the materialized path

Maps tasks.md 5.4.

**Files:**
- Modify: `internal/engine/status.go`
- Test: `internal/engine/status_test.go` (add a case)

**Interfaces:**
- Consumes: `Engine.CatalogDir` (Task 10); `config.ExpandedSkillEntriesForTool` (Task 7).

- [ ] **Step 1: Write the failing test**

Add to `internal/engine/status_test.go` (reuse `cometTOML` / `buildEngine` from Task 10's file — same package):

```go
func TestDoctorReportsBuiltinSkillLinked(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	os.WriteFile(filepath.Join(repo, "homonto.toml"), []byte(cometTOML), 0o644)

	e := buildEngine(t, home, repo)
	sets, _ := e.Plan()
	if err := e.Apply(sets); err != nil {
		t.Fatalf("apply: %v", err)
	}

	out := e.Doctor()
	joined := strings.Join(out, "\n")
	if !strings.Contains(joined, `skill "comet-open" linked (claude)`) {
		t.Fatalf("doctor did not report the builtin skill as linked:\n%s", joined)
	}
}
```

(Ensure `strings`, `os`, `path/filepath` are imported in status_test.go.)

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/engine/ -run TestDoctorReportsBuiltin -v`
Expected: FAIL — Doctor iterates `SkillEntriesForTool` (explicit only, so a framework config surfaces no skills) and resolves content from `ContentDir`, not the materialized catalog.

- [ ] **Step 3: Write the implementation**

In `internal/engine/status.go`, `Doctor()`, replace each of the two skill loops (claude and opencode) so they iterate the expanded entries and resolve builtin sources from `CatalogDir`. Replace the claude loop:

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
```

with a call that handles both tools via a helper (add the helper below `Doctor`):

```go
	claudeSkills, cerr := e.Cfg.ExpandedSkillEntriesForTool("claude")
	if cerr != nil {
		out = append(out, fmt.Sprintf("warn: cannot expand claude skills: %v", cerr))
	} else {
		out = append(out, e.doctorSkills("claude", claudeSkills)...)
	}
	opencodeSkills, oerr := e.Cfg.ExpandedSkillEntriesForTool("opencode")
	if oerr != nil {
		out = append(out, fmt.Sprintf("warn: cannot expand opencode skills: %v", oerr))
	} else {
		out = append(out, e.doctorSkills("opencode", opencodeSkills)...)
	}
```

Delete the old opencode skill loop (the second `for _, entry := range e.Cfg.SkillEntriesForTool("opencode")` block) — it is replaced by the `opencodeSkills` call above. Add the helper:

```go
// doctorSkills reports, per skill, whether its content is present at the right
// source (builtin: from the materialized catalog, local: from the content dir)
// and whether it is linked into the tool's skills directory.
func (e *Engine) doctorSkills(tool string, entries []config.NamedResource) []string {
	var out []string
	for _, entry := range entries {
		name := entry.Name
		var p string
		if strings.HasPrefix(entry.Resource.Source, "builtin:") {
			p = filepath.Join(e.CatalogDir, strings.TrimPrefix(entry.Resource.Source, "builtin:"))
		} else {
			sourceName := name
			if strings.HasPrefix(entry.Resource.Source, "local:") {
				sourceName = strings.TrimPrefix(entry.Resource.Source, "local:")
			}
			p = filepath.Join(e.ContentDir, "skills", sourceName)
		}
		if _, err := os.Stat(p); err != nil {
			out = append(out, fmt.Sprintf("warn: skill %q missing from %s (run apply)", name, p))
			continue
		}
		dst := filepath.Join(skillpath.Dir(tool, entry.Resource.Scope, e.Home, e.ProjectRoot), name)
		if target, err := os.Readlink(dst); err == nil && target == p {
			out = append(out, fmt.Sprintf("ok: skill %q linked (%s)", name, tool))
		} else {
			out = append(out, fmt.Sprintf("warn: skill %q content present, not linked for %s (run apply)", name, tool))
		}
	}
	return out
}
```

Add `"github.com/noviopenworks/homonto/internal/config"` to the imports of `status.go`.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/engine/ -count=1 -v`
Expected: PASS. Also confirm existing `status_test.go` cases (which use `local:` skills) still pass — the helper preserves the same `ok:`/`warn:` message shapes.

- [ ] **Step 5: Commit**

```bash
git add internal/engine/status.go internal/engine/status_test.go
git commit -m "feat(doctor): check builtin skills at the materialized catalog path

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 12: Dogfood config — switch to `[frameworks.comet]`

Maps tasks.md 6.1, 6.2, 6.3, 6.4. This materializes REAL skill content from `catalog/skills/` (copied in Task 1.2) and links it on the developer's own machine.

**Files:**
- Modify: `homonto.toml`

**Note on scope (6.2):** Every skill currently declared in `homonto.toml` (comet-*, the openspec-* subset, and the superpowers skills) is a member of the `comet`/`superpowers`/`openspec` frameworks, so `[frameworks.comet]` covers them all. There are no local-only skills to keep as explicit `[skills.X]` entries here; do not re-add any.

- [ ] **Step 1: Replace the explicit skill list with the framework declaration**

Replace the entire block of `[skills.*]` tables at the top of `homonto.toml` (from `[skills.comet]` through `[skills.dispatching-parallel-agents]`) with a single framework declaration, leaving the `[models.*]` tables untouched:

```toml
[frameworks.comet]
source = "builtin:comet"
scope = "project"
```

The file after editing is: this `[frameworks.comet]` block followed by the existing `[models.claude.*]` and `[models.opencode.*]` tables.

- [ ] **Step 2: Build the binary and run plan (dry check before apply)**

```bash
go build -o /tmp/homonto . && /tmp/homonto plan
```

Expected: plan lists `create` for each comet + superpowers + openspec skill link across claude and opencode (project scope). No conflict errors.

- [ ] **Step 3: Apply and verify materialization + links (6.3)**

```bash
/tmp/homonto apply --yes
ls .homonto/catalog/skills/ | head
ls -l .claude/skills/ | grep -- '->' | head
```

Expected: `.homonto/catalog/skills/` contains the expanded skills (comet-*, brainstorming, openspec-*, …); `.claude/skills/` and `.opencode/skills/` contain symlinks pointing into `.homonto/catalog/skills/`.

- [ ] **Step 4: Verify no drift (6.4)**

```bash
/tmp/homonto status
/tmp/homonto doctor
```

Expected: `status` reports no drift and 0 pending after a second apply; `doctor` reports every expanded skill as `linked`. Run `/tmp/homonto apply --yes` a second time and confirm it is a no-op (idempotent, catalog materialization skipped by version gate).

- [ ] **Step 5: Confirm the catalog cache is gitignored**

```bash
git status --porcelain .homonto/
```

Expected: no output (`.homonto/` is covered by the existing `/.homonto/` gitignore rule; the materialized catalog is not tracked).

- [ ] **Step 6: Commit**

```bash
git add homonto.toml
git commit -m "chore: dogfood builtin comet framework via catalog

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 13: Regression + docs

Maps tasks.md 7.1, 7.2, 7.3, 7.4.

**Files:**
- Modify: `docs/NEXT_AGENT.md` (append verification evidence)

- [ ] **Step 1: Full regression (7.1)**

```bash
go test ./... -count=1
go vet ./...
go build ./...
```

Expected: all tests PASS, vet clean, build succeeds.

- [ ] **Step 2: Stale-doc grep (7.2)**

Confirm no doc claims builtin skill projection is still unimplemented:

```bash
grep -rniE 'builtin.*(not|un)implemented|skills?.*not.*(installed|projected)|catalog.*(todo|planned)' docs/ README.md openspec/ 2>/dev/null || echo "no stale claims"
```

Expected: `no stale claims` (or only matches that are clearly about future non-skill resources — commands/subagents/frameworks projection, which remains future work per the config-model spec). If a stale skill-specific claim appears, update that doc line to state builtin skill projection is implemented.

- [ ] **Step 3: Record verification evidence (7.3)**

Append a dated section to `docs/NEXT_AGENT.md` summarizing: the catalog package + embed, `internal/catalog` Load/Expand/Materialize, config expansion, engine materialization, adapter builtin resolution, doctor check, and the dogfood run — with the exact commands from Task 12 Steps 3–4 and their observed results (materialized skill count, `status` clean, `doctor` all-linked, second apply idempotent).

- [ ] **Step 4: Final commit (7.4)**

```bash
git add docs/NEXT_AGENT.md
git commit -m "docs: record catalog-foundation-skills verification evidence

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Self-Review

**Spec coverage** (delta specs → tasks):

- builtin-catalog "Catalog loading from embedded filesystem" → Tasks 1, 2.
- builtin-catalog "Skill content materialization" (+ Spec Patch #2 partial-materialization) → Tasks 4, 10.
- builtin-catalog "Catalog version tracking" / upgrade re-materialization → Tasks 5, 10 (`TestApplyRematerializesWhenVersionStale`).
- builtin-catalog "Materialized catalog is generated state" (gitignore) → Task 12 Step 5 (verified; existing scaffold rule already covers it — noted in Global Constraints).
- config-model "Bundled catalog embedded in binary" → Task 1.
- config-model "Builtin skill source resolution" → Tasks 8, 9.
- config-model "Materialized catalog is generated state" / "Local provider content root" (MODIFIED, builtin resolves from materialized catalog) → Tasks 8, 9.
- framework-expansion "Framework metadata format" → Tasks 1, 2.
- framework-expansion "Framework expansion from builtin source" (+ Spec Patch #1 scope/targets inheritance) → Task 7 (`TestExpandedSkillsIncludeFrameworkAndDeps` asserts scope/targets).
- framework-expansion "Framework atomicity" / name collision → Task 7 (`TestExpandedSkillsCollisionWithExplicit`).
- framework-expansion "Dependency cycle detection" → Tasks 3 (`TestExpandDetectsCycle`), surfaced through 7.
- framework-expansion "First-release catalog frameworks" (comet deps) → Task 1 (framework.toml), asserted in Task 2/3 fixtures and Task 10 real-catalog test.
- tool-adapters "Owned content linked by symlink with conflict detection" (MODIFIED for builtin) → Tasks 6, 8, 9 (create/idempotent/prune/conflict tests for both adapters).

**Placeholder scan:** every code step contains complete code; every edit names the exact old→new text; every verification step has an exact command and expected output. No TBD/TODO left.

**Type consistency:** `Catalog`, `Framework`, `ExpandedSkill`, `Load`/`New`/`Version`/`Framework`/`SkillPath`/`Expand`/`Materialize` are defined in Tasks 2–4 and consumed with the same signatures in Tasks 7 and 10. `ExpandedSkillEntriesForTool` (Task 7) is consumed by Tasks 8, 9, 10, 11. `WithCatalogRoot`/`skillSource`/`catalogRoot` are defined per adapter in Tasks 8/9 and wired in Task 10. `CatalogVersionRecorded`/`SetCatalogVersion` (Task 5) are consumed in Task 10. Variadic `link.*` (Task 6) is consumed in Tasks 8, 9. `Engine.CatalogDir` (Task 10) is consumed in Task 11. All names match across tasks.

## Execution Handoff

**Plan complete and saved to `docs/superpowers/plans/2026-07-10-catalog-foundation-skills.md`. Two execution options:**

**1. Subagent-Driven (recommended)** — dispatch a fresh subagent per task, review between tasks, fast iteration.

**2. Inline Execution** — execute tasks in this session using executing-plans, batch execution with checkpoints.

**Which approach?**
