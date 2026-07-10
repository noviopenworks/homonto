package ontocli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGate_MissingHomontoToml covers gate order step 1: no homonto.toml at
// all means the workspace was never initialized by homonto, so the error
// must point the user at `homonto init`.
func TestGate_MissingHomontoToml(t *testing.T) {
	dir := t.TempDir()

	err := gate(dir)
	if err == nil {
		t.Fatal("gate() = nil, want error")
	}
	if !strings.Contains(err.Error(), "homonto init") {
		t.Errorf("gate() error = %q, want it to mention %q", err.Error(), "homonto init")
	}

	assertNoDocsFiles(t, dir)
}

// TestGate_MissingFrameworksOnto covers gate order step 2: homonto.toml
// exists but declares no [frameworks.onto] table, so the error must tell
// the user to declare it and run `homonto apply`.
func TestGate_MissingFrameworksOnto(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "homonto.toml"), "[models.claude.architectural]\nmodel=\"opus\"\n")

	err := gate(dir)
	if err == nil {
		t.Fatal("gate() = nil, want error")
	}
	if !strings.Contains(err.Error(), "[frameworks.onto]") {
		t.Errorf("gate() error = %q, want it to mention %q", err.Error(), "[frameworks.onto]")
	}
	if !strings.Contains(err.Error(), "homonto apply") {
		t.Errorf("gate() error = %q, want it to mention %q", err.Error(), "homonto apply")
	}

	assertNoDocsFiles(t, dir)
}

// TestGate_MissingCatalog covers gate order step 3: [frameworks.onto] is
// declared but has not been applied yet, so the onto skill catalog
// directory doesn't exist. The error must tell the user to run
// `homonto apply`.
func TestGate_MissingCatalog(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "homonto.toml"), "[frameworks.onto]\nsource=\"builtin:onto\"\nscope=\"project\"\n")

	err := gate(dir)
	if err == nil {
		t.Fatal("gate() = nil, want error")
	}
	if !strings.Contains(err.Error(), "homonto apply") {
		t.Errorf("gate() error = %q, want it to mention %q", err.Error(), "homonto apply")
	}

	assertNoDocsFiles(t, dir)
}

// TestGate_AllPresent covers gate order step 4: homonto.toml declares
// [frameworks.onto] and the catalog has been applied, so the gate must
// pass.
func TestGate_AllPresent(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "homonto.toml"), "[frameworks.onto]\nsource=\"builtin:onto\"\nscope=\"project\"\n")
	if err := os.MkdirAll(filepath.Join(dir, ".homonto", "catalog", "skills", "onto"), 0o755); err != nil {
		t.Fatalf("failed to create catalog dir: %v", err)
	}

	if err := gate(dir); err != nil {
		t.Errorf("gate() = %v, want nil", err)
	}
}

// assertNoDocsFiles verifies the gate performed no writes: it is a
// read-only check and must never scaffold docs/ files itself.
func assertNoDocsFiles(t *testing.T, dir string) {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(dir, "docs", "*"))
	if err != nil {
		t.Fatalf("glob docs: %v", err)
	}
	if len(matches) != 0 {
		t.Errorf("gate() created docs/ entries: %v", matches)
	}
}
