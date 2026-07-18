package tocli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestInit_Idempotent verifies a second `to init` reports both directories
// as pre-existing ("exists") and writes nothing new.
func TestInit_Idempotent(t *testing.T) {
	dir := setUpGatedWorkspace(t)
	run(t, false, "init", "--dir", dir)
	secondOut := run(t, false, "init", "--dir", dir)
	for _, line := range []string{"exists ", tasksDir(dir), archiveDir(dir)} {
		if !strings.Contains(secondOut, line) {
			t.Errorf("second init output %q missing %q", secondOut, line)
		}
	}
}

// TestInit_JSONShape verifies the init JSON carries created vs exists lists.
func TestInit_JSONShape(t *testing.T) {
	dir := setUpGatedWorkspace(t)
	firstJSON := run(t, false, "init", "--json", "--dir", dir)
	for _, want := range []string{`"created":`, `"exists":`} {
		if !strings.Contains(firstJSON, want) {
			t.Errorf("first init JSON %q missing %s", firstJSON, want)
		}
	}
}

// TestInit_Gated verifies init refuses without the framework install gate.
func TestInit_Gated(t *testing.T) {
	dir := t.TempDir()
	if out := runErr(t, "init", "--dir", dir); !strings.Contains(out, "homonto init") {
		t.Errorf("init-without-gate error %q missing 'homonto init'", out)
	}
	if _, err := os.Stat(tasksDir(dir)); !os.IsNotExist(err) {
		t.Errorf("gated init created %s, stat err = %v", tasksDir(dir), err)
	}
}

// TestInit_DirCreationFailsIsError verifies a non-creatable parent surfaces a
// real error (defense-in-depth on the gate contract: no partial state).
func TestInit_DirCreationFailsIsError(t *testing.T) {
	dir := setUpGatedWorkspace(t)
	// Make tasksDir's parent read-only so MkdirAll on a child fails.
	roParent := filepath.Join(dir, "docs")
	if err := os.MkdirAll(roParent, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(roParent, 0o755) })
	if out := runErr(t, "init", "--dir", dir); !strings.Contains(out, "to init:") {
		t.Errorf("init (read-only) error %q missing 'to init:' prefix", out)
	}
}
