package remote

import (
	"os"
	"path/filepath"
	"testing"
)

// TestCachePutRehashesRaceWinner exercises F26: when Put finds the cache
// directory already populated (a concurrent writer / rename-race winner), it
// must re-hash that directory before accepting it. A corrupt winner must be
// rejected, not trusted merely because it landed first.
func TestCachePutRehashesRaceWinner(t *testing.T) {
	c := &Cache{Root: t.TempDir()}
	good := sampleTree()
	d := CanonicalDigest(good)

	// Simulate a race winner: the destination is already present under this
	// digest but holds CORRUPT content (missing ref/n.md, tampered agent.md).
	dir := c.Dir(d)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "agent.md"), []byte("CORRUPT"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := c.Put(d, good); err == nil {
		t.Fatal("Put must re-hash and reject a corrupt cache-race winner")
	}
}
