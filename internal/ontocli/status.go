package ontocli

import (
	"path/filepath"
	"strings"

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
	matches, err := filepath.Glob(filepath.Join(root, "docs", "changes", "*", "onto-state.yaml"))
	if err != nil {
		return err
	}

	archivePrefix := filepath.Join(root, "docs", "changes", "archive") + string(filepath.Separator)
	for _, path := range matches {
		if strings.HasPrefix(path, archivePrefix) {
			continue
		}

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

		cmd.Printf("%s: %s\n", state.Change, phase)
	}

	return nil
}
