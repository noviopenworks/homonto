package catalog

import (
	"strings"
	"testing"
	"testing/fstest"
)

func capFS(providerCaps, requirerCaps string) fstest.MapFS {
	// provider framework declares [provides].capabilities; requirer declares
	// [dependencies].capabilities. Empty string omits the block.
	pv := ""
	if providerCaps != "" {
		pv = "[provides]\ncapabilities = [" + providerCaps + "]\n"
	}
	rq := ""
	if requirerCaps != "" {
		rq = "[dependencies]\ncapabilities = [" + requirerCaps + "]\n"
	}
	return fstest.MapFS{
		"version.txt":                       {Data: []byte("0.1.0")},
		"frameworks/prov/framework.toml":    {Data: []byte("name = \"prov\"\nversion = \"0.1.0\"\n" + pv + "[skills]\np = \"frameworks/prov/skills/p\"\n")},
		"frameworks/prov/skills/p/SKILL.md": {Data: []byte("p")},
		"frameworks/req/framework.toml":     {Data: []byte("name = \"req\"\nversion = \"0.1.0\"\n" + rq + "[skills]\nr = \"frameworks/req/skills/r\"\n")},
		"frameworks/req/skills/r/SKILL.md":  {Data: []byte("r")},
	}
}

func TestLoad_UnresolvedCapabilityFails(t *testing.T) {
	// req needs cap@1, prov provides nothing -> fail loud.
	_, err := Load(capFS("", `"logging@1"`))
	if err == nil || !strings.Contains(err.Error(), "logging@1") {
		t.Fatalf("unresolved capability should fail naming it, got %v", err)
	}
}

func TestLoad_SatisfiedCapabilityLoads(t *testing.T) {
	// prov provides logging@1, req requires it -> loads.
	if _, err := Load(capFS(`"logging@1"`, `"logging@1"`)); err != nil {
		t.Errorf("satisfied capability should load: %v", err)
	}
	// No capabilities at all -> loads (unchanged).
	if _, err := Load(capFS("", "")); err != nil {
		t.Errorf("no-capability catalog should load: %v", err)
	}
}

func TestLoad_MalformedCapabilityFails(t *testing.T) {
	if _, err := Load(capFS(`"logging"`, "")); err == nil {
		t.Errorf("a capability without @major should fail")
	}
	if _, err := Load(capFS(`"logging@x"`, "")); err == nil {
		t.Errorf("a capability with a non-integer major should fail")
	}
}

func TestLoadOverlays_CrossSourceCapabilityResolves(t *testing.T) {
	// requirer in base, provider in an overlay -> resolves (validation sees both).
	base := fstest.MapFS{
		"version.txt":                      {Data: []byte("0.1.0")},
		"frameworks/req/framework.toml":    {Data: []byte("name = \"req\"\nversion = \"0.1.0\"\n[dependencies]\ncapabilities = [\"logging@1\"]\n[skills]\nr = \"frameworks/req/skills/r\"\n")},
		"frameworks/req/skills/r/SKILL.md": {Data: []byte("r")},
	}
	overlay := fstest.MapFS{
		"frameworks/prov/framework.toml":    {Data: []byte("name = \"prov\"\nversion = \"0.1.0\"\n[provides]\ncapabilities = [\"logging@1\"]\n[skills]\np = \"frameworks/prov/skills/p\"\n")},
		"frameworks/prov/skills/p/SKILL.md": {Data: []byte("p")},
	}
	if _, err := LoadOverlays(base, overlay); err != nil {
		t.Errorf("cross-source capability should resolve: %v", err)
	}
}

// No shipped framework declares capabilities anymore (the mechanics stay
// covered by the fstest fixtures above); the embedded catalog's shipped
// surface is pinned by TestNew_CatalogShipsOnlyOnto.
