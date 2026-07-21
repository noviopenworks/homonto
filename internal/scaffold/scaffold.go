package scaffold

import (
	"os"
	"path/filepath"
	"strings"
)

var files = map[string]string{
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

# [subagents.reviewer]
# source = "builtin:onto-reviewer"
# scope = "project"
# Every declared subagent MUST declare a [subagents.<name>.<tool>] block per
# target tool with a non-empty model. Effort and variant are optional.
# [subagents.reviewer.claude]
# model = "opus"
# effort = "high"
# [subagents.reviewer.opencode]
# model = "anthropic/claude-opus-4-8"

# [plugins.claude.claude-hud]
# source = "claude-hud@official"       # name@marketplace
# [plugins.opencode.opencode-quota]
# source = "@slkiser/opencode-quota"   # npm package

# The main session model is operator-controlled. homonto projects it ONLY when
# you declare it explicitly here; otherwise each tool uses its own default.
# [settings.claude]
# model = "opus"
# [settings.opencode]
# model = "anthropic/claude-opus-4-8"
`,
	".gitignore":   "/.homonto/\n.env\n",
	".env.example": "# Document non-pass secrets here, then copy to .env (gitignored).\n# BRAVE_API_KEY=\n",
}

// Init scaffolds a homonto repo. It creates missing files, and — for an
// existing .gitignore — augments it with any missing homonto entries (so a repo
// that already has a .gitignore still ignores /.homonto/ and .env) rather than
// silently skipping it. It returns the files it created and the ones it updated.
func Init(dir string) (created, updated []string, err error) {
	for name, body := range files {
		p := filepath.Join(dir, name)
		if _, statErr := os.Stat(p); statErr == nil {
			if name == ".gitignore" {
				augmented, augErr := augmentGitignore(p, body)
				if augErr != nil {
					return created, updated, augErr
				}
				if augmented {
					updated = append(updated, p)
				}
			}
			continue
		}
		if writeErr := os.WriteFile(p, []byte(body), 0o644); writeErr != nil {
			return created, updated, writeErr
		}
		created = append(created, p)
	}
	keep := filepath.Join(dir, "homonto", "skills", ".gitkeep")
	if _, statErr := os.Stat(keep); statErr != nil {
		if mkErr := os.MkdirAll(filepath.Dir(keep), 0o755); mkErr != nil {
			return created, updated, mkErr
		}
		if writeErr := os.WriteFile(keep, nil, 0o644); writeErr != nil {
			return created, updated, writeErr
		}
		created = append(created, keep)
	}
	return created, updated, nil
}

// augmentGitignore appends to path any newline-separated entry in want that is
// not already present, preserving existing content. It reports whether it wrote.
func augmentGitignore(path, want string) (bool, error) {
	existing, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	have := map[string]bool{}
	for _, l := range strings.Split(string(existing), "\n") {
		have[strings.TrimSpace(l)] = true
	}
	var missing []string
	for _, l := range strings.Split(want, "\n") {
		if t := strings.TrimSpace(l); t != "" && !have[t] {
			missing = append(missing, t)
		}
	}
	if len(missing) == 0 {
		return false, nil
	}
	out := string(existing)
	if out != "" && !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	out += strings.Join(missing, "\n") + "\n"
	return true, os.WriteFile(path, []byte(out), 0o644)
}
