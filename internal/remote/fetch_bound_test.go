package remote

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestFetchGitEnforcesSizeBeforeCheckout exercises F27: a git fetch must enforce
// the size/file-count caps at or before checkout, so an oversized pinned repo is
// rejected before its content is fully written to disk. The guard error is
// distinct ("before checkout") from the post-walk cap error.
func TestFetchGitEnforcesSizeBeforeCheckout(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	repo := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	run("init", "--quiet")
	// A file comfortably larger than the tiny cap set below.
	if err := os.WriteFile(filepath.Join(repo, "agent.md"), []byte(strings.Repeat("x", 4096)), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "-A")
	run("commit", "--quiet", "-m", "init")
	revb, err := exec.Command("git", "-C", repo, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	rev := strings.TrimSpace(string(revb))

	src, err := ParseRemoteSource("remote:git+file://" + repo + "#" + rev)
	if err != nil {
		t.Fatal(err)
	}
	lim := Limits{MaxEntries: 10_000, MaxEntryBytes: 64 << 20, MaxTotalBytes: 64}
	_, _, err = Fetch(context.Background(), src, lim)
	if err == nil {
		t.Fatal("oversized git source must be rejected")
	}
	if !strings.Contains(err.Error(), "before checkout") {
		t.Fatalf("size cap must be enforced before checkout, got: %v", err)
	}
}
