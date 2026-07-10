package commandpath

import (
	"path/filepath"
	"testing"
)

func TestDir(t *testing.T) {
	home := filepath.Join("/home", "u")
	proj := filepath.Join("/work", "repo")
	cases := []struct {
		tool, scope string
		want        string
	}{
		{"claude", "user", filepath.Join(home, ".claude", "commands")},
		{"claude", "project", filepath.Join(proj, ".claude", "commands")},
		{"opencode", "user", filepath.Join(home, ".config", "opencode", "command")},
		{"opencode", "project", filepath.Join(proj, ".opencode", "command")},
		// Non-"project" scope (empty, unknown) is treated as user.
		{"claude", "", filepath.Join(home, ".claude", "commands")},
		{"opencode", "whatever", filepath.Join(home, ".config", "opencode", "command")},
		// Unknown tool returns "".
		{"nope", "user", ""},
	}
	for _, c := range cases {
		if got := Dir(c.tool, c.scope, home, proj); got != c.want {
			t.Errorf("Dir(%q,%q) = %q; want %q", c.tool, c.scope, got, c.want)
		}
	}
}
