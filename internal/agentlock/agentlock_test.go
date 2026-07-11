package agentlock

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// TestLoadAbsentReturnsEmptyLock: Load on a dir with no agents-lock.json returns
// an empty, non-nil Lock and no error.
func TestLoadAbsentReturnsEmptyLock(t *testing.T) {
	dir := t.TempDir()
	l, err := Load(dir)
	if err != nil {
		t.Fatalf("Load absent: %v", err)
	}
	if l == nil || l.Agents == nil {
		t.Fatalf("Load absent must return an empty non-nil Lock, got %+v", l)
	}
	if len(l.Agents) != 0 {
		t.Fatalf("empty lock must have no agents, got %+v", l.Agents)
	}
}

// TestSaveThenLoadRoundTrips: a saved Lock reloads deep-equal.
func TestSaveThenLoadRoundTrips(t *testing.T) {
	dir := t.TempDir()
	want := &Lock{Agents: map[string]Agent{
		"rev": {
			Source:  "local:rev",
			Version: "1.0.0",
			Mode:    "copy",
			Targets: []string{"claude", "opencode"},
			Installed: map[string]Install{
				"claude":   {Path: "/home/u/.claude/agents/rev.md", Hash: "abc"},
				"opencode": {Path: "/home/u/.config/opencode/agent/rev.md", Hash: "abc"},
			},
		},
	}}
	if err := want.Save(dir); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("round-trip mismatch:\n got=%+v\nwant=%+v", got, want)
	}
}

// TestSaveIsDeterministic: two saves of the same Lock produce byte-identical
// files (sorted map keys, stable indentation) so the lockfile diffs cleanly.
func TestSaveIsDeterministic(t *testing.T) {
	l := &Lock{Agents: map[string]Agent{
		"b": {Source: "local:b", Mode: "link", Targets: []string{"claude"},
			Installed: map[string]Install{"claude": {Path: "/x/b.md", Hash: "h2"}}},
		"a": {Source: "local:a", Mode: "copy", Targets: []string{"opencode", "claude"},
			Installed: map[string]Install{"opencode": {Path: "/y/a.md", Hash: "h1"}, "claude": {Path: "/z/a.md", Hash: "h1"}}},
	}}
	d1, d2 := t.TempDir(), t.TempDir()
	if err := l.Save(d1); err != nil {
		t.Fatal(err)
	}
	if err := l.Save(d2); err != nil {
		t.Fatal(err)
	}
	b1, _ := os.ReadFile(filepath.Join(d1, "agents-lock.json"))
	b2, _ := os.ReadFile(filepath.Join(d2, "agents-lock.json"))
	if string(b1) != string(b2) {
		t.Fatalf("Save must be deterministic:\n%s\n---\n%s", b1, b2)
	}
}

// TestParseErrorSurfaces: a corrupt lockfile surfaces the parse error rather
// than being silently treated as empty.
func TestParseErrorSurfaces(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "agents-lock.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(dir); err == nil {
		t.Fatal("Load must surface a parse error for a corrupt lockfile")
	}
}

// TestHashContentStableAndDistinct: HashContent is deterministic for equal
// bytes and differs for different bytes.
func TestHashContentStableAndDistinct(t *testing.T) {
	a := HashContent([]byte("hello"))
	if a == "" {
		t.Fatal("HashContent must return a non-empty hex digest")
	}
	if a != HashContent([]byte("hello")) {
		t.Fatal("HashContent must be stable for equal content")
	}
	if a == HashContent([]byte("world")) {
		t.Fatal("HashContent must differ for different content")
	}
}
