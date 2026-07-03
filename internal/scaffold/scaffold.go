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

# [skills]
# own = ["graphify"]

# [plugins]
# claude = ["claude-hud@official"]
# opencode = ["@slkiser/opencode-quota"]

# [settings.claude]
# model = "opus"
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
	keep := filepath.Join(dir, "content", "skills", ".gitkeep")
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
