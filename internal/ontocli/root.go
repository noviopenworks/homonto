package ontocli

import "github.com/spf13/cobra"

// Version is the onto build version. Release builds stamp it via
// -ldflags "-X github.com/noviopenworks/homonto/internal/ontocli.Version=...".
var Version = "0.1.0-dev"

// NewRootCmd builds the root cobra command and registers subcommands.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "onto",
		Short:         "Managed spec-driven workflow operator",
		Version:       Version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print the onto version",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.Printf("onto %s\n", Version)
			return nil
		},
	})
	root.AddCommand(statusCmd())
	root.AddCommand(initCmd())
	root.AddCommand(newCmd())
	return root
}
