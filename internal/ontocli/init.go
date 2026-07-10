package ontocli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/cobra"
)

// docsLayout is the fixed set of documentation directories "onto init"
// scaffolds once the framework gate passes.
var docsLayout = []string{
	filepath.Join("docs", "changes"),
	filepath.Join("docs", "specs"),
	filepath.Join("docs", "adr"),
	filepath.Join("docs", "guides"),
}

// initCmd builds the "onto init" subcommand: it enforces gate(dir) (the
// framework-install precondition) before scaffolding the docs/ layout, and
// performs no writes at all if the gate fails.
func initCmd() *cobra.Command {
	var dir string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Scaffold the onto docs layout, if the onto framework is installed",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runInit(cmd, dir)
		},
	}
	cmd.Flags().StringVar(&dir, "dir", ".", "workspace root to initialize")
	return cmd
}

// runInit enforces gate(root) and, only on success, idempotently scaffolds
// the docs/ layout: each directory is created with os.MkdirAll if missing,
// and left untouched (never overwritten) if it already exists. It reports
// one line per directory describing the outcome.
func runInit(cmd *cobra.Command, root string) error {
	if err := gate(root); err != nil {
		return err
	}

	for _, d := range docsLayout {
		path := filepath.Join(root, d)

		_, statErr := os.Stat(path)
		preExisted := statErr == nil

		if err := os.MkdirAll(path, 0o755); err != nil {
			return fmt.Errorf("onto init: creating %s: %w", path, err)
		}

		if preExisted {
			cmd.Printf("exists %s\n", path)
		} else {
			cmd.Printf("created %s\n", path)
		}
	}

	return nil
}

// homontoConfig is the minimal shape of homonto.toml that the gate needs:
// just enough to detect whether a [frameworks.onto] table is declared.
// It is intentionally a standalone struct, not homonto's own config type,
// so that onto stays isolated from homonto's projection pipeline packages.
type homontoConfig struct {
	Frameworks map[string]any `toml:"frameworks"`
}

// gate enforces the framework-install precondition that "onto init" (a
// mutating command) requires before it may scaffold anything: the project
// must have declared and applied [frameworks.onto] through Homonto. It
// checks, in order, and returns on the first failure:
//
//  1. <root>/homonto.toml exists.
//  2. it declares a [frameworks.onto] table.
//  3. <root>/.homonto/catalog/skills/onto exists as a directory (i.e. the
//     declaration has been applied).
//
// gate performs no writes; it is safe to call before any scaffolding.
func gate(root string) error {
	tomlPath := filepath.Join(root, "homonto.toml")

	data, err := os.ReadFile(tomlPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("onto init: no homonto.toml found in %s; run `homonto init` first", root)
		}
		return fmt.Errorf("onto init: reading %s: %w", tomlPath, err)
	}

	var cfg homontoConfig
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("onto init: parsing %s: %w", tomlPath, err)
	}

	if _, ok := cfg.Frameworks["onto"]; !ok {
		return fmt.Errorf("onto init: %s has no [frameworks.onto] table; declare [frameworks.onto] and run `homonto apply`", tomlPath)
	}

	catalogPath := filepath.Join(root, ".homonto", "catalog", "skills", "onto")
	info, err := os.Stat(catalogPath)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("onto init: %s not found; run `homonto apply` to install the onto framework", catalogPath)
	}

	return nil
}
