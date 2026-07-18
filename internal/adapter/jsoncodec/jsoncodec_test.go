package jsoncodec

import "testing"

func TestCodec_RoundTrip(t *testing.T) {
	var c Codec

	// EnsureRoot normalizes empty -> {}
	root, err := c.EnsureRoot(nil)
	if err != nil || string(root) != "{}" {
		t.Fatalf("EnsureRoot(nil) = %q, %v; want {} ", root, err)
	}
	// EnsureRoot rejects a non-object root.
	if _, err := c.EnsureRoot([]byte("[1,2]")); err == nil {
		t.Errorf("EnsureRoot(array) should error")
	}

	// Set a nested value; Get returns it canonicalized; present=true.
	doc, err := c.Set(root, "a.b", `{"y":2,"x":1}`)
	if err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, ok, gerr := c.Get(doc, "a.b")
	if gerr != nil || !ok {
		t.Fatalf("Get after Set: not present (ok=%v, err=%v)", ok, gerr)
	}
	// canonical form sorts keys
	if got != `{"x":1,"y":2}` {
		t.Errorf("Get canonical = %q, want {\"x\":1,\"y\":2}", got)
	}

	// Canonical is key-order-independent.
	if c.Canonical(`{"y":2,"x":1}`) != c.Canonical(`{"x":1,"y":2}`) {
		t.Errorf("Canonical not order-independent")
	}

	// Delete removes it; Get -> not present.
	doc, err = c.Delete(doc, "a.b")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, ok, _ := c.Get(doc, "a.b"); ok {
		t.Errorf("Get after Delete: still present")
	}
}
