package engine

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/noviopenworks/homonto/internal/remote"
)

// buildSubagentTarGz writes a tar.gz containing <name>.md and returns its path.
func buildSubagentTarGz(t *testing.T, dir, name, body string) string {
	t.Helper()
	var raw bytes.Buffer
	tw := tar.NewWriter(&raw)
	hdr := &tar.Header{Name: name + ".md", Mode: 0o644, Size: int64(len(body)), Typeflag: tar.TypeReg}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write([]byte(body)); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	var gzbuf bytes.Buffer
	zw := gzip.NewWriter(&gzbuf)
	zw.Write(raw.Bytes())
	zw.Close()
	p := filepath.Join(dir, name+".tar.gz")
	if err := os.WriteFile(p, gzbuf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func pinFor(t *testing.T, tarPath string) remote.Digest {
	t.Helper()
	src, err := remote.ParseRemoteSource("remote:file://" + tarPath)
	if err != nil {
		t.Fatal(err)
	}
	tree, _, err := remote.Fetch(context.Background(), src, remote.DefaultLimits)
	if err != nil {
		t.Fatal(err)
	}
	return remote.CanonicalDigest(tree)
}

const remoteModels = "[models.claude.architectural]\nmodel=\"o\"\nvariant=\"m\"\n[models.claude.coding]\nmodel=\"o\"\nvariant=\"m\"\n[models.claude.trivial]\nmodel=\"o\"\nvariant=\"m\"\n"

func TestRemoteSubagentEndToEnd(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	fixtures := t.TempDir()
	tarPath := buildSubagentTarGz(t, fixtures, "reviewer", "# remote reviewer agent")
	pin := pinFor(t, tarPath)

	cfg := "[subagents.reviewer]\nsource=\"remote:file://" + tarPath + "\"\ndigest=\"" + pin.String() + "\"\nscope=\"project\"\ntargets=[\"claude\"]\n" + remoteModels
	cfgPath := filepath.Join(repo, "homonto.toml")
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}

	e, err := Build(cfgPath, home, filepath.Join(repo, "content"))
	if err != nil {
		t.Fatal(err)
	}
	sets, err := e.Plan()
	if err != nil {
		t.Fatal(err)
	}
	if err := e.Apply(sets); err != nil {
		t.Fatalf("apply: %v", err)
	}

	// Remote content materialized at the deterministic remote root.
	contentFile := filepath.Join(e.RemoteRoot, "subagents", "reviewer.md")
	got, err := os.ReadFile(contentFile)
	if err != nil {
		t.Fatalf("remote content not materialized: %v", err)
	}
	if string(got) != "# remote reviewer agent" {
		t.Fatalf("materialized content mismatch: %q", got)
	}

	// Lock records provenance; cache holds the pin.
	lock, err := remote.LoadLock(filepath.Join(repo, ".homonto", "remote.lock.json"))
	if err != nil {
		t.Fatal(err)
	}
	if entry, ok := lock.Get("subagent", "reviewer"); !ok || entry.Digest != pin.String() {
		t.Fatalf("lock missing/incorrect entry: %+v ok=%v", entry, ok)
	}
	cache := &remote.Cache{Root: filepath.Join(repo, ".homonto", "cache", "remote")}
	if !cache.Has(pin) {
		t.Fatal("verified content should be cached")
	}

	// Second apply is idempotent (no error, content still present).
	sets2, err := e.Plan()
	if err != nil {
		t.Fatal(err)
	}
	if err := e.Apply(sets2); err != nil {
		t.Fatalf("second apply: %v", err)
	}

	// De-declare → prune: the remote content and lock entry are gone.
	if err := os.WriteFile(cfgPath, []byte(remoteModels), 0o644); err != nil {
		t.Fatal(err)
	}
	e2, err := Build(cfgPath, home, filepath.Join(repo, "content"))
	if err != nil {
		t.Fatal(err)
	}
	sets3, err := e2.Plan()
	if err != nil {
		t.Fatal(err)
	}
	if err := e2.Apply(sets3); err != nil {
		t.Fatalf("prune apply: %v", err)
	}
	if _, err := os.Stat(contentFile); !os.IsNotExist(err) {
		t.Fatal("de-declared remote content should be pruned")
	}
	lock2, _ := remote.LoadLock(filepath.Join(repo, ".homonto", "remote.lock.json"))
	if _, ok := lock2.Get("subagent", "reviewer"); ok {
		t.Fatal("de-declared lock entry should be dropped")
	}
}

func TestRemoteSubagentPinMismatchAbortsApply(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	fixtures := t.TempDir()
	tarPath := buildSubagentTarGz(t, fixtures, "reviewer", "# real content")

	// Declare a wrong pin: apply must fail closed and write no remote content.
	wrong := "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	cfg := "[subagents.reviewer]\nsource=\"remote:file://" + tarPath + "\"\ndigest=\"" + wrong + "\"\nscope=\"project\"\ntargets=[\"claude\"]\n" + remoteModels
	cfgPath := filepath.Join(repo, "homonto.toml")
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	e, err := Build(cfgPath, home, filepath.Join(repo, "content"))
	if err != nil {
		t.Fatal(err)
	}
	sets, err := e.Plan()
	if err != nil {
		t.Fatal(err)
	}
	if err := e.Apply(sets); err == nil {
		t.Fatal("apply must fail closed on a pin mismatch")
	}
	if _, err := os.Stat(filepath.Join(e.RemoteRoot, "subagents", "reviewer.md")); !os.IsNotExist(err) {
		t.Fatal("no remote content should be written on a pin mismatch")
	}
}
