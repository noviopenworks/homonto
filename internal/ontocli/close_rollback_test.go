package ontocli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/noviopenworks/homonto/internal/ontostate"
)

// TestCloseCommand_ArchiveMoveFailureRollsBackArchived proves that when the
// archive move fails after `archived: true` was written, onto close rolls the
// flag back so the change is left fully un-archived (spec: archives NOTHING on
// failure).
func TestCloseCommand_ArchiveMoveFailureRollsBackArchived(t *testing.T) {
	dir := prepWorkspace(t)
	seedClose(t, dir, "demo", nil)
	// Force the archive move to fail deterministically: make the archive parent
	// a regular file so os.MkdirAll(docs/changes/archive) fails during close.
	// Committing it keeps the worktree clean so the dirty-worktree gate passes.
	archiveParent := filepath.Join(dir, "docs", "changes", "archive")
	if err := os.WriteFile(archiveParent, []byte("blocker"), 0o644); err != nil {
		t.Fatal(err)
	}
	commitAll(t, dir, "seed change + archive blocker")

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"close", "demo", "--dir", dir})
	if err := cmd.Execute(); err == nil {
		t.Fatal("execute() = nil, want error (archive move must fail)")
	}

	// The change directory must remain at its original path.
	if _, err := os.Stat(filepath.Join(dir, "docs", "changes", "demo")); err != nil {
		t.Errorf("change dir should remain in place: %v", err)
	}
	// The archived flag must have been rolled back to false.
	st, err := ontostate.Load(filepath.Join(dir, "docs", "changes", "demo", "onto-state.yaml"))
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if st.Archived {
		t.Errorf("st.Archived = true after a failed move, want false (rolled back)")
	}
}
