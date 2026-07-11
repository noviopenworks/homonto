package cli

import (
	"sort"
	"strings"

	"github.com/noviopenworks/homonto/internal/config"
	"github.com/spf13/cobra"
)

func agentsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agents",
		Short: "Inspect lifecycle-managed agents",
	}
	cmd.AddCommand(agentsListCmd())
	return cmd
}

func agentsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List declared agents (read-only)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfgPath, _ := cmd.Flags().GetString("config")
			c, err := config.Load(cfgPath)
			if err != nil {
				return err
			}
			names := make([]string, 0, len(c.Agents))
			for n := range c.Agents {
				names = append(names, n)
			}
			sort.Strings(names)
			if len(names) == 0 {
				cmd.Println("No agents declared.")
				return nil
			}
			for _, n := range names {
				ag := c.Agents[n]
				v := ag.Version
				if v == "" {
					v = "unpinned"
				}
				cmd.Printf("%s: %s  version=%s  targets=%s  mode=%s\n",
					n, ag.Source, v, strings.Join(ag.TargetsOrAll(), ","), ag.ModeOrDefault())
			}
			return nil
		},
	}
}
