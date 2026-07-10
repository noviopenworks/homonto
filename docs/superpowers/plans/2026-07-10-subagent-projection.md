---
change: subagent-projection
design-doc: docs/superpowers/specs/2026-07-10-subagent-projection-design.md
base-ref: a53950f972987344a78663294a7f12315f540be5
---

# Subagent Projection Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Materialize and symlink declared subagents from the bundled catalog into Claude Code (`agents/`) and OpenCode (`agent/`), scope-aware with adopt/prune/relocate and doctor verification, mirroring the archived `command-projection` change almost exactly.

**Architecture:** Add a parallel `subagent.*` path alongside the existing `skill.*` and `command.*` paths — a new `internal/subagentpath` package, `Framework.Subagents` + `ExpandSubagents` + `MaterializeSubagents` in `internal/catalog`, `ExpandedSubagentEntriesForTool` in `internal/config`, engine materialization + `WithSubagentCatalogRoot` wiring, and `subagent.*` link plan/apply/adopt/prune/relocate blocks in both adapters. Do **not** generalize skills/commands/subagents into one loop (Design D1). Subagents are single verbatim Markdown files — no frontmatter rewrite, no model injection (Design D2).

**Tech Stack:** Go 1.23; `github.com/pelletier/go-toml/v2` (only third-party dep; no YAML library available — frontmatter checks use string parsing); `go:embed` catalog; `internal/link` (multi-root symlink manager) and `internal/state` (catalog version + per-key hashes), both reused unchanged.

## Global Constraints

Copied verbatim from the Design Doc and delta specs — every task's requirements implicitly include these:

- **Mirror the command pipeline; do not generalize** (D1). Replicate the fresh, tested `command.*` code as a sibling `subagent.*` path; never refactor the working skills/commands paths.
- **Verbatim single-file materialization** (D2). `catalog/subagents/<name>.md` → `.homonto/catalog/subagents/<name>.md`, byte-for-byte. No `RemoveAll` (single-file overwrite replaces prior content). Never rewrite frontmatter; never inject a model route. Version-gated on the **same** catalog version already tracked in state; version recorded only after skills + commands + subagents all materialize.
- **Directory naming** (D3). Claude uses plural `agents/` at both scopes; OpenCode uses singular `agent/` at both scopes. New sibling package `internal/subagentpath` (do not extend `commandpath`). Reuse `skillpath.Other` for scope flipping.
- **Framework `[subagents]` expansion** (D4). Transitive across dependencies, deduped by name; a subagent name colliding with an explicit `[subagents.X]` entry is a config error. Inherit the framework declaration's `scope` and `targets`.
- **Three real bundled subagents** (D5): `code-reviewer` and `codebase-explorer` as loose builtin subagents (both tools) plus one comet-framework subagent (both tools). Each is a single verbatim file whose frontmatter is the **minimal shared subset**: `name`, `description`, `mode: subagent`. **Omit `model` and `tools`** — they are the only hard-conflicting fields.
- **State keys** (D6). `subagent.<name>` handled exactly like `command.<name>`: symlink hash `Hash(dst + " -> " + src)`, adopt of a correct-but-unrecorded link, orphan prune only when the link points into a managed root (`homonto/subagents/` or `.homonto/catalog/subagents/`), scope switch rendered as a relocation. Never clobber a real file or foreign link — report a conflict.
- **Empty-root guard.** `managedRoots()` must include the subagent catalog root only when non-empty: `link.managed("")` prefix-matches every absolute path.
- **Additive only.** No change to existing skill/command/MCP/settings behavior or the existing model-route validation.
- **Verification gates (Design "Testing Strategy" step 9):** `go test ./... -count=1`, `go test -race ./...`, `go vet ./...`, `go build ./...`, `gofmt -l .`.
- Module path is `github.com/noviopenworks/homonto`. The catalog embed package is `github.com/noviopenworks/homonto/catalog` (import alias `embedded`); the logic package is `internal/catalog`.

---

## File Structure

**New files:**
- `internal/subagentpath/subagentpath.go` — `Dir(tool, scope, home, projectRoot) string` for the agent directory. Single responsibility: the `(tool, scope) → agent dir` map.
- `internal/subagentpath/subagentpath_test.go` — all tool/scope combos; singular/plural assertion.
- `internal/subagentpath/frontmatter_test.go` — the empirical both-tools shared-frontmatter contract check over fixtures.
- `internal/subagentpath/testdata/claude/agents/sample.md`, `internal/subagentpath/testdata/opencode/agent/sample.md` — real-layout fixtures (minimal shared frontmatter) for the contract check.
- `catalog/subagents/code-reviewer.md`, `catalog/subagents/codebase-explorer.md`, `catalog/subagents/comet-navigator.md` — the three bundled subagents.

**Modified files:**
- `catalog/embed.go` — add `all:subagents` to the `//go:embed` directive.
- `catalog/frameworks/comet/framework.toml` — add `[subagents]` entry for `comet-navigator`.
- `internal/catalog/catalog.go` — `Framework.Subagents`, `frameworkTOML.Subagents`, `Catalog.subagents` index, parse+validate loop, `SubagentPath`.
- `internal/catalog/expand.go` — `ExpandedSubagent` type + `ExpandSubagents` method.
- `internal/catalog/materialize.go` — `MaterializeSubagents` (verbatim single-file).
- `internal/config/config.go` — `SubagentEntriesForTool`, `ExpandedSubagentEntriesForTool`.
- `internal/engine/engine.go` — `SubagentCatalogRoot` field, `Build` wiring, `SubagentDir`, `materializeCatalog` subagent collection + gate + `allSubagentFilesExist`.
- `internal/adapter/claude/claude.go` and `internal/adapter/opencode/opencode.go` — `subagentCatalogRoot`, `subagents` field, `WithSubagentCatalogRoot`, `managedRoots` extension, `subagentsDir`/`inactiveSubagentsDir`/`subagentSource`/`subagentLinks`, Plan block, Apply block, ObserveHashes branch, declared map.
- `internal/adapter/claude/util.go` and `internal/adapter/opencode/util.go` — `managedPrefix` gains `"subagent."`.
- `internal/engine/status.go` — `doctorSubagents` + `Doctor()` wiring.
- `homonto.toml` — declare `code-reviewer` and `codebase-explorer` (dogfood).
- `README.md`, `docs/guides/using-homonto.md`, `docs/roadmap.md` — mark subagent projection shipped.
- Test files alongside each modified package.

Existing model-route validation already counts subagent-targeted tools (`EnabledModelTools` iterates `c.Subagents`); no code change there, only a confirming test (Task 4c).

---

## Task 1: `subagentpath` package, real-layout fixtures, and frontmatter contract

**Files:**
- Create: `internal/subagentpath/subagentpath.go`
- Create: `internal/subagentpath/subagentpath_test.go`
- Create: `internal/subagentpath/frontmatter_test.go`
- Create: `internal/subagentpath/testdata/claude/agents/sample.md`
- Create: `internal/subagentpath/testdata/opencode/agent/sample.md`

**Interfaces:**
- Produces: `subagentpath.Dir(tool, scope, home, projectRoot string) string` — mirrors `commandpath.Dir`; used by both adapters and doctor.
- Produces the validated **minimal shared frontmatter shape** (`name`, `description`, `mode: subagent`; no `model`, no `tools`) that Task 2 authors the real subagents against.

- [x] **Step 1: Write the real-layout fixtures (both tools, both directory names)**

Create `internal/subagentpath/testdata/claude/agents/sample.md`:

```markdown
---
name: sample
description: Fixture subagent used to lock the shared minimal frontmatter contract.
mode: subagent
---

Sample body.
```

