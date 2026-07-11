package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/noviopenworks/homonto/internal/agentlock"
	"github.com/noviopenworks/homonto/internal/catalog"
	"github.com/noviopenworks/homonto/internal/config"
	"github.com/spf13/cobra"
)

// This file holds the `agents` parent command and the helpers shared across the
// subcommands. Each subcommand lives in its own agents_<verb>.go for reviewability:
// agents_list.go, agents_add.go, agents_update.go, agents_doctor.go,
// agents_prune.go.

func agentsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agents",
		Short: "Inspect lifecycle-managed agents",
	}
	cmd.AddCommand(agentsListCmd())
	cmd.AddCommand(agentsAddCmd())
	cmd.AddCommand(agentsUpdateCmd())
	cmd.AddCommand(agentsDoctorCmd())
	cmd.AddCommand(agentsPruneCmd())
	return cmd
}

// firstRecordedHash returns the hash of the first install by sorted tool key.
// All targets record the same content hash at install, so any one suffices.
func firstRecordedHash(installed map[string]agentlock.Install) string {
	for _, tool := range sortedKeys(installed) {
		return installed[tool].Hash
	}
	return ""
}

func containsStr(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}

func sortedStrings(xs []string) []string {
	out := append([]string(nil), xs...)
	sort.Strings(out)
	return out
}

func sortedKeys(m map[string]agentlock.Install) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedKeysAgents(m map[string]agentlock.Agent) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// agentMode returns the effective materialize mode for an agent. A builtin:
// source has no stable on-disk path to symlink, so it is copy-only: an explicit
// mode=link on a builtin agent is an error, and an unspecified mode defaults to
// copy (rather than the general link default). local: agents keep the normal
// mode default.
func agentMode(name string, ag config.Agent) (string, error) {
	if strings.HasPrefix(ag.Source, "builtin:") {
		if ag.Mode == "link" {
			return "", fmt.Errorf("%q uses builtin: with link mode, but builtin sources have no local path to link; use mode=copy", name)
		}
		return "copy", nil
	}
	return ag.ModeOrDefault(), nil
}

// resolveAgentSource resolves a declared agent's source to its content:
// local:<x> reads homonto/agents/<x>.md under the config dir; builtin:<x> reads
// the embedded catalog's curated agent content by name (unknown name is an
// error); any other scheme is not yet supported (remote deferred).
func resolveAgentSource(ag config.Agent, cfgDir string) ([]byte, error) {
	switch {
	case strings.HasPrefix(ag.Source, "local:"):
		p := filepath.Join(cfgDir, "homonto", "agents", strings.TrimPrefix(ag.Source, "local:")+".md")
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, fmt.Errorf("source file %s: %w", p, err)
		}
		return b, nil
	case strings.HasPrefix(ag.Source, "builtin:"):
		name := strings.TrimPrefix(ag.Source, "builtin:")
		cat, err := catalog.New()
		if err != nil {
			return nil, err
		}
		b, ok, err := cat.SubagentContent(name)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, fmt.Errorf("unknown builtin agent %q", name)
		}
		return b, nil
	default:
		return nil, fmt.Errorf("unsupported agent source %q (remote sources are not yet supported)", ag.Source)
	}
}

// isSymlinkTo reports whether dst is a symlink whose target is exactly src.
func isSymlinkTo(dst, src string) bool {
	fi, err := os.Lstat(dst)
	if err != nil || fi.Mode()&os.ModeSymlink == 0 {
		return false
	}
	target, err := os.Readlink(dst)
	return err == nil && target == src
}
