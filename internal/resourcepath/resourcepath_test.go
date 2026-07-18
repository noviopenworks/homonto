package resourcepath

import "testing"

func TestDir(t *testing.T) {
	cases := []struct {
		name          string
		kind          Kind
		tool, scope   string
		home, project string
		want          string
	}{
		{"claude skill user", Skill, "claude", "user", "/h", "/p", "/h/.claude/skills"},
		{"claude skill project", Skill, "claude", "project", "/h", "/p", "/p/.claude/skills"},
		{"opencode skill user", Skill, "opencode", "user", "/h", "/p", "/h/.config/opencode/skills"},
		{"opencode skill project", Skill, "opencode", "project", "/h", "/p", "/p/.opencode/skills"},

		{"claude command user", Command, "claude", "user", "/h", "/p", "/h/.claude/commands"},
		{"claude command project", Command, "claude", "project", "/h", "/p", "/p/.claude/commands"},
		{"opencode command user", Command, "opencode", "user", "/h", "/p", "/h/.config/opencode/command"},
		{"opencode command project", Command, "opencode", "project", "/h", "/p", "/p/.opencode/command"},

		{"claude subagent user", Subagent, "claude", "user", "/h", "/p", "/h/.claude/agents"},
		{"claude subagent project", Subagent, "claude", "project", "/h", "/p", "/p/.claude/agents"},
		{"opencode subagent user", Subagent, "opencode", "user", "/h", "/p", "/h/.config/opencode/agent"},
		{"opencode subagent project", Subagent, "opencode", "project", "/h", "/p", "/p/.opencode/agent"},

		// unknown kind / tool
		{"unknown kind", Kind("nope"), "claude", "user", "/h", "/p", ""},
		{"unknown tool", Skill, "vscode", "user", "/h", "/p", ""},
		{"subagent unknown tool", Subagent, "vscode", "user", "/h", "/p", ""},

		// Any scope other than "project" is treated as "user".
		{"empty scope falls back to user", Skill, "claude", "", "/h", "/p", "/h/.claude/skills"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Dir(tc.kind, tc.tool, tc.scope, tc.home, tc.project)
			if got != tc.want {
				t.Fatalf("Dir(%s, %q, %q, %q, %q) = %q, want %q", tc.kind, tc.tool, tc.scope, tc.home, tc.project, got, tc.want)
			}
		})
	}
}

func TestOtherScope(t *testing.T) {
	cases := []struct{ in, want string }{
		{"project", "user"},
		{"user", "project"},
		{"", "project"},
		{"bogus", "project"},
	}
	for _, tc := range cases {
		if got := OtherScope(tc.in); got != tc.want {
			t.Errorf("OtherScope(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
