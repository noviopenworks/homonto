package cli

import (
	"fmt"
	"os"

	"github.com/noviopenworks/homonto/internal/fsutil"
	"github.com/noviopenworks/homonto/internal/importer"
	"github.com/spf13/cobra"
)

func importCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Bootstrap homonto.toml from your current setup",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfgPath, _ := cmd.Flags().GetString("config")
			force, _ := cmd.Flags().GetBool("force")
			if _, err := os.Stat(cfgPath); err == nil && !force {
				cmd.Printf("%s already exists; use --force to overwrite\n", cfgPath)
				return nil
			}
			home, _ := os.UserHomeDir()
			c, warnings, err := importer.Import(home)
			if err != nil {
				return err
			}
			data, err := importer.MarshalTOML(c)
			if err != nil {
				return err
			}
			// Back up an existing config before overwriting it (F48), so a
			// forced import over a valid hand-tuned config is recoverable.
			if existing, rerr := os.ReadFile(cfgPath); rerr == nil {
				if berr := os.WriteFile(cfgPath+".bak", existing, 0o644); berr != nil {
					return fmt.Errorf("import: backing up existing config to %s.bak: %w", cfgPath, berr)
				}
				cmd.Println("backed up existing config to", cfgPath+".bak")
			}
			if err := fsutil.WriteAtomic(cfgPath, data); err != nil {
				return err
			}
			cmd.Println("wrote", cfgPath)
			for _, w := range warnings {
				cmd.Println("  warn:", w)
			}
			return nil
		},
	}
	cmd.Flags().Bool("force", false, "overwrite existing config")
	return cmd
}
