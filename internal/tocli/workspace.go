package tocli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/noviopenworks/homonto/internal/tostate"
	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/cobra"
)

// tasksDir/archiveDir are to's territory. Fully disjoint from onto's
// docs/changes/ so a mixed repo never confuses either tool's commands —
// though homonto refuses to declare both frameworks anyway.
func tasksDir(root string) string   { return filepath.Join(root, "docs", "tasks") }
func archiveDir(root string) string { return filepath.Join(root, "docs", "tasks", "archive") }

func changeDir(root, name string) string { return filepath.Join(tasksDir(root), name) }
func statePath(root, name string) string {
	return filepath.Join(changeDir(root, name), tostate.FileName)
}
func planPath(root, name string) string { return filepath.Join(changeDir(root, name), "plan.md") }

// homontoConfig is the minimal shape of homonto.toml that the gate needs:
// just enough to detect whether a [frameworks.to] table is declared. It is
// intentionally a standalone struct, not homonto's own config type, so that
// to stays isolated from homonto's projection pipeline packages.
type homontoConfig struct {
	Frameworks map[string]any `toml:"frameworks"`
}

// gate enforces the framework-install precondition every mutating command
// (init, new, phase, done, abandon) requires: the project must have declared
// and applied [frameworks.to] through Homonto. The skills are the product —
// the gate guarantees no agent works inside the framework without them. It
// checks, in order, and returns on the first failure:
//
//  1. <root>/homonto.toml exists.
//  2. it declares a [frameworks.to] table.
//  3. <root>/.homonto/catalog/skills/to exists as a directory (i.e. the
//     declaration has been applied).
//
// Read-only commands (status, handoff) never call gate. gate performs no
// writes.
func gate(root string) error {
	tomlPath := filepath.Join(root, "homonto.toml")

	data, err := os.ReadFile(tomlPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("to: no homonto.toml found in %s; run `homonto init` first", root)
		}
		return fmt.Errorf("to: reading %s: %w", tomlPath, err)
	}

	var cfg homontoConfig
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("to: parsing %s: %w", tomlPath, err)
	}

	if _, ok := cfg.Frameworks["to"]; !ok {
		return fmt.Errorf("to: %s has no [frameworks.to] table; declare [frameworks.to] and run `homonto apply`", tomlPath)
	}

	catalogPath := filepath.Join(root, ".homonto", "catalog", "skills", "to")
	info, err := os.Stat(catalogPath)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("to: %s not found; run `homonto apply` to install the to framework", catalogPath)
	}

	return nil
}

// changeNamePattern is the accepted shape for a change name: one or more
// lowercase-alphanumeric segments joined by single hyphens. "archive" is
// additionally reserved (it is the archive directory itself).
var changeNamePattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

func validChangeName(name string) error {
	if name == "" {
		return fmt.Errorf("to: change name must not be empty")
	}
	if name != filepath.Base(name) || strings.Contains(name, "..") {
		return fmt.Errorf("to: change name %q must not contain path separators or \"..\"", name)
	}
	if !changeNamePattern.MatchString(name) {
		return fmt.Errorf("to: change name %q must match %s", name, changeNamePattern.String())
	}
	if name == "archive" {
		return fmt.Errorf("to: change name %q is reserved", name)
	}
	return nil
}

// loadChange loads an active (non-archived) change's state, with an error
// that distinguishes "never existed" from "already archived".
func loadChange(root, name string) (tostate.State, error) {
	if err := validChangeName(name); err != nil {
		return tostate.State{}, err
	}
	st, err := tostate.Load(statePath(root, name))
	if err == nil {
		return st, nil
	}
	if _, archErr := os.Stat(filepath.Join(archiveDir(root), name, tostate.FileName)); archErr == nil {
		return tostate.State{}, fmt.Errorf("to: change %q is archived at %s", name, filepath.Join(archiveDir(root), name))
	}
	return tostate.State{}, err
}

// archive moves an active change directory into docs/tasks/archive/. It
// refuses to clobber an existing archived change of the same name.
func archive(root, name string) (string, error) {
	dest := filepath.Join(archiveDir(root), name)
	if _, err := os.Stat(dest); err == nil {
		return "", fmt.Errorf("to: archive destination %s already exists", dest)
	}
	if err := os.MkdirAll(archiveDir(root), 0o755); err != nil {
		return "", fmt.Errorf("to: creating %s: %w", archiveDir(root), err)
	}
	if err := os.Rename(changeDir(root, name), dest); err != nil {
		return "", fmt.Errorf("to: archiving %s: %w", name, err)
	}
	return dest, nil
}

// printJSON marshals v with indentation to the command's stdout.
func printJSON(cmd *cobra.Command, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("to: encoding json: %w", err)
	}
	cmd.Println(string(b))
	return nil
}
