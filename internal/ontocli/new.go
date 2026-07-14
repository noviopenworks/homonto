package ontocli

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/noviopenworks/homonto/internal/ontostate"
	"github.com/spf13/cobra"
)

// changeNamePattern is the accepted shape for a change name: one or more
// lowercase-alphanumeric segments joined by single hyphens.
var changeNamePattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// validChangeName rejects any change name that is empty, escapes its own
// base name (e.g. via ".." or a path separator), or does not match the
// lowercase-hyphenated shape required for a docs/changes/<name> directory.
func validChangeName(name string) error {
	if name == "" {
		return fmt.Errorf("onto new: change name must not be empty")
	}
	if name != filepath.Base(name) {
		return fmt.Errorf("onto new: change name %q must not contain path separators", name)
	}
	if strings.Contains(name, "..") || strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return fmt.Errorf("onto new: change name %q must not contain \"..\", \"/\", or \"\\\"", name)
	}
	if !changeNamePattern.MatchString(name) {
		return fmt.Errorf("onto new: change name %q must match %s", name, changeNamePattern.String())
	}
	return nil
}

// newCmd builds the "onto new <change-name>" subcommand: it enforces
// gate(dir) (the framework-install precondition) and validChangeName before
// scaffolding a new change-workspace skeleton, and performs no writes at
// all if either check fails or the change directory already exists.
func newCmd() *cobra.Command {
	var (
		dir      string
		workflow string
	)

	cmd := &cobra.Command{
		Use:   "new <change-name>",
		Short: "Create a new change-workspace skeleton, if the onto framework is installed",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runNew(cmd, dir, args[0], workflow)
		},
	}
	cmd.Flags().StringVar(&dir, "dir", ".", "workspace root to create the change in")
	cmd.Flags().StringVar(&workflow, "workflow", "full", "workflow for the change: full, fix, or tweak")
	return cmd
}

// runNew enforces gate(root) then validChangeName(name), refuses to
// clobber an existing docs/changes/<name> directory, and only then
// scaffolds onto-state.yaml plus an empty proposal.md (and, for the fix/tweak
// presets, tasks.md — full derives its task list in design). Each file is
// written only if absent. It reports the created change and its files.
func runNew(cmd *cobra.Command, root, name, workflow string) error {
	if err := gate(root); err != nil {
		return err
	}

	if err := validChangeName(name); err != nil {
		return err
	}

	if !ontostate.ValidWorkflow(workflow) {
		return fmt.Errorf("onto new: workflow %q is not one of full|fix|tweak", workflow)
	}

	changeDir := filepath.Join(root, "docs", "changes", name)
	if _, err := os.Stat(changeDir); err == nil {
		return fmt.Errorf("onto new: change %q already exists at %s", name, changeDir)
	}

	if err := os.MkdirAll(changeDir, 0o755); err != nil {
		return fmt.Errorf("onto new: creating %s: %w", changeDir, err)
	}

	st := ontostate.State{
		Change:   name,
		ID:       ontostate.NewID(),
		Workflow: workflow,
		Phase:    "open",
		Created:  time.Now().Format("2006-01-02"),
	}
	statePath := filepath.Join(changeDir, "onto-state.yaml")
	if err := ontostate.Save(statePath, st); err != nil {
		return fmt.Errorf("onto new: %w", err)
	}

	// Scaffold the open-phase skeleton. A full change writes its task list from
	// the confirmed design (onto-design creates tasks.md), so `new` only lays down
	// proposal.md; the fix/tweak presets skip design and decompose at open-lite,
	// so they also get tasks.md now. This matches RequiredArtifacts(open, …).
	files := []string{"proposal.md"}
	if workflow == "fix" || workflow == "tweak" {
		files = append(files, "tasks.md")
	}
	for _, f := range files {
		path := filepath.Join(changeDir, f)
		if _, err := os.Stat(path); err == nil {
			continue
		}
		if err := os.WriteFile(path, []byte{}, 0o644); err != nil {
			return fmt.Errorf("onto new: creating %s: %w", path, err)
		}
	}

	cmd.Printf("created change %q at %s\n", name, changeDir)
	cmd.Printf("  %s\n", statePath)
	for _, f := range files {
		cmd.Printf("  %s\n", filepath.Join(changeDir, f))
	}

	return nil
}
