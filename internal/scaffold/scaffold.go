package scaffold

import (
	"os"
	"path/filepath"
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

# [plugins.claude.claude-hud]
# source = "claude-hud@official"       # name@marketplace
# [plugins.opencode.opencode-quota]
# source = "@slkiser/opencode-quota"   # npm package

# [settings.claude]
# model = "opus"

# A tool gains its three model routes (architectural/coding/trivial) as soon as a
# framework, command, or subagent targets it — declare all three for every such
# tool. The examples above target both tools, so both are shown here.
# [models.claude.architectural]
# model = "opus"
# variant = "max"
# [models.claude.coding]
# model = "sonnet"
# effort = "normal"
# [models.claude.trivial]
# model = "haiku"
# effort = "fast"
# [models.opencode.architectural]
# model = "anthropic/claude-opus-4-8"
# variant = "max"
# [models.opencode.coding]
# model = "anthropic/claude-sonnet-4-5"
# effort = "normal"
# [models.opencode.trivial]
# model = "anthropic/claude-haiku-4-5"
# effort = "fast"
`,
	".gitignore":   "/.homonto/\n.env\n",
	".env.example": "# Document non-pass secrets here, then copy to .env (gitignored).\n# BRAVE_API_KEY=\n",
}

// Init scaffolds a homonto repo, skipping files that already exist.
func Init(dir string) ([]string, error) {
	var created []string
	for name, body := range files {
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); err == nil {
			continue
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			return created, err
		}
		created = append(created, p)
	}
	keep := filepath.Join(dir, "homonto", "skills", ".gitkeep")
	if _, err := os.Stat(keep); err != nil {
		if err := os.MkdirAll(filepath.Dir(keep), 0o755); err != nil {
			return created, err
		}
		if err := os.WriteFile(keep, nil, 0o644); err != nil {
			return created, err
		}
		created = append(created, keep)
	}
	return created, nil
}
