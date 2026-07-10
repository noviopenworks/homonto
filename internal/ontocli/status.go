package ontocli

import (
	"path/filepath"

	"github.com/noviopenworks/homonto/internal/ontostate"
	"github.com/spf13/cobra"
)

// statusCmd builds the "onto status" subcommand: a strictly read-only,
// config-independent diagnostic that inspects an existing workspace's
// docs/changes/*/onto-state.yaml files. It never constructs a homonto
// config/engine, never reads homonto.toml, and performs zero writes.
func statusCmd() *cobra.Command {
	var dir string

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Report the derived phase of each active onto change (read-only)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runStatus(cmd, dir)
		},
	}
	cmd.Flags().StringVar(&dir, "dir", ".", "workspace root to inspect")
	return cmd
}

func runStatus(cmd *cobra.Command, root string) error {
	// The single "*" wildcard only matches direct children of docs/changes/
	// (it does not cross path separators), so it structurally cannot match
	// archived changes, which live one level deeper at
	// docs/changes/archive/<name>/onto-state.yaml. Archived changes are
	// therefore excluded from the results by the shape of this glob, with
	// no separate filtering step required.
	matches, err := filepath.Glob(filepath.Join(root, "docs", "changes", "*", "onto-state.yaml"))
	if err != nil {
		return err
	}

	for _, path := range matches {
		changeDir := filepath.Base(filepath.Dir(path))

		state, loadErr := ontostate.Load(path)
		if loadErr != nil {
			cmd.Printf("%s: invalid (%v)\n", changeDir, loadErr)
			continue
		}

		phase, deriveErr := state.DerivePhase()
		if deriveErr != nil {
			cmd.Printf("%s: invalid (%v)\n", changeDir, deriveErr)
			continue
		}

		if skeletonErr := ontostate.ValidateSkeleton(filepath.Dir(path)); skeletonErr != nil {
			cmd.Printf("%s: %s — skeleton: %v\n", state.Change, phase, skeletonErr)
		} else {
			cmd.Printf("%s: %s — skeleton ok\n", state.Change, phase)
		}
	}

	return nil
}
