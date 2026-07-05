package jsonutil

import (
	"strings"
	"testing"

	"github.com/tidwall/gjson"
)

func TestSetJSONPreservesUnmanaged(t *testing.T) {
	in := []byte(`{"keep":1,"mcpServers":{"old":{"command":["x"]}}}`)
	out, err := SetJSON(in, "mcpServers.brave", map[string]any{"command": []string{"npx"}})
	if err != nil {
		t.Fatal(err)
	}
	if gjson.GetBytes(out, "keep").Int() != 1 {
		t.Fatal("unmanaged key lost")
	}
	if gjson.GetBytes(out, "mcpServers.old.command.0").String() != "x" {
		t.Fatal("sibling lost")
	}
	if gjson.GetBytes(out, "mcpServers.brave.command.0").String() != "npx" {
		t.Fatal("new value missing")
	}
}

func TestStandardizeStripsComments(t *testing.T) {
	out, err := Standardize([]byte("{// hi\n\"a\":1,}"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), "//") || gjson.GetBytes(out, "a").Int() != 1 {
		t.Fatalf("standardize wrong: %s", out)
	}
}

func TestStandardizeEmpty(t *testing.T) {
	out, _ := Standardize(nil)
	if strings.TrimSpace(string(out)) != "{}" {
		t.Fatalf("empty -> %q", out)
	}
}

func TestEnsureArrayElemIdempotent(t *testing.T) {
	out, _ := EnsureArrayElem([]byte(`{"plugin":["a"]}`), "plugin", "b")
	out, _ = EnsureArrayElem(out, "plugin", "b") // second time no-op
	if gjson.GetBytes(out, "plugin.#").Int() != 2 {
		t.Fatalf("array = %s", out)
	}
}

func TestCanonicalIsOrderIndependent(t *testing.T) {
	a := Canonical(`{"b":2,"a":1}`)
	b := Canonical(`{"a":1,"b":2}`)
	if a != b {
		t.Fatalf("canonical differs by key order: %q vs %q", a, b)
	}
}

func TestCanonicalPassesThroughInvalid(t *testing.T) {
	if got := Canonical("${pass:x}"); got != "${pass:x}" {
		t.Fatalf("invalid JSON should pass through: %q", got)
	}
}

func TestGetJSON(t *testing.T) {
	raw, ok := GetJSON([]byte(`{"a":{"b":2}}`), "a.b")
	if !ok || strings.TrimSpace(raw) != "2" {
		t.Fatalf("GetJSON a.b = %q ok=%v", raw, ok)
	}
	if _, ok := GetJSON([]byte(`{}`), "missing"); ok {
		t.Fatal("missing path reported present")
	}
}

// TestEscapePathRoundTrips: every gjson/sjson special character must write
// and read back the literal key. Unescaped, "a.b" nests and "a@b" writes
// nothing at all.
func TestEscapePathRoundTrips(t *testing.T) {
	for _, name := range []string{"corp.internal", "foo@bar.dots", "a|b", "a#b", "a*b?c", `a\b`, "plain"} {
		p := "root." + EscapePath(name)
		out, err := SetJSON(nil, p, true)
		if err != nil {
			t.Fatalf("SetJSON %q: %v", name, err)
		}
		var literal bool
		gjson.GetBytes(out, "root").ForEach(func(k, v gjson.Result) bool {
			if k.String() == name {
				literal = true
			}
			return true
		})
		if !literal {
			t.Fatalf("EscapePath(%q): literal key missing: %s", name, out)
		}
		if raw, ok := GetJSON(out, p); !ok || strings.TrimSpace(raw) != "true" {
			t.Fatalf("EscapePath(%q): escaped read = %q ok=%v", name, raw, ok)
		}
		out, err = DeleteJSON(out, p)
		if err != nil {
			t.Fatalf("DeleteJSON %q: %v", name, err)
		}
		if _, ok := GetJSON(out, p); ok {
			t.Fatalf("EscapePath(%q): escaped delete left the key: %s", name, out)
		}
	}
}

// TestObjectRoot: only an object root is accepted; the error names the kind.
func TestObjectRoot(t *testing.T) {
	if err := ObjectRoot([]byte(" {\"a\":1} ")); err != nil {
		t.Fatalf("object root rejected: %v", err)
	}
	for doc, kind := range map[string]string{
		`[]`: "array", `"s"`: "string", `true`: "boolean", `null`: "null", `42`: "number",
	} {
		err := ObjectRoot([]byte(doc))
		if err == nil || !strings.Contains(err.Error(), kind) {
			t.Fatalf("ObjectRoot(%s) = %v; want error naming %q", doc, err, kind)
		}
	}
}
