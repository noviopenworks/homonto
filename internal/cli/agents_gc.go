package cli

import (
	"path/filepath"

	"github.com/noviopenworks/homonto/internal/agentblob"
	"github.com/noviopenworks/homonto/internal/agentlock"
	"github.com/spf13/cobra"
)

// agentsGCCmd builds "agents gc": it reclaims base blobs under
// .homonto/agents-blobs that no lockfile install still references. Base blobs
// accumulate as `agents update` advances an install's recorded base to a new
// source; the superseded bases are then dead. GC is safe because content is
// addressed by hash — an unreferenced blob can never be needed again. --dry-run
// lists what would be reclaimed without removing anything.
func agentsGCCmd() *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "gc",
		Short: "Reclaim unreferenced agent base blobs (.homonto/agents-blobs)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfgPath, _ := cmd.Flags().GetString("config")
			homontoDir := filepath.Join(filepath.Dir(cfgPath), ".homonto")

			lock, err := agentlock.Load(homontoDir)
			if err != nil {
				return err
			}
			// A blob is live iff some installed target still records its hash as
			// the current base.
			referenced := map[string]bool{}
			for _, ag := range lock.Agents {
				for _, ins := range ag.Installed {
					if ins.Hash != "" {
						referenced[ins.Hash] = true
					}
				}
			}

			dead, err := agentblob.Reclaim(homontoDir, referenced, dryRun)
			if err != nil {
				return err
			}
			if len(dead) == 0 {
				cmd.Println("no unreferenced blobs")
				return nil
			}
			verb := "reclaimed"
			if dryRun {
				verb = "would reclaim"
			}
			for _, h := range dead {
				cmd.Printf("%s %s\n", verb, h)
			}
			cmd.Printf("agents gc: %s %d blob(s)\n", verb, len(dead))
			return nil
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "list unreferenced blobs without removing anything")
	return cmd
}
