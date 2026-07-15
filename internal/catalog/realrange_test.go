package catalog

import (
	"io/fs"
	"testing"

	embedded "github.com/noviopenworks/homonto/catalog"
)

// TestNew_CatalogShipsOnlyOnto pins the shipped-framework surface: the
// embedded catalog carries exactly the onto framework (plus loose,
// framework-agnostic skills/commands indexed separately). comet, openspec,
// and superpowers were removed deliberately — a framework reappearing here
// is a packaging regression, not a feature. (Ranged-dep and capability
// mechanics keep their fstest coverage in version_test.go and
// capabilities_test.go; no shipped framework exercises them anymore.)
func TestNew_CatalogShipsOnlyOnto(t *testing.T) {
	if _, err := New(); err != nil {
		t.Fatalf("embedded catalog failed to load: %v", err)
	}
	entries, err := fs.ReadDir(embedded.FS, "frameworks")
	if err != nil {
		t.Fatalf("reading embedded frameworks dir: %v", err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	if len(names) != 1 || names[0] != "onto" {
		t.Errorf("shipped frameworks = %v, want exactly [onto]", names)
	}
}
