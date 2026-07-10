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
		{"claude", "", "/home/u/.claude/agents"}, // unknown scope -> user
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
