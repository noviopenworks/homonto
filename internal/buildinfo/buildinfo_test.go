package buildinfo

import "testing"

// A stamped version (release ldflags applied) is returned unchanged.
func TestResolve_KeepsStampedVersion(t *testing.T) {
	if got := Resolve("v0.1.0-rc.1", "0.1.0-dev"); got != "v0.1.0-rc.1" {
		t.Fatalf("Resolve(stamped) = %q, want v0.1.0-rc.1", got)
	}
}

// Under `go test` the main module version is "" or "(devel)", so an unstamped
// binary keeps the dev default rather than inventing a version. (The
// `go install <path>@<tag>` path, where Main.Version is the tag, is exercised by
// the release smoke test — it cannot be simulated in a unit test.)
func TestResolve_UnstampedKeepsDevUnderTest(t *testing.T) {
	if got := Resolve("0.1.0-dev", "0.1.0-dev"); got != "0.1.0-dev" {
		t.Fatalf("Resolve(unstamped) = %q, want 0.1.0-dev kept under go test", got)
	}
}
