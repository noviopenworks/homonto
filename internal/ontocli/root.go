package ontocli

import (
	"github.com/noviopenworks/homonto/internal/buildinfo"
	"github.com/spf13/cobra"
)

// devVersion is the unstamped default; release builds override Version via
// -ldflags "-X github.com/noviopenworks/homonto/internal/ontocli.Version=...".
const devVersion = "0.1.0-dev"

// Version is the onto build version. It is a constant-initialized string so the
// linker's -X stamp takes effect; when unstamped (e.g. `go install ...@tag`,
// which applies no ldflags) buildinfo.Resolve recovers the module version.
var Version = devVersion

// NewRootCmd builds the root cobra command and registers subcommands.
func NewRootCmd() *cobra.Command {
	version := buildinfo.Resolve(Version, devVersion)
	root := &cobra.Command{
		Use:           "onto",
		Short:         "Managed spec-driven workflow operator",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print the onto version",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.Printf("onto %s\n", version)
			return nil
		},
	})
	root.AddCommand(statusCmd())
	root.AddCommand(graphCmd())
	root.AddCommand(initCmd())
	root.AddCommand(newCmd())
	root.AddCommand(advanceCmd())
	root.AddCommand(closeCmd())
	root.AddCommand(abandonCmd())
	root.AddCommand(doctorCmd())
	root.AddCommand(setCmd())
	root.AddCommand(stateCmd())
	root.AddCommand(gateCmd())
	root.AddCommand(scaleCmd())
	root.AddCommand(mergeDeltasCmd())
	root.AddCommand(handoffCmd())
	return root
}
