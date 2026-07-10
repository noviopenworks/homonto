package ontocli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// treeSnapshot walks dir and records each file's relative path, size, and
// mod time, so callers can assert a run left the tree byte-for-byte
// untouched (no file created, modified, or removed).
func treeSnapshot(t *testing.T, dir string) map[string]string {
	t.Helper()
	snap := make(map[string]string)
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(dir, path)
		if relErr != nil {
			return relErr
		}
		snap[rel] = fmt.Sprintf("%s|%d", info.ModTime().String(), info.Size())
		return nil
	})
	if err != nil {
		t.Fatalf("failed to snapshot tree: %v", err)
	}
	return snap
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("failed to create dir for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}

func TestStatusCommand_ReportsValidAndInvalidChanges(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "docs", "changes", "alpha", "onto-state.yaml"), "change: alpha\nphase: build\n")
	writeFile(t, filepath.Join(dir, "docs", "changes", "beta", "onto-state.yaml"), "change: [unterminated\n  phase: build")

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"status", "--dir", dir})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "alpha: build") {
		t.Errorf("output = %q, want it to contain %q", got, "alpha: build")
	}
	if !strings.Contains(got, "beta:") || !strings.Contains(got, "invalid") {
		t.Errorf("output = %q, want a line for beta containing %q", got, "invalid")
	}
}

func TestStatusCommand_IsReadOnly(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "docs", "changes", "alpha", "onto-state.yaml"), "change: alpha\nphase: build\n")
	writeFile(t, filepath.Join(dir, "docs", "changes", "beta", "onto-state.yaml"), "change: [unterminated\n  phase: build")

	before := treeSnapshot(t, dir)

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"status", "--dir", dir})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	after := treeSnapshot(t, dir)

	if len(before) != len(after) {
		t.Fatalf("file count changed: before=%d after=%d", len(before), len(after))
	}
	for path, sig := range before {
		afterSig, ok := after[path]
		if !ok {
			t.Errorf("file %q was removed by status", path)
			continue
		}
		if sig != afterSig {
			t.Errorf("file %q was modified by status: before=%q after=%q", path, sig, afterSig)
		}
	}
	for path := range after {
		if _, ok := before[path]; !ok {
			t.Errorf("file %q was created by status", path)
		}
	}
}

func TestStatusCommand_SucceedsWithoutHomontoToml(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "docs", "changes", "alpha", "onto-state.yaml"), "change: alpha\nphase: open\n")

	if _, err := os.Stat(filepath.Join(dir, "homonto.toml")); !os.IsNotExist(err) {
		t.Fatalf("expected no homonto.toml in temp workspace, stat err = %v", err)
	}

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"status", "--dir", dir})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !strings.Contains(out.String(), "alpha: open") {
		t.Errorf("output = %q, want it to contain %q", out.String(), "alpha: open")
	}
}

func TestStatusCommand_SkipsArchivedChanges(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "docs", "changes", "alpha", "onto-state.yaml"), "change: alpha\nphase: build\n")
	writeFile(t, filepath.Join(dir, "docs", "changes", "archive", "old-change", "onto-state.yaml"), "change: old-change\nphase: close\n")

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"status", "--dir", dir})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if strings.Contains(out.String(), "old-change") {
		t.Errorf("output = %q, want archived change to be skipped", out.String())
	}
}
