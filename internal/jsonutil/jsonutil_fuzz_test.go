package jsonutil

import (
	"encoding/json"
	"testing"
	"unicode/utf8"
)

// FuzzEscapePathRoundTrip asserts the security-relevant invariant behind
// EscapePath: a config-supplied name, escaped and used as an sjson path segment,
// addresses the LITERAL key — never nested objects or a silently-vanished write.
// If this ever fails, a plugin/marketplace/setting name with a special byte
// could corrupt or silently drop a managed key.
func FuzzEscapePathRoundTrip(f *testing.F) {
	for _, s := range []string{"model", "a.b", "x|y", "p#q", "@root", "back\\slash", "a*b?c", "opencode-dark", "claude-hud@official"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, segment string) {
		// Empty names are rejected by the config model; invalid UTF-8 is not a
		// realistic config key and only adds encoder noise.
		if segment == "" || !utf8.ValidString(segment) {
			return
		}
		path := EscapePath(segment)
		doc, err := SetJSON([]byte("{}"), path, "v")
		if err != nil {
			return // sjson rejected the path; not a round-trip claim
		}
		var m map[string]any
		if err := json.Unmarshal(doc, &m); err != nil {
			t.Fatalf("SetJSON produced invalid JSON for segment %q: %v\n%s", segment, err, doc)
		}
		got, ok := m[segment]
		if !ok {
			t.Fatalf("escaped path did not set the LITERAL key %q; doc=%s", segment, doc)
		}
		if got != "v" {
			t.Fatalf("literal key %q set to %v, want \"v\"; doc=%s", segment, got, doc)
		}
		// The write must be surgical: exactly the one key exists.
		if len(m) != 1 {
			t.Fatalf("segment %q produced %d top-level keys, want 1: %v", segment, len(m), m)
		}
	})
}
