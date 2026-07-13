package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestCacheGC_Registered(t *testing.T) {
	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"cache", "gc", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("cache gc --help errored (command not registered?): %v", err)
	}
	if !strings.Contains(out.String(), "dry-run") {
		t.Errorf("cache gc --help should mention --dry-run; got %q", out.String())
	}
}

func TestCacheGC_RejectsStrayArg(t *testing.T) {
	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"cache", "gc", "stray"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("cache gc accepted a stray positional, want NoArgs rejection")
	}
}
