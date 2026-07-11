package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/noviopenworks/homonto/internal/agentblob"
	"github.com/noviopenworks/homonto/internal/agentlock"
)

// TestAgentsAddPersistsBaseBlob: after `agents add`, the source content is
// retrievable from the blob store by its recorded install hash.
func TestAgentsAddPersistsBaseBlob(t *testing.T) {
	home := t.TempDir()
	body := "# Rev agent\nreview carefully\n"
	cfg, cfgDir := addWorkspace(t, copyAgentTOML, map[string]string{"rev": body})

	if out, err := runCmd(t, home, "", "agents", "add", "rev", "--config", cfg); err != nil {
		t.Fatalf("agents add: %v\n%s", err, out)
	}

	homontoDir := filepath.Join(cfgDir, ".homonto")
	hash := agentlock.HashContent([]byte(body))
	p := filepath.Join(homontoDir, "agents-blobs", hash)
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("expected base blob at %s: %v", p, err)
	}
	got, ok, err := agentblob.Get(homontoDir, hash)
	if err != nil || !ok {
		t.Fatalf("agentblob.Get(%s) = ok %v err %v", hash, ok, err)
	}
	if string(got) != body {
		t.Fatalf("blob content = %q, want %q", got, body)
	}
}

// TestAgentsUpdatePersistsNewBaseBlob: after `agents update` to a NEW source,
// the new source's content is retrievable from the blob store by its hash.
func TestAgentsUpdatePersistsNewBaseBlob(t *testing.T) {
	home := t.TempDir()
	old := "# Rev agent v1\n"
	cfg, cfgDir := addWorkspace(t, copyAgentTOML, map[string]string{"rev": old})

	if out, err := runCmd(t, home, "", "agents", "add", "rev", "--config", cfg); err != nil {
		t.Fatalf("agents add: %v\n%s", err, out)
	}

	neu := "# Rev agent v2 (new source)\n"
	srcPath := filepath.Join(cfgDir, "homonto", "agents", "rev.md")
	if err := os.WriteFile(srcPath, []byte(neu), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := runCmd(t, home, "", "agents", "update", "rev", "--config", cfg); err != nil {
		t.Fatalf("agents update: %v\n%s", err, out)
	}

	homontoDir := filepath.Join(cfgDir, ".homonto")
	hash := agentlock.HashContent([]byte(neu))
	got, ok, err := agentblob.Get(homontoDir, hash)
	if err != nil || !ok {
		t.Fatalf("new-source blob not persisted: ok %v err %v", ok, err)
	}
	if string(got) != neu {
		t.Fatalf("new blob content = %q, want %q", got, neu)
	}
}
