package ontocli

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewRootCmdUse(t *testing.T) {
	cmd := NewRootCmd()
	if cmd.Use != "onto" {
		t.Fatalf("Use = %q, want %q", cmd.Use, "onto")
	}
}

func TestVersionCommand(t *testing.T) {
	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"version"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if got := out.String(); !strings.Contains(got, "onto "+Version) {
		t.Fatalf("got %q, want it to contain %q", got, "onto "+Version)
	}
}
