package tomlutil

import "testing"

func TestSetGetRoundTrip(t *testing.T) {
	doc, err := Set([]byte(""), "mcp_servers.demo.command", `["codex-mcp"]`)
	if err != nil {
		t.Fatal(err)
	}
	got, ok, err := Get(doc, "mcp_servers.demo.command")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !ok {
		t.Fatal("key should be present after set")
	}
	if got != `["codex-mcp"]` {
		t.Fatalf("get = %q, want %q", got, `["codex-mcp"]`)
	}
}

func TestSetPreservesUnmanaged(t *testing.T) {
	base := "model = \"o3\"\n\n[mcp_servers.user_owned]\ncommand = [\"x\"]\n"
	doc, err := Set([]byte(base), "mcp_servers.demo.command", `["codex-mcp","serve"]`)
	if err != nil {
		t.Fatal(err)
	}
	if v, ok, err := Get(doc, "model"); err != nil || !ok || v != `"o3"` {
		t.Fatalf("unmanaged top-level key lost: %q ok=%v err=%v", v, ok, err)
	}
	if v, ok, err := Get(doc, "mcp_servers.user_owned.command"); err != nil || !ok || v != `["x"]` {
		t.Fatalf("unmanaged server table lost: %q ok=%v err=%v", v, ok, err)
	}
	if v, ok, err := Get(doc, "mcp_servers.demo.command"); err != nil || !ok || v != `["codex-mcp","serve"]` {
		t.Fatalf("managed value wrong: %q ok=%v err=%v", v, ok, err)
	}
}

func TestDeletePrunesEmptyParents(t *testing.T) {
	doc, err := Set([]byte("model = \"o3\"\n"), "mcp_servers.demo.command", `["x"]`)
	if err != nil {
		t.Fatal(err)
	}
	doc, err = Delete(doc, "mcp_servers.demo")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok, err := Get(doc, "mcp_servers.demo"); err != nil || ok {
		t.Fatal("deleted table should be gone")
	}
	// The unmanaged sibling key survives.
	if v, ok, err := Get(doc, "model"); err != nil || !ok || v != `"o3"` {
		t.Fatalf("unmanaged key lost after delete: %q ok=%v err=%v", v, ok, err)
	}
}

func TestGetAbsent(t *testing.T) {
	if _, ok, err := Get([]byte("model=\"o3\"\n"), "mcp_servers.none.command"); err != nil || ok {
		t.Fatal("absent key must report not present (and no parse error on a valid doc)")
	}
}

// TestGetSurfacesParseError guards the H5 fix: a corrupted document must
// return an error rather than silently collapse to ok=false, so a caller
// doing plan diffing cannot mistake a broken file for a deleted key and emit
// a destructive plan.
func TestGetSurfacesParseError(t *testing.T) {
	corrupt := []byte("not = valid = toml = =")
	if _, _, err := Get(corrupt, "mcp_servers.demo.command"); err == nil {
		t.Fatal("Get on corrupt TOML must return a parse error")
	}
}

func TestCanonicalStable(t *testing.T) {
	a := Canonical(`{"b":1,"a":2}`)
	b := Canonical(`{"a":2,"b":1}`)
	if a != b {
		t.Fatalf("canonical form must be key-order independent: %q vs %q", a, b)
	}
}

func TestEnsureRootEmpty(t *testing.T) {
	doc, err := EnsureRoot([]byte("   \n"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Set(doc, "mcp_servers.x.command", `["y"]`); err != nil {
		t.Fatalf("set on ensured-empty doc: %v", err)
	}
}

func TestGetNestedEnv(t *testing.T) {
	doc, err := Set([]byte(""), "mcp_servers.x.env", `{"KEY":"val"}`)
	if err != nil {
		t.Fatal(err)
	}
	if v, ok, err := Get(doc, "mcp_servers.x.env"); err != nil || !ok || v != `{"KEY":"val"}` {
		t.Fatalf("nested env round-trip: %q ok=%v err=%v", v, ok, err)
	}
}
