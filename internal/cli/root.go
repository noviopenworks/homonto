package cli

import "github.com/spf13/cobra"

// Version is the homonto build version. Release builds stamp it via
// -ldflags "-X github.com/noviopenworks/homonto/internal/cli.Version=...".
var Version = "0.1.0-dev"

// NewRootCmd builds the root cobra command and registers subcommands.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "homonto",
		Short:         "Declarative config for AI coding tools",
		Version:       Version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().String("config", "homonto.toml", "path to homonto config")

	root.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print the homonto version",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.Printf("homonto %s\n", Version)
			return nil
		},
	})
	root.AddCommand(planCmd(), applyCmd(), statusCmd(), doctorCmd(), initCmd(), importCmd(), cacheCmd())
	return root
}
