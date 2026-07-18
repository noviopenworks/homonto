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

# [subagents.architect]
# source = "builtin:architect"
# scope = "project"
# Retune THIS agent without restating its tier: any field set in a per-tool
# block wins over the role's route, field by field.
# [subagents.architect.claude]
# effort = "xhigh"

# [plugins.claude.claude-hud]
# source = "claude-hud@official"       # name@marketplace
# [plugins.opencode.opencode-quota]
# source = "@slkiser/opencode-quota"   # npm package

# [settings.claude]
# model = "opus"

# A tool gains its four model routes (architectural/coding/review/trivial) as
# soon as a framework, command, or subagent targets it — declare all four for
# every such tool. The examples above target both tools, so both are shown
# here. An agent picks its route by the role it declares; model is required,
# the rest optional.
#
# Claude: model is an alias (opus/sonnet/haiku/fable) or a full id; variant
# brackets an ALIAS only (opus[1m] — 1m is the only one); effort is one of
# low|medium|high|xhigh|max.
# [models.claude.architectural]
# model = "opus"
# effort = "high"
# [models.claude.coding]
# model = "sonnet"
# effort = "medium"
# [models.claude.review]
# model = "opus"
# effort = "high"
# [models.claude.trivial]
# model = "haiku"
# effort = "low"
#
# OpenCode: the mirror image — variant is its own field taking whatever your
# provider defines, and there is no effort setting at all.
# [models.opencode.architectural]
# model = "anthropic/claude-opus-4-8"
# [models.opencode.coding]
# model = "anthropic/claude-sonnet-4-5"
# [models.opencode.review]
# model = "anthropic/claude-opus-4-8"
# [models.opencode.trivial]
# model = "anthropic/claude-haiku-4-5"
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
