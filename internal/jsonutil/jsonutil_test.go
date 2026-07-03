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

func TestGetJSON(t *testing.T) {
	raw, ok := GetJSON([]byte(`{"a":{"b":2}}`), "a.b")
	if !ok || strings.TrimSpace(raw) != "2" {
		t.Fatalf("GetJSON a.b = %q ok=%v", raw, ok)
	}
	if _, ok := GetJSON([]byte(`{}`), "missing"); ok {
		t.Fatal("missing path reported present")
	}
}
