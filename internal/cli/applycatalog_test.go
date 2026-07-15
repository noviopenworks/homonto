package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// catalogSubagentTOML installs a builtin subagent whose rendered frontmatter is
// stamped from the [models.opencode.*] routes. %MODEL% is the architectural
// route that `code-reviewer`'s `role: architectural` resolves through.
//
// [settings.opencode].model is pinned explicitly, which wins over the
// route-derived default — so changing the architectural route re-stamps the
// AGENT while projecting no setting diff at all. That is the case this file
// exists for: the projection plan comes out empty, and an empty plan is exactly
// what the CLI used to treat as "nothing to do".
const catalogSubagentTOML = `
[subagents.code-reviewer]
source = "builtin:code-reviewer"
scope = "project"
targets = ["opencode"]

[settings.opencode]
model = "pinned/explicit-model"

[models.opencode.architectural]
model = "%MODEL%"
variant = "high"
[models.opencode.coding]
model = "some/coding-model"
effort = "medium"
[models.opencode.trivial]
model = "some/trivial-model"
effort = "medium"
`

func writeCatalogConfig(t *testing.T, repo, model string) string {
	t.Helper()
	cfg := filepath.Join(repo, "homonto.toml")
	body := strings.Replace(catalogSubagentTOML, "%MODEL%", model, 1)
	if err := os.WriteFile(cfg, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return cfg
}

func renderedVariant(t *testing.T, repo string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repo, ".homonto", "catalog", "subagents", "code-reviewer.opencode.md"))
	if err != nil {
		t.Fatalf("read rendered variant: %v", err)
	}
	return string(data)
}

// TestApplyRematerializesCatalogWhenProjectionPlanIsEmpty guards the CLI-level
// half of the stale-render bug. A catalog file's symlink target is name-based,
// so re-rendering a subagent changes no projected value and the projection plan
// comes out EMPTY — and an empty plan used to skip apply entirely ("No changes.
// Everything up to date."), so the engine's materialize never ran and the agent
// stayed frozen at its old model. Deleting the rendered variant has the same
// shape: no plan change, no repair, a dangling link forever.
//
// Both cases must instead run apply, exactly as the HasRemoteResources carve-out
// already does for the identical name-based-target reason.
func TestApplyRematerializesCatalogWhenProjectionPlanIsEmpty(t *testing.T) {
	t.Run("stale render after a model-route change", func(t *testing.T) {
		home := t.TempDir()
		repo := t.TempDir()
		cfg := writeCatalogConfig(t, repo, "first/model-a")
		if out, err := runCmd(t, home, "", "apply", "--yes", "--config", cfg); err != nil {
			t.Fatalf("first apply: %v\n%s", err, out)
		}
		if got := renderedVariant(t, repo); !strings.Contains(got, "model: first/model-a") {
			t.Fatalf("first apply did not stamp the route:\n%s", got)
		}

		// Change ONLY the model route. settings.opencode.model is pinned, so no
		// projected value moves: the plan is empty, and the CLI's empty-plan
		// branch alone decides whether apply — and thus the re-render — runs.
		writeCatalogConfig(t, repo, "second/model-b")
		out, err := runCmd(t, home, "", "apply", "--yes", "--config", cfg)
		if err != nil {
			t.Fatalf("second apply: %v\n%s", err, out)
		}
		if strings.Contains(out, "setting.model") {
			t.Fatalf("precondition broken: the plan was not empty, so this no longer tests the empty-plan path:\n%s", out)
		}
		if got := renderedVariant(t, repo); !strings.Contains(got, "model: second/model-b") {
			t.Fatalf("route change did not re-render the agent (CLI skipped apply on an empty plan):\n%s", got)
		}
	})

	t.Run("deleted rendered variant", func(t *testing.T) {
		home := t.TempDir()
		repo := t.TempDir()
		cfg := writeCatalogConfig(t, repo, "first/model-a")
		if out, err := runCmd(t, home, "", "apply", "--yes", "--config", cfg); err != nil {
			t.Fatalf("first apply: %v\n%s", err, out)
		}
		variant := filepath.Join(repo, ".homonto", "catalog", "subagents", "code-reviewer.opencode.md")
		if err := os.Remove(variant); err != nil {
			t.Fatal(err)
		}

		out, err := runCmd(t, home, "", "apply", "--yes", "--config", cfg)
		if err != nil {
			t.Fatalf("repair apply: %v\n%s", err, out)
		}
		if _, err := os.Stat(variant); err != nil {
			t.Fatalf("apply left the projected link dangling — variant not restored: %v\n%s", err, out)
		}
	})

	t.Run("a genuinely settled workspace still reports no changes", func(t *testing.T) {
		home := t.TempDir()
		repo := t.TempDir()
		cfg := writeCatalogConfig(t, repo, "first/model-a")
		if out, err := runCmd(t, home, "", "apply", "--yes", "--config", cfg); err != nil {
			t.Fatalf("first apply: %v\n%s", err, out)
		}
		// Nothing changed: the carve-out must not fire, or every apply would
		// re-materialize and the no-op path would never be reachable.
		out, err := runCmd(t, home, "", "apply", "--yes", "--config", cfg)
		if err != nil {
			t.Fatalf("second apply: %v\n%s", err, out)
		}
		if !strings.Contains(out, "No changes. Everything up to date.") {
			t.Fatalf("a settled workspace must report no changes, got:\n%s", out)
		}
	})
}
