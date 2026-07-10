package subagentpath

import (
	"os"
	"strings"
	"testing"
)

// frontmatter returns the text between the first two "---" fence lines.
func frontmatter(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	s := string(b)
	if !strings.HasPrefix(s, "---\n") {
		t.Fatalf("%s: no leading frontmatter fence", path)
	}
	rest := s[len("---\n"):]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		t.Fatalf("%s: unterminated frontmatter", path)
	}
	return rest[:end]
}

func TestSharedFrontmatterContract(t *testing.T) {
	for _, path := range []string{
		"testdata/claude/agents/sample.md",
		"testdata/opencode/agent/sample.md",
	} {
		fm := frontmatter(t, path)
		for _, want := range []string{"name:", "description:", "mode: subagent"} {
			if !strings.Contains(fm, want) {
				t.Errorf("%s frontmatter missing %q", path, want)
			}
		}
		for _, forbidden := range []string{"model:", "tools:"} {
			for _, line := range strings.Split(fm, "\n") {
				if strings.HasPrefix(strings.TrimSpace(line), forbidden) {
					t.Errorf("%s frontmatter must omit %q (hard-conflicting field)", path, forbidden)
				}
			}
		}
	}
}
