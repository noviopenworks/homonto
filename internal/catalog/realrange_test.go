package catalog

import "testing"

// TestNew_RealCatalogWithRangedDepsLoads proves the embedded production catalog
// (where comet declares superpowers@>=0.1.0, openspec@>=0.1.0) loads clean.
func TestNew_RealCatalogWithRangedDepsLoads(t *testing.T) {
	c, err := New()
	if err != nil {
		t.Fatalf("real catalog with ranged deps failed to load: %v", err)
	}
	fw, ok := c.Framework("comet")
	if !ok {
		t.Fatal("comet framework missing")
	}
	// Dependencies key on name (constraint stripped) for the graph.
	if len(fw.Dependencies) != 2 {
		t.Errorf("comet deps = %v, want [superpowers openspec] names", fw.Dependencies)
	}
	if fw.DependencyConstraints["superpowers"] != ">=0.1.0" {
		t.Errorf("comet superpowers constraint = %q, want >=0.1.0", fw.DependencyConstraints["superpowers"])
	}
}
