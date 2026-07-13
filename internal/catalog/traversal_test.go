package catalog

import (
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"
)

// TestIndexFramework_RejectsTraversalResourceNames is a security regression: a
// framework-manifest resource *name* (map key) that is not a plain path component
// must be rejected at index time. Otherwise the name flows into
// filepath.Join(root, name) + os.RemoveAll / os.Rename during materialization and
// escapes the managed root — an arbitrary directory-delete + file-write vector
// reachable through a pinned-remote or shared local framework, defeating the
// remote-source trust boundary.
func TestIndexFramework_RejectsTraversalResourceNames(t *testing.T) {
	cases := []struct {
		kind string
		toml string
		file string
	}{
		{
			kind: "skill",
			toml: "name = \"evil\"\nversion = \"0.1.0\"\n[skills]\n\"../../../../escape\" = \"skills/payload\"\n",
			file: "skills/payload/SKILL.md",
		},
		{
			kind: "command",
			toml: "name = \"evil\"\nversion = \"0.1.0\"\n[commands]\n\"../../escape\" = \"commands/payload.md\"\n",
			file: "commands/payload.md",
		},
		{
			kind: "subagent",
			toml: "name = \"evil\"\nversion = \"0.1.0\"\n[subagents]\n\"../../escape\" = \"subagents/payload.md\"\n",
			file: "subagents/payload.md",
		},
	}
	for _, tc := range cases {
		t.Run(tc.kind, func(t *testing.T) {
			evil := fstest.MapFS{
				"framework.toml": {Data: []byte(tc.toml)},
				tc.file:          {Data: []byte("payload")},
			}
			_, err := LoadWithLocal(baseFS(), map[string]fs.FS{"evil": evil})
			if err == nil {
				t.Fatalf("%s: traversal resource name was accepted; want rejection", tc.kind)
			}
			if !strings.Contains(err.Error(), "path traversal") {
				t.Fatalf("%s: error = %v; want a path-traversal rejection", tc.kind, err)
			}
		})
	}
}

// TestIndexFramework_RejectsSeparatorAndDotNames covers the other non-plain names
// the guard must reject (bare separators, dot, dot-dot, empty).
func TestIndexFramework_RejectsSeparatorAndDotNames(t *testing.T) {
	// Single-quoted TOML keys are literal (no escape processing), so a backslash
	// reaches the validator intact rather than becoming a TOML escape.
	for _, bad := range []string{"a/b", "..", ".", `a\b`} {
		toml := "name = \"evil\"\nversion = \"0.1.0\"\n[skills]\n'" + bad + "' = \"skills/payload\"\n"
		evil := fstest.MapFS{
			"framework.toml":          {Data: []byte(toml)},
			"skills/payload/SKILL.md": {Data: []byte("payload")},
		}
		if _, err := LoadWithLocal(baseFS(), map[string]fs.FS{"evil": evil}); err == nil {
			t.Errorf("skill name %q was accepted; want rejection", bad)
		}
	}
}
