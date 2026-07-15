package catalog

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestSatisfies(t *testing.T) {
	cases := []struct {
		v, c string
		want bool
	}{
		{"0.1.0", ">=0.1.0", true},
		{"0.2.0", ">=0.1.0", true},
		{"0.0.9", ">=0.1.0", false},
		{"1.0.0", ">1.0.0", false},
		{"1.0.1", ">1.0.0", true},
		{"1.0.0", "<=1.0.0", true},
		{"1.0.1", "<=1.0.0", false},
		{"0.9.0", "<1.0.0", true},
		{"1.0.0", "<1.0.0", false},
		{"0.1.0", "0.1.0", true}, // bare = exact
		{"0.1.0", "=0.1.0", true},
		{"0.2.0", "=0.1.0", false},
		{"0.10.0", ">=0.9.0", true}, // numeric, not lexical
	}
	for _, tc := range cases {
		got, err := satisfies(tc.v, tc.c)
		if err != nil {
			t.Errorf("satisfies(%q,%q) err=%v", tc.v, tc.c, err)
			continue
		}
		if got != tc.want {
			t.Errorf("satisfies(%q,%q)=%v, want %v", tc.v, tc.c, got, tc.want)
		}
	}
	// malformed → error
	if _, err := satisfies("1.0", ">=0.1.0"); err == nil {
		t.Errorf("malformed version should error")
	}
	if _, err := satisfies("1.0.0", "~1.0.0"); err == nil {
		t.Errorf("unsupported operator should error")
	}
}

func rangeFS(depfwDeps string) fstest.MapFS {
	return fstest.MapFS{
		"version.txt":                        {Data: []byte("0.1.0")},
		"frameworks/sp/framework.toml":       {Data: []byte("name = \"sp\"\nversion = \"0.1.0\"\n[skills]\ns = \"frameworks/sp/skills/s\"\n")},
		"frameworks/sp/skills/s/SKILL.md":    {Data: []byte("s")},
		"frameworks/depfw/framework.toml":    {Data: []byte("name = \"depfw\"\nversion = \"0.1.0\"\n[dependencies]\nframeworks = [" + depfwDeps + "]\n[skills]\nc = \"frameworks/depfw/skills/c\"\n")},
		"frameworks/depfw/skills/c/SKILL.md": {Data: []byte("c")},
	}
}

func TestLoad_RejectsOutOfRangeDependency(t *testing.T) {
	_, err := Load(rangeFS(`"sp@>=2.0.0"`))
	if err == nil {
		t.Fatal("out-of-range dependency should fail load")
	}
	if !strings.Contains(err.Error(), "sp") || !strings.Contains(err.Error(), "2.0.0") {
		t.Errorf("error should name the dep and constraint: %v", err)
	}
}

func TestLoad_AcceptsSatisfiedAndBareDependency(t *testing.T) {
	if _, err := Load(rangeFS(`"sp@>=0.1.0"`)); err != nil {
		t.Errorf("satisfied range should load: %v", err)
	}
	if _, err := Load(rangeFS(`"sp"`)); err != nil {
		t.Errorf("bare dep should load: %v", err)
	}
}
