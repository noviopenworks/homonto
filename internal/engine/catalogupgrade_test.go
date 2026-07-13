package engine

import (
	"strings"
	"testing"
)

func TestCatalogUpgradeFinding(t *testing.T) {
	if f, pending := catalogUpgradeFinding("1", "2"); !pending || !strings.Contains(f, "1") || !strings.Contains(f, "2") {
		t.Errorf("differ: got (%q,%v), want a finding naming both versions", f, pending)
	}
	if f, pending := catalogUpgradeFinding("", "1"); !pending {
		t.Errorf("unrecorded vs embedded: got (%q,%v), want pending", f, pending)
	}
	if _, pending := catalogUpgradeFinding("2", "2"); pending {
		t.Error("equal versions should not be a finding")
	}
}
