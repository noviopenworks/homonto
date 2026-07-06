package skillpath

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
		{"claude", "user", filepath.Join(home, ".claude", "skills")},
		{"claude", "project", filepath.Join(proj, ".claude", "skills")},
		{"opencode", "user", filepath.Join(home, ".config", "opencode", "skills")},
		{"opencode", "project", filepath.Join(proj, ".opencode", "skills")},
		// A non-"project" scope (empty, unknown) is treated as user — defense in
		// depth; config.Load already normalizes, but the helper must not misplace.
		{"claude", "", filepath.Join(home, ".claude", "skills")},
		{"opencode", "whatever", filepath.Join(home, ".config", "opencode", "skills")},
	}
	for _, c := range cases {
		if got := Dir(c.tool, c.scope, home, proj); got != c.want {
			t.Errorf("Dir(%q,%q) = %q; want %q", c.tool, c.scope, got, c.want)
		}
	}
}

func TestOther(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"user", "project"},
		{"project", "user"},
		{"", "project"}, // non-project is user-like, so its opposite is project
	} {
		if got := Other(c.in); got != c.want {
			t.Errorf("Other(%q) = %q; want %q", c.in, got, c.want)
		}
	}
}
