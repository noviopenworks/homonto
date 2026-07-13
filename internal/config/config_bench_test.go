package config

import (
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkLoad(b *testing.B) {
	dir := b.TempDir()
	p := filepath.Join(dir, "homonto.toml")
	body := `[mcps.demo]
command = ["true"]

[models.claude.architectural]
model = "anthropic/claude-opus-4-1"
effort = "high"
[models.claude.coding]
model = "anthropic/claude-sonnet-4-5"
effort = "normal"
[models.claude.trivial]
model = "anthropic/claude-haiku-4-5"
effort = "fast"
[models.opencode.architectural]
model = "anthropic/claude-opus-4-1"
effort = "high"
[models.opencode.coding]
model = "anthropic/claude-sonnet-4-5"
effort = "normal"
[models.opencode.trivial]
model = "anthropic/claude-haiku-4-5"
effort = "fast"

[skills.a]
source = "local:a"
scope = "user"
[skills.b]
source = "builtin:b"
scope = "project"
[subagents.rev]
source = "local:rev"
`
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := Load(p); err != nil {
			b.Fatal(err)
		}
	}
}
