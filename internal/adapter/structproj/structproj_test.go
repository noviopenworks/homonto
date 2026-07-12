package structproj

import (
	"testing"

	"github.com/noviopenworks/homonto/internal/adapter"
	"github.com/noviopenworks/homonto/internal/secret"
	"github.com/noviopenworks/homonto/internal/state"
	"github.com/noviopenworks/homonto/internal/tomlutil"
)

// tomlTestCodec adapts the tomlutil package funcs to the Codec interface.
type tomlTestCodec struct{}

func (tomlTestCodec) EnsureRoot(doc []byte) ([]byte, error) { return tomlutil.EnsureRoot(doc) }
func (tomlTestCodec) Get(doc []byte, p string) (string, bool) {
	return tomlutil.Get(doc, p)
}
func (tomlTestCodec) Set(doc []byte, p, v string) ([]byte, error) { return tomlutil.Set(doc, p, v) }
func (tomlTestCodec) Delete(doc []byte, p string) ([]byte, error) { return tomlutil.Delete(doc, p) }
func (tomlTestCodec) Canonical(v string) string                   { return tomlutil.Canonical(v) }

func pathFor(key string) string { return "mcp_servers." + trimPrefix(key, "mcp.") }
func trimPrefix(s, p string) string {
	if len(s) >= len(p) && s[:len(p)] == p {
		return s[len(p):]
	}
	return s
}

const tool = "codex"
const prefix = "mcp."

func emptyState(t *testing.T) *state.State {
	t.Helper()
	st, err := state.Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return st
}

func TestProjectCreateThenNoop(t *testing.T) {
	codec := tomlTestCodec{}
	st := emptyState(t)
	desired := map[string]string{"mcp.demo": `{"command":["x"]}`}
	res := secret.NewResolver()

	// create
	changes := Project(tool, prefix, desired, []byte(""), st, codec, pathFor)
	if len(changes) != 1 || changes[0].Action != "create" {
		t.Fatalf("want one create, got %+v", changes)
	}
	doc, changed, err := Apply(tool, prefix, changes, []byte(""), codec, res, st, pathFor)
	if err != nil || !changed {
		t.Fatalf("apply: changed=%v err=%v", changed, err)
	}
	if v, ok := tomlutil.Get(doc, "mcp_servers.demo.command"); !ok || v != `["x"]` {
		t.Fatalf("projected value wrong: %q ok=%v", v, ok)
	}

	// second plan → noop
	changes2 := Project(tool, prefix, desired, doc, st, codec, pathFor)
	if len(changes2) != 1 || changes2[0].Action != "noop" {
		t.Fatalf("want noop, got %+v", changes2)
	}
}

func TestProjectUpdateAndDelete(t *testing.T) {
	codec := tomlTestCodec{}
	st := emptyState(t)
	res := secret.NewResolver()
	doc, _, _ := Apply(tool, prefix, Project(tool, prefix, map[string]string{"mcp.demo": `{"command":["x"]}`}, []byte(""), st, codec, pathFor), []byte(""), codec, res, st, pathFor)

	// update
	upd := map[string]string{"mcp.demo": `{"command":["y"]}`}
	ch := Project(tool, prefix, upd, doc, st, codec, pathFor)
	if len(ch) != 1 || ch[0].Action != "update" {
		t.Fatalf("want update, got %+v", ch)
	}
	doc, _, _ = Apply(tool, prefix, ch, doc, codec, res, st, pathFor)
	if v, _ := tomlutil.Get(doc, "mcp_servers.demo.command"); v != `["y"]` {
		t.Fatalf("update not applied: %q", v)
	}

	// de-declare → delete
	del := Project(tool, prefix, map[string]string{}, doc, st, codec, pathFor)
	if len(del) != 1 || del[0].Action != "delete" {
		t.Fatalf("want delete, got %+v", del)
	}
	doc, _, _ = Apply(tool, prefix, del, doc, codec, res, st, pathFor)
	if _, ok := tomlutil.Get(doc, "mcp_servers.demo"); ok {
		t.Fatal("de-declared server should be pruned")
	}
}

func TestProjectAdoptsPreexisting(t *testing.T) {
	codec := tomlTestCodec{}
	st := emptyState(t)
	// disk already has the desired value but state has no record → adopt
	doc, _ := tomlutil.Set([]byte(""), "mcp_servers.demo.command", `["x"]`)
	changes := Project(tool, prefix, map[string]string{"mcp.demo": `{"command":["x"]}`}, doc, st, codec, pathFor)
	if len(changes) != 1 || changes[0].Action != "adopt" {
		t.Fatalf("want adopt, got %+v", changes)
	}
}

func TestProjectSecretRedaction(t *testing.T) {
	codec := tomlTestCodec{}
	st := emptyState(t)
	// a secret-bearing value that differs from disk must redact Old and never
	// appear in the change.
	doc, _ := tomlutil.Set([]byte(""), "mcp_servers.demo.env", `{"K":"old"}`)
	desired := map[string]string{"mcp.demo": `{"env":{"K":"${MISSING}"}}`}
	changes := Project(tool, prefix, desired, doc, st, codec, pathFor)
	if len(changes) != 1 {
		t.Fatalf("want one change, got %+v", changes)
	}
	if changes[0].Old != adapter.SecretRedaction {
		t.Fatalf("secret Old must be redacted, got %q", changes[0].Old)
	}
}