Create `internal/subagentpath/testdata/opencode/agent/sample.md` with **identical bytes** (the point is one verbatim file is valid at both tools' real paths — Claude `agents/`, OpenCode `agent/`):

```markdown
---
name: sample
description: Fixture subagent used to lock the shared minimal frontmatter contract.
mode: subagent
---

Sample body.
```

- [x] **Step 2: Write the failing frontmatter contract test**

Create `internal/subagentpath/frontmatter_test.go`. No YAML library is available, so parse the frontmatter block by string. This is the empirical both-tools load check (Design Risks): it asserts each real-layout fixture carries exactly the shared keys and omits the two hard-conflicting keys.

```go
package subagentpath

import (
	"os"
	"strings"
	"testing"
)

// frontmatter returns the text between the first two "---" fence lines.
func frontmatter(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	s := string(b)
	if !strings.HasPrefix(s, "---\n") {
		t.Fatalf("%s: no leading frontmatter fence", path)
	}
	rest := s[len("---\n"):]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		t.Fatalf("%s: unterminated frontmatter", path)
	}
	return rest[:end]
}

func TestSharedFrontmatterContract(t *testing.T) {
	for _, path := range []string{
		"testdata/claude/agents/sample.md",
		"testdata/opencode/agent/sample.md",
	} {
		fm := frontmatter(t, path)
		for _, want := range []string{"name:", "description:", "mode: subagent"} {
			if !strings.Contains(fm, want) {
				t.Errorf("%s frontmatter missing %q", path, want)
			}
		}
		for _, forbidden := range []string{"model:", "tools:"} {
			for _, line := range strings.Split(fm, "\n") {
				if strings.HasPrefix(strings.TrimSpace(line), forbidden) {
					t.Errorf("%s frontmatter must omit %q (hard-conflicting field)", path, forbidden)
				}
			}
		}
	}
}
```

- [x] **Step 3: Run the test to verify it fails**

Run: `go test ./internal/subagentpath/ -run TestSharedFrontmatterContract -v`
Expected: FAIL — package has no non-test `.go` file yet, so it will not compile ("no Go files" / build error). (Once Step 4 lands, this test passes.)

- [x] **Step 4: Write `subagentpath.Dir`**

Create `internal/subagentpath/subagentpath.go`:

```go
// Package subagentpath is the single source of truth for where each tool's
// owned subagents are linked, as a function of the install scope. It parallels
// commandpath/skillpath; a future change may unify them into a
// resourcepath.Dir(kind, …). Scope flipping (for inactive-scope pruning) reuses
// skillpath.Other, so this package exposes only Dir.
package subagentpath

import "path/filepath"

// Dir returns the directory a tool's owned subagents are linked into.
//
//	claude   + user     -> <home>/.claude/agents
//	claude   + project  -> <projectRoot>/.claude/agents
//	opencode + user     -> <home>/.config/opencode/agent
//	opencode + project  -> <projectRoot>/.opencode/agent
//
// Claude Code uses the PLURAL "agents" directory at both scopes; OpenCode uses
// the SINGULAR "agent" directory at both scopes (consistent with its singular
// "command"). Any scope other than "project" is treated as "user". An unknown
// tool returns "".
func Dir(tool, scope, home, projectRoot string) string {
	project := scope == "project"
	switch tool {
	case "claude":
		if project {
			return filepath.Join(projectRoot, ".claude", "agents")
		}
		return filepath.Join(home, ".claude", "agents")
	case "opencode":
		if project {
			return filepath.Join(projectRoot, ".opencode", "agent")
		}
		return filepath.Join(home, ".config", "opencode", "agent")
	}
	return ""
}
```

- [x] **Step 5: Write the failing `Dir` unit test**

Create `internal/subagentpath/subagentpath_test.go`:

```go
package subagentpath

import "testing"

func TestDir(t *testing.T) {
	const home = "/home/u"
	const proj = "/repo"
	cases := []struct {
		tool, scope, want string
	}{
		{"claude", "user", "/home/u/.claude/agents"},
		{"claude", "project", "/repo/.claude/agents"},
		{"opencode", "user", "/home/u/.config/opencode/agent"},
		{"opencode", "project", "/repo/.opencode/agent"},
		{"claude", "", "/home/u/.claude/agents"},       // unknown scope -> user
		{"opencode", "bogus", "/home/u/.config/opencode/agent"},
		{"unknown", "user", ""},
	}
	for _, c := range cases {
		if got := Dir(c.tool, c.scope, home, proj); got != c.want {
			t.Errorf("Dir(%q,%q) = %q, want %q", c.tool, c.scope, got, c.want)
		}
	}
	// Singular/plural split assertion (the whole reason for this package).
	if Dir("claude", "user", home, proj) == Dir("opencode", "user", home, proj) {
		t.Fatal("claude and opencode agent dirs must differ (agents/ vs agent/)")
	}
}
```

- [x] **Step 6: Run the tests to verify they pass**

Run: `go test ./internal/subagentpath/ -v`
Expected: PASS (`TestDir`, `TestSharedFrontmatterContract`).

- [x] **Step 7: Commit**

```bash
git add internal/subagentpath
git commit -m "feat(subagentpath): agent dir mapping + shared-frontmatter contract fixtures"
```

---

## Task 2: Bundled subagent content and embed

**Files:**
- Create: `catalog/subagents/code-reviewer.md`
- Create: `catalog/subagents/codebase-explorer.md`
- Create: `catalog/subagents/comet-navigator.md`
- Modify: `catalog/embed.go`
- Create: `internal/catalog/subagents_embed_test.go`

**Interfaces:**
- Produces: three embedded files under `subagents/` in `embedded.FS`, each valid against the Task 1 shared-frontmatter contract. `comet-navigator` is the one wired into the comet framework in Task 3.

- [x] **Step 1: Author `catalog/subagents/code-reviewer.md`**

```markdown
---
name: code-reviewer
description: Use to review a diff or set of changes for correctness, security, and clarity before merging; reports findings ranked by severity.
mode: subagent
---

You are a focused code reviewer. Given a change (a diff, a set of files, or a
description of what was modified), review it for defects and report findings.

Priorities, in order:

1. Correctness — logic errors, off-by-one, nil/undefined access, wrong
   conditionals, broken error handling, race conditions, resource leaks.
2. Security — injection, unsafe deserialization, secret leakage, missing
   authorization, unvalidated input crossing a trust boundary.
3. Contract — API/type mismatches, violated invariants, misuse of a called
   function's documented behavior.
4. Clarity and maintainability — dead code, needless duplication, misleading
   names, missing or wrong tests for the changed behavior.

Rules:

- Read the surrounding code before judging a change; do not flag something that
  the existing context already handles.
- Report each finding with: file and line, severity (critical/major/minor), a
  one-sentence statement of the defect, and a concrete failure scenario
  (inputs/state → wrong result).
- Rank findings most-severe first. If you find nothing substantive, say so
  plainly rather than inventing nits.
- Do not rewrite the whole change; propose the smallest fix that addresses each
  finding.
```

- [x] **Step 2: Author `catalog/subagents/codebase-explorer.md`**

```markdown
---
name: codebase-explorer
description: Use to answer questions about how a codebase works or to locate where behavior lives, by reading across many files and returning conclusions rather than raw dumps.
mode: subagent
---

You are a read-only codebase explorer. Given a question about how something
works or where a behavior lives, investigate the repository and return a
grounded answer.

Method:

- Start broad, then narrow. Search by symbol, filename, and naming convention;
  follow imports and call sites to trace a flow end to end.
- Prefer the repository's own code-intelligence tooling when present; fall back
  to grep/find and direct reads otherwise.
- Read enough surrounding context to be correct — check multiple locations and
  alternative names before concluding something is absent.

Output:

- Answer the question directly first, then cite the exact files (and line
  ranges where load-bearing) that support the answer.
- Include a code snippet only when the exact text matters (a signature, a bug,
  a specific branch); do not recap code you merely read.
- If the answer is genuinely not in the codebase, say so and name where you
  looked. Never edit files — this agent only investigates and reports.
```

- [x] **Step 3: Author `catalog/subagents/comet-navigator.md`**

```markdown
---
name: comet-navigator
description: Use to orient within the Comet five-phase OpenSpec workflow — identify the active change's phase and the allowed next action, and point to the right phase skill.
mode: subagent
---

You are a navigator for the Comet five-phase OpenSpec workflow. Given the state
of a repository using Comet, determine where the work stands and what is allowed
next.

The five phases and their gate order: open → design → build → verify → archive.

Method:

- Look for an active change under `openspec/changes/<name>/` and read its
  `.comet.yaml` `phase` field to establish the current phase. Never guess the
  phase from conversation alone.
- Map the phase to its allowed operations (e.g. `build` allows writing source,
  tests, and executing the plan; `design` forbids writing implementation code).
- Point to the phase-appropriate skill (comet-open / comet-design /
  comet-build / comet-verify / comet-archive) and the next required script or
  confirmation gate.

Output:

- State the active change, its phase, and the single next allowed action.
- Flag any operation that would violate the current phase's rules.
- If no active change exists, say so and describe how a new change is started.
  This agent orients and reports; it does not perform phase transitions itself.
```

- [x] **Step 4: Extend the embed directive**

In `catalog/embed.go`, change the directive line:

```go
//go:embed all:frameworks all:skills all:commands all:subagents version.txt
```

- [x] **Step 5: Write the failing embed presence test**

Create `internal/catalog/subagents_embed_test.go`:

```go
package catalog

import (
	"io/fs"
	"testing"

	embedded "github.com/noviopenworks/homonto/catalog"
)

func TestSubagentsEmbedded(t *testing.T) {
	for _, name := range []string{"code-reviewer", "codebase-explorer", "comet-navigator"} {
		p := "subagents/" + name + ".md"
		if _, err := fs.Stat(embedded.FS, p); err != nil {
			t.Errorf("%s not embedded: %v", p, err)
		}
	}
}
```

- [x] **Step 6: Run the test to verify it passes (and the embed compiles)**

Run: `go build ./... && go test ./internal/catalog/ -run TestSubagentsEmbedded -v`
Expected: build succeeds; test PASS. (If Step 4 were omitted the `go:embed` would still compile but `fs.Stat` would fail — the test is the gate.)

- [x] **Step 7: Commit**

```bash
git add catalog/subagents catalog/embed.go internal/catalog/subagents_embed_test.go
git commit -m "feat(catalog): bundle code-reviewer, codebase-explorer, comet-navigator subagents"
```

---

## Task 3: Catalog subagent parse, index, expand, materialize + comet framework wiring

**Files:**
- Modify: `internal/catalog/catalog.go`
- Modify: `internal/catalog/expand.go`
- Modify: `internal/catalog/materialize.go`
- Modify: `catalog/frameworks/comet/framework.toml`
- Modify: `internal/catalog/catalog_test.go` (or a new `subagents_test.go`)
- Modify: `internal/catalog/expand_test.go`
- Modify: `internal/catalog/materialize_test.go`

**Interfaces:**
- Consumes: `Catalog.fsys`, `Catalog.frameworks`, `expandResources` (Task-independent — all already exist).
- Produces:
  - `Framework.Subagents map[string]string` (subagent name → `subagents/<n>.md`).
  - `(c *Catalog) SubagentPath(name string) (string, bool)`.
  - `type ExpandedSubagent struct { Name, Framework string }` and `(c *Catalog) ExpandSubagents(frameworkNames []string) ([]ExpandedSubagent, error)`.
  - `(c *Catalog) MaterializeSubagents(dstRoot string, names []string) error` — writes `<dstRoot>/<name>.md` byte-for-byte.

- [x] **Step 1: Write the failing parse + expand + materialize tests**

In `internal/catalog/materialize_test.go`, add (mirrors `TestMaterializeCommandsWritesFile`, adding a byte-for-byte assertion against the embedded source):

```go
func TestMaterializeSubagentsWritesFileVerbatim(t *testing.T) {
	c, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	dst := t.TempDir()
	if err := c.MaterializeSubagents(dst, []string{"code-reviewer"}); err != nil {
		t.Fatalf("materialize: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dst, "code-reviewer.md"))
	if err != nil {
		t.Fatalf("read materialized: %v", err)
	}
	sp, _ := c.SubagentPath("code-reviewer")
	want, err := fs.ReadFile(embedded.FS, sp)
	if err != nil {
		t.Fatalf("read source: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("materialized subagent is not byte-for-byte identical to catalog source")
	}
}

func TestMaterializeSubagentsUnknownErrors(t *testing.T) {
	c, _ := New()
	if err := c.MaterializeSubagents(t.TempDir(), []string{"nope"}); err == nil {
		t.Fatal("expected error for unknown subagent")
	}
}
```

Ensure `materialize_test.go` imports include `bytes`, `io/fs`, and the `embedded "github.com/noviopenworks/homonto/catalog"` alias (add any missing).

In `internal/catalog/expand_test.go`, add (mirrors `TestExpandCommandsTransitiveAndDedup` — use a local synthetic `Catalog` if that test does, otherwise expand the real comet framework):

```go
func TestExpandSubagentsIncludesFrameworkSubagent(t *testing.T) {
	c, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	got, err := c.ExpandSubagents([]string{"comet"})
	if err != nil {
		t.Fatalf("expand: %v", err)
	}
	found := false
	for _, e := range got {
		if e.Name == "comet-navigator" {
			found = true
			if e.Framework != "comet" {
				t.Errorf("comet-navigator framework = %q, want comet", e.Framework)
			}
		}
	}
	if !found {
		t.Fatal("comet-navigator not returned by ExpandSubagents([comet])")
	}
}
```

- [x] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/catalog/ -run 'TestMaterializeSubagents|TestExpandSubagents' -v`
Expected: FAIL to compile — `SubagentPath`, `MaterializeSubagents`, `ExpandSubagents` undefined, and the comet `[subagents]` entry not yet present.

- [x] **Step 3: Add the `Subagents` field, index, and parse+validate loop**

In `internal/catalog/catalog.go`:

Add to `Framework`:
```go
	Commands     map[string]string // command name -> catalog-relative path ("commands/<n>.md")
	Subagents    map[string]string // subagent name -> catalog-relative path ("subagents/<n>.md")
```

Add to `Catalog`:
```go
	commands   map[string]string // command name -> catalog-relative path (global index)
	subagents  map[string]string // subagent name -> catalog-relative path (global index)
```

Add to `frameworkTOML`:
```go
	Commands  map[string]string `toml:"commands"`
	Subagents map[string]string `toml:"subagents"`
```

In `Load`, initialize the index:
```go
		commands:   map[string]string{},
		subagents:  map[string]string{},
```

After the existing `for command, cp := range ft.Commands { … }` loop, add the parallel loop:
```go
		for subagent, sap := range ft.Subagents {
			if _, err := fs.Stat(fsys, sap); err != nil {
				return nil, fmt.Errorf("catalog: framework %q subagent %q path %q missing from catalog", dir, subagent, sap)
			}
			if prev, ok := c.subagents[subagent]; ok && prev != sap {
				return nil, fmt.Errorf("catalog: subagent %q mapped to both %q and %q", subagent, prev, sap)
			}
			c.subagents[subagent] = sap
		}
```

Add `Subagents: ft.Subagents,` to the `c.frameworks[dir] = Framework{…}` literal.

Add the lookup after `CommandPath`:
```go
// SubagentPath returns a subagent's catalog-relative path ("subagents/<n>.md")
// and whether it is known.
func (c *Catalog) SubagentPath(name string) (string, bool) {
	p, ok := c.subagents[name]
	return p, ok
}
```

- [x] **Step 4: Add `ExpandSubagents`**

In `internal/catalog/expand.go`, add the type next to `ExpandedCommand`:
```go
// ExpandedSubagent is one subagent reached by framework expansion, tagged with
// the framework it originated from.
type ExpandedSubagent struct {
	Name      string
	Framework string
}
```

Add the method next to `ExpandCommands`:
```go
// ExpandSubagents returns the transitive, deduplicated set of subagents
// reachable from the given framework names, sorted by subagent name, or an
// error naming a dependency cycle.
func (c *Catalog) ExpandSubagents(frameworkNames []string) ([]ExpandedSubagent, error) {
	res, err := c.expandResources(frameworkNames, func(f Framework) map[string]string { return f.Subagents })
	if err != nil {
		return nil, err
	}
	out := make([]ExpandedSubagent, len(res))
	for i, e := range res {
		out[i] = ExpandedSubagent{Name: e.Name, Framework: e.Framework}
	}
	return out, nil
}
```

- [x] **Step 5: Add `MaterializeSubagents`**

In `internal/catalog/materialize.go`, add after `MaterializeCommands` (identical single-file, verbatim shape — subagent index instead of command index):
```go
// MaterializeSubagents writes each named builtin subagent from the embedded FS
// to dstRoot/<name>.md (a single file), replacing any existing file
// byte-for-byte. Like MaterializeCommands, no RemoveAll is needed — a
// single-file overwrite fully replaces prior content on upgrade. Homonto never
// rewrites the subagent's frontmatter and never injects a model route, so the
// written file is identical to the embedded catalog source. It is the caller's
// job (engine) to gate this on the catalog version.
func (c *Catalog) MaterializeSubagents(dstRoot string, names []string) error {
	for _, name := range names {
		sp, ok := c.subagents[name]
		if !ok {
			return fmt.Errorf("catalog: unknown subagent %q", name)
		}
		data, err := fs.ReadFile(c.fsys, sp)
		if err != nil {
			return fmt.Errorf("catalog: read %q: %w", sp, err)
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

- [x] **Step 6: Wire the comet framework's `[subagents]` entry**

Append to `catalog/frameworks/comet/framework.toml`:
```toml

[subagents]
comet-navigator = "subagents/comet-navigator.md"
```

- [x] **Step 7: Run the tests to verify they pass**

Run: `go test ./internal/catalog/ -count=1 -v`
Expected: PASS, including the two new tests and every pre-existing catalog test (the parse loop must not break existing skills/commands parsing).

- [x] **Step 8: Commit**

```bash
git add internal/catalog catalog/frameworks/comet/framework.toml
git commit -m "feat(catalog): parse/index/expand/materialize subagents; wire comet-navigator into comet"
```

---

## Task 4: Config subagent expansion

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`

**Interfaces:**
- Consumes: `Config.Subagents` (already parsed), `entriesForTool`, `sameResource`, `loadedCatalog`, `Catalog.ExpandSubagents`.
- Produces:
  - `(c *Config) SubagentEntriesForTool(tool string) []NamedResource`.
  - `(c *Config) ExpandedSubagentEntriesForTool(tool string) ([]NamedResource, error)` — explicit `[subagents.X]` + framework-expanded subagents, scope/targets inheritance, explicit-vs-framework collision error, conflicting-framework error, cycle propagation.

- [ ] **Step 1: Write the failing config tests**

In `internal/config/config_test.go`, add (mirrors `TestExpandedCommandsExplicitAndTargetFilter` and the collision test for commands):

```go
func TestExpandedSubagentsExplicitAndTargetFilter(t *testing.T) {
	doc := `
[subagents.code-reviewer]
source = "builtin:code-reviewer"
scope = "project"
targets = ["claude"]

[models.claude.architectural]
model = "opus"
variant = "max"
[models.claude.coding]
model = "sonnet"
effort = "normal"
[models.claude.trivial]
model = "haiku"
effort = "fast"
`
	c := mustLoad(t, doc)
	claude, err := c.ExpandedSubagentEntriesForTool("claude")
	if err != nil {
		t.Fatalf("claude: %v", err)
	}
	if len(claude) != 1 || claude[0].Name != "code-reviewer" {
		t.Fatalf("claude subagents = %+v, want [code-reviewer]", claude)
	}
	oc, err := c.ExpandedSubagentEntriesForTool("opencode")
	if err != nil {
		t.Fatalf("opencode: %v", err)
	}
	if len(oc) != 0 {
		t.Fatalf("opencode subagents = %+v, want none (target filter)", oc)
	}
}

func TestExpandedSubagentsFrameworkInheritsScopeTargets(t *testing.T) {
	doc := `
[frameworks.comet]
source = "builtin:comet"
scope = "project"
targets = ["claude", "opencode"]

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
	c := mustLoad(t, doc)
	got, err := c.ExpandedSubagentEntriesForTool("claude")
	if err != nil {
		t.Fatalf("expand: %v", err)
	}
	var nav *NamedResource
	for i := range got {
		if got[i].Name == "comet-navigator" {
			nav = &got[i]
		}
	}
	if nav == nil {
		t.Fatal("comet-navigator not expanded for claude")
	}
	if nav.Resource.Scope != "project" || nav.Resource.Source != "builtin:comet-navigator" {
		t.Fatalf("comet-navigator inherited wrong scope/source: %+v", nav.Resource)
	}
}

func TestExpandedSubagentsExplicitVsFrameworkCollision(t *testing.T) {
	doc := `
[frameworks.comet]
source = "builtin:comet"
scope = "project"

[subagents.comet-navigator]
source = "builtin:comet-navigator"
scope = "user"

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
	c := mustLoad(t, doc)
	if _, err := c.ExpandedSubagentEntriesForTool("claude"); err == nil {
		t.Fatal("expected collision error: comet-navigator declared explicitly and by framework")
	}
}
```

Note: use whatever load helper the existing command tests use. If they call `mustLoad`/`loadDoc`, match it; the existing `TestExpandedCommandsExplicitAndTargetFilter` shows the exact helper name — reuse it verbatim rather than introducing a new one.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/config/ -run TestExpandedSubagents -v`
Expected: FAIL — `ExpandedSubagentEntriesForTool` undefined.

- [ ] **Step 3: Add `SubagentEntriesForTool` and `ExpandedSubagentEntriesForTool`**

In `internal/config/config.go`, add next to `CommandEntriesForTool`:
```go
func (c *Config) SubagentEntriesForTool(tool string) []NamedResource {
	return entriesForTool(c.Subagents, tool)
}
```

Add next to `ExpandedCommandEntriesForTool` (a verbatim copy with `Subagents`/`ExpandSubagents`/`subagent` wording):
```go
// ExpandedSubagentEntriesForTool returns the effective subagents for a tool:
// explicit [subagents.X] entries plus, for each [frameworks.<fw>]
// source="builtin:<fw>" targeting the tool, its transitively expanded
// subagents. Each expanded subagent inherits the framework declaration's scope
// and targets. A framework subagent whose name collides with an explicit
// [subagents.X] entry, or with another framework's subagent under a conflicting
// declaration, is an error, as is a dependency cycle (surfaced from
// catalog.ExpandSubagents). Collision is subagent-vs-subagent only.
func (c *Config) ExpandedSubagentEntriesForTool(tool string) ([]NamedResource, error) {
	byName := map[string]NamedResource{}
	explicitNames := map[string]bool{}
	for _, e := range c.SubagentEntriesForTool(tool) {
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
		expanded, err := cl.ExpandSubagents([]string{builtin})
		if err != nil {
			return nil, fmt.Errorf("config: framework %q: %w", fwName, err)
		}
		for _, es := range expanded {
			if explicitNames[es.Name] {
				return nil, fmt.Errorf("config: subagent %q is declared both explicitly in [subagents] and by framework %q", es.Name, fwName)
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
					return nil, fmt.Errorf("config: subagent %q expanded by multiple frameworks with conflicting scope/targets (framework %q)", es.Name, fwName)
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

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/config/ -run TestExpandedSubagents -v`
Expected: PASS.

- [ ] **Step 5: Add the model-validation-gap confirming test**

`EnabledModelTools` already iterates `c.Subagents`, so a subagent targeting a tool with no model routes must already fail at `Load`. Add a test that locks this behavior (mirrors `TestLoadRequiresAllModelLevelsForEnabledTools` but driven by `[subagents.X]`):

```go
func TestLoadRequiresModelsForSubagentTargetedTool(t *testing.T) {
	doc := `
[subagents.code-reviewer]
source = "builtin:code-reviewer"
scope = "project"
targets = ["opencode"]
`
	if err := loadDoc(t, doc); err == nil {
		t.Fatal("subagent enabling opencode without model routes was accepted; want load error")
	} else if !strings.Contains(err.Error(), "models.opencode") {
		t.Fatalf("error %v does not mention missing opencode model routes", err)
	}
}
```

Use the same `loadDoc` helper `TestLoadRequiresAllModelLevelsForEnabledTools` uses.

- [ ] **Step 6: Run the test to verify it passes**

Run: `go test ./internal/config/ -run TestLoadRequiresModelsForSubagentTargetedTool -v`
Expected: PASS (confirms existing validation covers subagents — no production change needed).

- [ ] **Step 7: Commit**

```bash
git add internal/config
git commit -m "feat(config): ExpandedSubagentEntriesForTool + model-route gap test"
```

---

## Task 5: Engine materialization orchestration + `WithSubagentCatalogRoot` wiring

**Files:**
- Modify: `internal/engine/engine.go`
- Modify: `internal/engine/materialize_test.go`

**Interfaces:**
- Consumes: `config.ExpandedSubagentEntriesForTool`, `catalog.MaterializeSubagents`, both adapters' `WithSubagentCatalogRoot` (added in Task 6 — this task adds the engine field + Build call; Task 6 adds the adapter method).
- Produces: `Engine.SubagentCatalogRoot string`, `(e *Engine) SubagentDir() string`, subagent collection + version gate in `materializeCatalog`, `allSubagentFilesExist`.

> **Ordering note:** the `Build` change in Step 5 calls `.WithSubagentCatalogRoot(...)` on each adapter, which does not exist until Task 6. To keep this task independently compilable, add the adapter `WithSubagentCatalogRoot` method (Task 6 Step 1) **before** running this task's build — or execute Task 6 Step 1 first, then this task, then the rest of Task 6. The subagent-driven executor should treat "engine field + adapter setter" as the compilable unit; the plan lists them separately only for review clarity.

- [ ] **Step 1: Write the failing engine tests**

In `internal/engine/materialize_test.go`, add (mirror `TestApplyMaterializesBuiltinCommand` and `TestApplyRematerializesWhenCommandFileMissing`, using a subagent config and asserting the file at `<SubagentDir()>/<name>.md`):

```go
func TestApplyMaterializesBuiltinSubagent(t *testing.T) {
	e := buildEngineWithSubagent(t) // helper below
	if err := e.Apply(mustPlan(t, e)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	p := filepath.Join(e.SubagentDir(), "code-reviewer.md")
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("subagent not materialized: %v", err)
	}
}

func TestApplyRematerializesWhenSubagentFileMissing(t *testing.T) {
	e := buildEngineWithSubagent(t)
	if err := e.Apply(mustPlan(t, e)); err != nil {
		t.Fatalf("first apply: %v", err)
	}
	p := filepath.Join(e.SubagentDir(), "code-reviewer.md")
	if err := os.Remove(p); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if err := e.Apply(mustPlan(t, e)); err != nil {
		t.Fatalf("second apply: %v", err)
	}
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("subagent not re-materialized when file missing: %v", err)
	}
}
```

Reuse the existing test scaffolding pattern from the command tests in this file (`buildEngine…`, `mustPlan`, the temp `homonto.toml` writer). Add a `buildEngineWithSubagent` helper that writes a `homonto.toml` declaring `[subagents.code-reviewer] source = "builtin:code-reviewer"` with `scope = "project"` and the required `[models.*]` blocks, exactly mirroring how the existing command engine test builds its config.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/engine/ -run 'TestApply.*Subagent' -v`
Expected: FAIL — `SubagentDir` undefined and no subagent materialization.

- [ ] **Step 3: Add the engine field, `SubagentDir`, and Build wiring**

In `internal/engine/engine.go`:

Add to `Engine`:
```go
	CommandCatalogRoot string // materialized builtin command root (<stateDir>/catalog/commands)
	SubagentCatalogRoot string // materialized builtin subagent root (<stateDir>/catalog/subagents)
```

In `Build`, add the dir:
```go
	commandCatalogDir := filepath.Join(stateDir, "catalog", "commands")
	subagentCatalogDir := filepath.Join(stateDir, "catalog", "subagents")
```

Wire both adapters (append `.WithSubagentCatalogRoot(subagentCatalogDir)`):
```go
			claude.New(home, contentDir).WithProjectRoot(projectRoot).WithCatalogRoot(catalogDir).WithCommandCatalogRoot(commandCatalogDir).WithSubagentCatalogRoot(subagentCatalogDir),
			opencode.New(home, contentDir).WithProjectRoot(projectRoot).WithCatalogRoot(catalogDir).WithCommandCatalogRoot(commandCatalogDir).WithSubagentCatalogRoot(subagentCatalogDir),
```

Set the field in the returned `&Engine{…}`:
```go
		CommandCatalogRoot: commandCatalogDir,
		SubagentCatalogRoot: subagentCatalogDir,
```

Add the accessor after `CommandDir`:
```go
// SubagentDir returns the materialized builtin subagent root.
func (e *Engine) SubagentDir() string { return e.SubagentCatalogRoot }
```

- [ ] **Step 4: Extend `materializeCatalog` with the subagent set + gate**

In `internal/engine/engine.go`, in `materializeCatalog`:

Add the collection set alongside `skillSet`/`cmdSet`:
```go
	skillSet := map[string]bool{}
	cmdSet := map[string]bool{}
	subSet := map[string]bool{}
```

Inside the per-tool loop, after the command-entries block, add:
```go
		saEntries, err := e.Cfg.ExpandedSubagentEntriesForTool(tool)
		if err != nil {
			return err
		}
		for _, entry := range saEntries {
			if strings.HasPrefix(entry.Resource.Source, "builtin:") {
				subSet[strings.TrimPrefix(entry.Resource.Source, "builtin:")] = true
			}
		}
```

Update the early-return guard:
```go
	if len(skillSet) == 0 && len(cmdSet) == 0 && len(subSet) == 0 {
		return nil
	}
```

Build the sorted names slice alongside the others:
```go
	subNames := make([]string, 0, len(subSet))
	for n := range subSet {
		subNames = append(subNames, n)
	}
	sort.Strings(subNames)
```

Extend the version gate to also require subagent files present:
```go
	if e.State.CatalogVersionRecorded() == cl.Version() &&
		allSkillDirsExist(e.CatalogRoot, skillNames) &&
		allCommandFilesExist(e.CommandCatalogRoot, cmdNames) &&
		allSubagentFilesExist(e.SubagentCatalogRoot, subNames) {
		return nil
	}
```

Materialize subagents after commands, before recording the version:
```go
	if err := cl.MaterializeCommands(e.CommandCatalogRoot, cmdNames); err != nil {
		return err
	}
	if err := cl.MaterializeSubagents(e.SubagentCatalogRoot, subNames); err != nil {
		return err
	}
	e.State.SetCatalogVersion(cl.Version())
```

Add the helper next to `allCommandFilesExist` (single-file, identical shape):
```go
func allSubagentFilesExist(root string, names []string) bool {
	for _, n := range names {
		fi, err := os.Stat(filepath.Join(root, n+".md"))
		if err != nil || fi.IsDir() {
			return false
		}
	}
	return true
}
```

- [ ] **Step 5: Add the adapter `WithSubagentCatalogRoot` setter (both adapters) so Build compiles**

This is Task 6 Step 1; add it now (see Task 6) if executing tasks strictly in order. Minimum needed here: the field `subagentCatalogRoot string` + the `WithSubagentCatalogRoot` method on each adapter, returning the adapter.

- [ ] **Step 6: Run the tests to verify they pass**

Run: `go test ./internal/engine/ -run 'TestApply' -count=1 -v`
Expected: PASS, including the two new subagent tests and every pre-existing skill/command materialization test (the shared version gate must still record only after all three kinds materialize).

- [ ] **Step 7: Commit**

```bash
git add internal/engine/engine.go internal/engine/materialize_test.go internal/adapter
git commit -m "feat(engine): materialize builtin subagents under shared version gate; WithSubagentCatalogRoot wiring"
```

---

## Task 6: Adapter subagent projection (both tools)

Apply every step to **both** `internal/adapter/claude/claude.go` and `internal/adapter/opencode/opencode.go` — they are structurally identical for links (only `desired()`, MCP JSON shape, and the tool-name literal differ, none of which touch subagents). Use `commandpath.Dir(…)` as the exact model but call `subagentpath.Dir("<tool>", …)` and file suffix `.md`.

**Files:**
- Modify: `internal/adapter/claude/claude.go`, `internal/adapter/opencode/opencode.go`
- Modify: `internal/adapter/claude/util.go`, `internal/adapter/opencode/util.go`
- Modify: `internal/adapter/claude/builtin_test.go` (and the opencode equivalent test file)

**Interfaces:**
- Consumes: `config.ExpandedSubagentEntriesForTool`, `subagentpath.Dir`, `skillpath.Other`, `link.Plan/Link/Remove/IsManaged`, `state.State`, `recordedDst`.
- Produces on each adapter: `subagentCatalogRoot` field, `subagents []config.NamedResource` field, `WithSubagentCatalogRoot`, `subagentsDir`, `inactiveSubagentsDir`, `subagentSource`, `subagentLinks`; `subagent.*` handling in `Plan`, `Apply`, `ObserveHashes`, the declared-keys map, and `managedPrefix`.

- [ ] **Step 1: Write the failing adapter tests (both tools)**

In `internal/adapter/claude/builtin_test.go`, add (mirror `TestBuiltinCommandLinksToCommandCatalogRoot`, `TestBuiltinCommandPrunedWhenDeDeclared`, `TestBuiltinCommandConflictNotClobbered`, targeting `.claude/agents/<name>.md` and the `subagent.<name>` state key). Add a `builtinSubagentCfg()` helper returning a `*config.Config` with `[subagents.code-reviewer] source="builtin:code-reviewer"`, scope project (or user, matching how `builtinCmdCfg()` sets scope), targeting claude:

```go
func TestBuiltinSubagentLinksToSubagentCatalogRoot(t *testing.T) {
	home := t.TempDir()
	saRoot := t.TempDir()
	os.WriteFile(filepath.Join(saRoot, "code-reviewer.md"), []byte("body"), 0o644)

	a := New(home, t.TempDir()).WithSubagentCatalogRoot(saRoot)
	st, _ := state.Load(t.TempDir())
	cs, err := a.Plan(builtinSubagentCfg(), st)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if err := a.Apply(cs, resolver(), st); err != nil {
		t.Fatalf("apply: %v", err)
	}

	dst := filepath.Join(home, ".claude", "agents", "code-reviewer.md")
	target, err := os.Readlink(dst)
	if err != nil {
		t.Fatalf("subagent link missing: %v", err)
	}
	if want := filepath.Join(saRoot, "code-reviewer.md"); target != want {
		t.Fatalf("link target = %q, want %q", target, want)
	}
	if _, ok := st.Get("claude", "subagent.code-reviewer"); !ok {
		t.Fatal("subagent.code-reviewer not recorded in state")
	}
	cs2, _ := a.Plan(builtinSubagentCfg(), st)
	for _, c := range cs2.Changes {
		if c.Key == "subagent.code-reviewer" && c.Action != "noop" {
			t.Fatalf("re-plan not idempotent: %+v", c)
		}
	}
}

func TestBuiltinSubagentPrunedWhenDeDeclared(t *testing.T) {
	home := t.TempDir()
	saRoot := t.TempDir()
	os.WriteFile(filepath.Join(saRoot, "code-reviewer.md"), []byte("body"), 0o644)
	a := New(home, t.TempDir()).WithSubagentCatalogRoot(saRoot)
	st, _ := state.Load(t.TempDir())
	cs, _ := a.Plan(builtinSubagentCfg(), st)
	if err := a.Apply(cs, resolver(), st); err != nil {
		t.Fatalf("apply: %v", err)
	}
	dst := filepath.Join(home, ".claude", "agents", "code-reviewer.md")
	cs2, err := a.Plan(&config.Config{}, st)
	if err != nil {
		t.Fatalf("plan empty: %v", err)
	}
	if err := a.Apply(cs2, resolver(), st); err != nil {
		t.Fatalf("apply empty: %v", err)
	}
	if _, err := os.Lstat(dst); !os.IsNotExist(err) {
		t.Fatal("de-declared builtin subagent link not pruned")
	}
}

func TestBuiltinSubagentConflictNotClobbered(t *testing.T) {
	home := t.TempDir()
	saRoot := t.TempDir()
	os.WriteFile(filepath.Join(saRoot, "code-reviewer.md"), []byte("body"), 0o644)
	dst := filepath.Join(home, ".claude", "agents", "code-reviewer.md")
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(dst, []byte("REAL USER FILE"), 0o644)

	a := New(home, t.TempDir()).WithSubagentCatalogRoot(saRoot)
	st, _ := state.Load(t.TempDir())
	cs, _ := a.Plan(builtinSubagentCfg(), st)
	if err := a.Apply(cs, resolver(), st); err == nil {
		t.Fatal("expected conflict error, got nil")
	}
	b, _ := os.ReadFile(dst)
	if string(b) != "REAL USER FILE" {
		t.Fatal("conflicting real file was clobbered")
	}
}

func TestBuiltinSubagentAdoptsExistingLink(t *testing.T) {
	home := t.TempDir()
	saRoot := t.TempDir()
	src := filepath.Join(saRoot, "code-reviewer.md")
	os.WriteFile(src, []byte("body"), 0o644)
	dst := filepath.Join(home, ".claude", "agents", "code-reviewer.md")
	os.MkdirAll(filepath.Dir(dst), 0o755)
	if err := os.Symlink(src, dst); err != nil {
		t.Fatal(err)
	}
	a := New(home, t.TempDir()).WithSubagentCatalogRoot(saRoot)
	st, _ := state.Load(t.TempDir())
	cs, _ := a.Plan(builtinSubagentCfg(), st)
	adopted := false
	for _, c := range cs.Changes {
		if c.Key == "subagent.code-reviewer" && c.Action == "adopt" {
			adopted = true
		}
	}
	if !adopted {
		t.Fatalf("pre-existing correct link not adopted: %+v", cs.Changes)
	}
	if err := a.Apply(cs, resolver(), st); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if tgt, _ := os.Readlink(dst); tgt != src {
		t.Fatal("adopt must leave the on-disk link untouched")
	}
}
```

Add the same four tests to the OpenCode adapter's test file (the file holding `TestBuiltinCommand…` for opencode), changing the destination to `filepath.Join(home, ".config", "opencode", "agent", "code-reviewer.md")` and the state tool to `"opencode"`. For a **scope-switch relocate** test, mirror whatever the command suite already has (if the command suite has a scope-switch test, copy it for subagents on at least one adapter; if not, add one on Claude: apply with `scope="user"`, then re-plan/apply the same subagent with `scope="project"` and assert the user-scope link is gone and the project-scope link exists, and the plan rendered an `update` for `subagent.code-reviewer`).

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/adapter/claude/ ./internal/adapter/opencode/ -run 'TestBuiltinSubagent' -v`
Expected: FAIL — `WithSubagentCatalogRoot` and subagent projection undefined.

- [ ] **Step 3: Add the field, setter, and `managedRoots` extension (both adapters)**

Add the import `"github.com/noviopenworks/homonto/internal/subagentpath"` to each adapter.

Add to the `Adapter` struct:
```go
	commandCatalogRoot  string // materialized builtin command root (.homonto/catalog/commands)
	subagentCatalogRoot string // materialized builtin subagent root (.homonto/catalog/subagents)
	...
	commands  []config.NamedResource
	subagents []config.NamedResource
```

Add the setter after `WithCommandCatalogRoot`:
```go
// WithSubagentCatalogRoot sets the materialized builtin-subagent root that
// builtin:<name> subagents link from. Mirrors WithCommandCatalogRoot.
func (a *Adapter) WithSubagentCatalogRoot(subagentCatalogRoot string) *Adapter {
	a.subagentCatalogRoot = subagentCatalogRoot
	return a
}
```

Extend `managedRoots` (non-empty guard preserved):
```go
	if a.commandCatalogRoot != "" {
		roots = append(roots, a.commandCatalogRoot)
	}
	if a.subagentCatalogRoot != "" {
		roots = append(roots, a.subagentCatalogRoot)
	}
	return roots
```

- [ ] **Step 4: Add the subagent dir/source/link helpers (both adapters)**

Add after `commandLinks` (use `"claude"` in the claude adapter and `"opencode"` in the opencode adapter for the tool literal):
```go
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
// builtin:<n> from the materialized subagent root (<n>.md), otherwise the local
// content dir (homonto/subagents/<n>.md).
func (a *Adapter) subagentSource(entry config.NamedResource) string {
	if s := entry.Resource.Source; strings.HasPrefix(s, "builtin:") {
		return filepath.Join(a.subagentCatalogRoot, strings.TrimPrefix(s, "builtin:")+".md")
	}
	return filepath.Join(a.content, "subagents", localSourceName(entry.Resource.Source, entry.Name)+".md")
}

// subagentLinks maps each owned subagent's destination (<name>.md) to its source.
func (a *Adapter) subagentLinks() map[string]string {
	out := map[string]string{}
	for _, entry := range a.subagents {
		out[filepath.Join(a.subagentsDir(entry.Resource.Scope), entry.Name+".md")] = a.subagentSource(entry)
	}
	return out
}
```

- [ ] **Step 5: Wire subagents into `Plan` (both adapters)**

In `Plan`, after the `a.commands = commands` assignment, load subagents:
```go
	subagents, err := c.ExpandedSubagentEntriesForTool("claude") // "opencode" in the opencode adapter
	if err != nil {
		return adapter.ChangeSet{}, err
	}
	a.subagents = subagents
```

After the command-links block (the `cmdOps` block ending with the command adopt loop), add the parallel subagent-links block — a verbatim copy of the command block with `cmd`→`sub`, `command.`→`subagent.`, `commandLinks`→`subagentLinks`, `inactiveCommandsDir`→`inactiveSubagentsDir`, `commandsDir`→`subagentsDir`, `commandSource`→`subagentSource`:
```go
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
		if e, ok := st.Get("claude", "subagent."+entry.Name); ok && e.Applied == secret.Hash(dst+" -> "+src) { // "opencode" in the opencode adapter
			continue
		}
		cs.Changes = append(cs.Changes, adapter.Change{Action: "adopt", Key: "subagent." + entry.Name, New: dst + " -> " + src})
	}
```

In the orphan-pruning declared-map block, add subagents next to commands:
```go
	for _, entry := range a.commands {
		declared["command."+entry.Name] = true
	}
	for _, entry := range a.subagents {
		declared["subagent."+entry.Name] = true
	}
```

(The `sort.SliceStable` at the end already covers the new keys.)

- [ ] **Step 6: Wire subagents into `Apply` (both adapters)**

In the change loop, extend the `adopt` branch:
```go
			if hasPrefix(c.Key, "command.") {
				st.Set("claude", c.Key, c.New, secret.Hash(c.New))
				continue
			}
			if hasPrefix(c.Key, "subagent.") {
				st.Set("claude", c.Key, c.New, secret.Hash(c.New)) // "opencode" in the opencode adapter
				continue
			}
```

In the `delete` switch, add a `subagent.` case after the `command.` case:
```go
			case hasPrefix(c.Key, "subagent."):
				name := trim(c.Key, "subagent.")
				dst := ""
				if e, ok := st.Get("claude", c.Key); ok { // "opencode" in opencode
					dst, _ = recordedDst(e.Desired)
				}
				if dst == "" {
					dst = filepath.Join(a.subagentsDir("user"), name+".md")
				}
				err = link.Remove(dst, a.managedRoots()...)
```

After the `command.` symlink-skip guard, add:
```go
		if hasPrefix(c.Key, "command.") {
			continue
		}
		if hasPrefix(c.Key, "subagent.") {
			continue
		}
```

After the command `link.Plan` conflict pre-check, add the subagent pre-check (before any file write):
```go
	cmdLinks := a.commandLinks()
	if _, err := link.Plan(cmdLinks, a.managedRoots()...); err != nil {
		return err
	}
	subLinks := a.subagentLinks()
	if _, err := link.Plan(subLinks, a.managedRoots()...); err != nil {
		return err
	}
```

After the command inactive-scope prune + link loop (the block ending with `st.Set("claude", "command."+…)`), add the subagent equivalent:
```go
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
		st.Set("claude", "subagent."+strings.TrimSuffix(filepath.Base(dst), ".md"), dst+" -> "+src, secret.Hash(dst+" -> "+src)) // "opencode" in opencode
	}
```

- [ ] **Step 7: Wire subagents into `ObserveHashes` (both adapters)**

After the `command.` branch in `ObserveHashes`, add:
```go
		if hasPrefix(key, "subagent.") {
			e, ok := st.Get("claude", key) // "opencode" in opencode
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

- [ ] **Step 8: Add `"subagent."` to `managedPrefix` (both util.go files)**

In `internal/adapter/claude/util.go` and `internal/adapter/opencode/util.go`:
```go
	for _, p := range []string{"mcp.", "setting.", "plugin.", "skill.", "command.", "subagent."} {
```

- [ ] **Step 9: Run the tests to verify they pass**

Run: `go test ./internal/adapter/... -count=1 -v`
Expected: PASS, including the new subagent tests on both adapters and every pre-existing skill/command adapter test.

- [ ] **Step 10: Commit**

```bash
git add internal/adapter
git commit -m "feat(adapter): project subagents into claude agents/ and opencode agent/ (plan/apply/adopt/prune/relocate)"
```

---

## Task 7: Doctor verification

**Files:**
- Modify: `internal/engine/status.go`
- Modify: the engine doctor test file (wherever `doctorCommands` is tested)

**Interfaces:**
- Consumes: `config.ExpandedSubagentEntriesForTool`, `subagentpath.Dir`, `Engine.SubagentDir`, `Engine.ContentDir`.
- Produces: `(e *Engine) doctorSubagents(tool string, entries []config.NamedResource) []string` and its two `Doctor()` call sites (claude + opencode).

- [ ] **Step 1: Write the failing doctor test**

Add a test mirroring the existing command-doctor test: build an engine with a declared builtin subagent, `Apply`, then assert `Doctor()` output contains `ok: subagent "code-reviewer" linked (claude)` and `... (opencode)`. Reuse the command doctor test's scaffolding verbatim, swapping "command" → "subagent".

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./internal/engine/ -run 'Doctor.*Subagent|Subagent.*Doctor' -v`
Expected: FAIL — `doctorSubagents` undefined / no subagent lines emitted.

- [ ] **Step 3: Add `doctorSubagents` and wire it into `Doctor()`**

Add the import `"github.com/noviopenworks/homonto/internal/subagentpath"` to `status.go`.

Add after `doctorCommands` (verbatim copy with `Command`→`Subagent`, `command`→`subagent`, `CommandDir`→`SubagentDir`, `commandpath.Dir`→`subagentpath.Dir`, `"commands"`→`"subagents"`):
```go
// doctorSubagents reports, per subagent, whether its content file is present at
// the right source (builtin: from the materialized subagent root, local: from
// the content dir) and whether it is linked into the tool's agent directory.
func (e *Engine) doctorSubagents(tool string, entries []config.NamedResource) []string {
	var out []string
	for _, entry := range entries {
		name := entry.Name
		var p string
		if strings.HasPrefix(entry.Resource.Source, "builtin:") {
			p = filepath.Join(e.SubagentDir(), strings.TrimPrefix(entry.Resource.Source, "builtin:")+".md")
		} else {
			sourceName := name
			if strings.HasPrefix(entry.Resource.Source, "local:") {
				sourceName = strings.TrimPrefix(entry.Resource.Source, "local:")
			}
			p = filepath.Join(e.ContentDir, "subagents", sourceName+".md")
		}
		if _, err := os.Stat(p); err != nil {
			out = append(out, fmt.Sprintf("warn: subagent %q missing from %s (run apply)", name, p))
			continue
		}
		dst := filepath.Join(subagentpath.Dir(tool, entry.Resource.Scope, e.Home, e.ProjectRoot), name+".md")
		if target, err := os.Readlink(dst); err == nil && target == p {
			out = append(out, fmt.Sprintf("ok: subagent %q linked (%s)", name, tool))
		} else {
			out = append(out, fmt.Sprintf("warn: subagent %q content present, not linked for %s (run apply)", name, tool))
		}
	}
	return out
}
```

In `Doctor()`, after the two `doctorCommands` call sites, add:
```go
	claudeSubagents, csaerr := e.Cfg.ExpandedSubagentEntriesForTool("claude")
	if csaerr != nil {
		out = append(out, fmt.Sprintf("warn: cannot expand claude subagents: %v", csaerr))
	} else {
		out = append(out, e.doctorSubagents("claude", claudeSubagents)...)
	}
	opencodeSubagents, osaerr := e.Cfg.ExpandedSubagentEntriesForTool("opencode")
	if osaerr != nil {
		out = append(out, fmt.Sprintf("warn: cannot expand opencode subagents: %v", osaerr))
	} else {
		out = append(out, e.doctorSubagents("opencode", opencodeSubagents)...)
	}
```

- [ ] **Step 4: Run it to verify it passes**

Run: `go test ./internal/engine/ -count=1 -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/engine/status.go internal/engine
git commit -m "feat(doctor): verify subagent links and materialized files for both tools"
```

---

## Task 8: Dogfood

**Files:**
- Modify: `homonto.toml`

**Interfaces:** Consumes the fully built pipeline; produces on-disk links + state validating the whole change end to end.

- [ ] **Step 1: Declare the two loose subagents in `homonto.toml`**

Add after the `[commands.example-command]` block (keep the existing `[frameworks.comet]` block — it supplies the `comet-navigator` framework subagent):
```toml
[subagents.code-reviewer]
source = "builtin:code-reviewer"
scope = "project"

[subagents.codebase-explorer]
source = "builtin:codebase-explorer"
scope = "project"
```

- [ ] **Step 2: Build and apply**

Run:
```bash
go build -o /tmp/homonto ./... 2>/dev/null || go build -o /tmp/homonto ./cmd/... 2>/dev/null; \
go run . apply --yes
```
(Use whatever the repo's real entrypoint is — check `cli/`/`cmd/` for `main`; the existing dogfood in command-projection used the same invocation. If `go run .` is not the entrypoint, use the module's main package path.)

Expected: apply succeeds; it materializes `code-reviewer`, `codebase-explorer`, and `comet-navigator` under `.homonto/catalog/subagents/` and creates links:
- `.claude/agents/code-reviewer.md`, `.claude/agents/codebase-explorer.md`, `.claude/agents/comet-navigator.md`
- `.opencode/agent/code-reviewer.md`, `.opencode/agent/codebase-explorer.md`, `.opencode/agent/comet-navigator.md`

- [ ] **Step 3: Verify no drift and doctor OK**

Run: `go run . status` — expected: **No drift** (a second status shows every subagent as a noop). Then `go run . doctor` — expected: `ok: subagent "code-reviewer" linked (claude)` / `(opencode)` and the same for `codebase-explorer` and `comet-navigator`.

Manually confirm one materialized file is byte-for-byte the catalog source:
```bash
diff .homonto/catalog/subagents/code-reviewer.md catalog/subagents/code-reviewer.md && echo "verbatim OK"
```
Expected: no diff, `verbatim OK`.

- [ ] **Step 4: Commit**

```bash
git add homonto.toml
git commit -m "chore: dogfood code-reviewer and codebase-explorer subagents"
```

---

## Task 9: Regression and docs

**Files:**
- Modify: `README.md`, `docs/guides/using-homonto.md`, `docs/roadmap.md`

- [ ] **Step 1: Full regression suite**

Run each and confirm clean:
```bash
go test ./... -count=1
go test -race ./...
go vet ./...
go build ./...
gofmt -l .
```
Expected: all tests PASS; `go vet` and `go build` clean; `gofmt -l .` prints nothing.

- [ ] **Step 2: Stale-doc grep**

Run: `grep -rn -i "subagent" README.md docs/`
For every hit claiming subagent projection is unimplemented/pending/parsed-but-ignored, update it to reflect that projection now ships with real content. Focus points (confirmed present at plan time):
- `README.md:144` "Known limitations" and lines ~151/159 (currently: "subagents are... [not projected]"). Replace with a statement that framework skill, command, **and subagent** projection are implemented; the `[subagents.X]` table now materializes and links into Claude Code `agents/` and OpenCode `agent/`.
- `docs/guides/using-homonto.md:181` "Known limitations" and ~192 (currently: "`[subagents.X]` is parsed and validated ... [but not projected]"). Update to describe subagent projection as working, with the directory table (Claude `agents/`, OpenCode `agent/`, user and project scopes).

- [ ] **Step 3: Update the roadmap**

In `docs/roadmap.md`:
- "Immediate Next Work" item 2 ("Subagent projection (`[subagents.X]`)", lines ~48-55): mark **done** — subagent projection landed on `main` with real bundled content (`code-reviewer`, `codebase-explorer`, comet's `comet-navigator`). The remaining immediate work is the `onto` binary.
- v1.1 status (line ~69 area): note subagent projection landed with real content alongside skills and commands.

- [ ] **Step 4: Final commit**

```bash
git add README.md docs/guides/using-homonto.md docs/roadmap.md
git commit -m "docs: subagent projection shipped with real content (roadmap, README, guide)"
```

---

## Self-Review

**Spec coverage** (delta specs + Design Doc → task):
- Builtin/local source resolution (`subagent.spec` R1; `config-model.spec`): `subagentSource` handles `builtin:`→`.homonto/catalog/subagents/<n>.md` and `local:`→`homonto/subagents/<n>.md` — Task 6 Step 4.
- Single-file verbatim materialization, version-gated, no model injection (`subagent.spec` R2; D2): `MaterializeSubagents` (byte-for-byte, Task 3 Step 5 + test Step 1) under the shared version gate (Task 5 Step 4).
- Projection into agent dirs, plan changes, adopt/prune/relocate, conflict-safe (`subagent.spec` R3, R4; D3, D6): Task 6 Steps 5-7 + `subagentpath` Task 1.
- Framework `[subagents]` expansion (`subagent.spec` R5, `framework-expansion.spec`; D4): catalog parse Task 3 Step 3, `ExpandSubagents` Step 4, comet wiring Step 6, config expansion Task 4.
- Doctor verification (`subagent.spec` R6): Task 7.
- Three real bundled subagents with minimal shared frontmatter (`subagent.spec` R7; D5): Task 2 + the frontmatter contract Task 1.
- Fixtures-first / empirical frontmatter load check before adapters (Design Risks, tasks.md 1.1): Task 1 Steps 1-3.
- Model-validation gap already covered by `EnabledModelTools` (Design Risks): confirming test Task 4 Step 5.

**Placeholder scan:** every code step carries complete code; no "TBD"/"similar to"/"add validation" left. The two places that say "mirror the command test scaffolding" point at named existing tests (`TestBuiltinCommand…`, `TestApplyMaterializesBuiltinCommand`, `builtinCmdCfg`, `loadDoc`) whose exact shape the executor copies — this is deliberate reuse of proven helpers, not a missing spec.

**Type consistency:** `subagentpath.Dir`, `Framework.Subagents`/`Catalog.subagents`/`SubagentPath`, `ExpandedSubagent`/`ExpandSubagents`, `MaterializeSubagents`, `SubagentEntriesForTool`/`ExpandedSubagentEntriesForTool`, `Engine.SubagentCatalogRoot`/`SubagentDir`, adapter `subagentCatalogRoot`/`subagents`/`WithSubagentCatalogRoot`/`subagentsDir`/`inactiveSubagentsDir`/`subagentSource`/`subagentLinks`, state key prefix `subagent.`, and doctor `doctorSubagents` are used consistently across every task. State-key semantics (`Hash(dst + " -> " + src)`) match `command.` exactly.

**Cross-task ordering caveat:** Task 5's `Build` change references the adapter setter added in Task 6 Step 3; the note in Task 5 tells the executor to land the adapter field+setter together with the engine field so each commit compiles. Executors using strict per-task isolation should fold Task 6 Step 3 into Task 5 or run `go build ./...` only after both are present.
