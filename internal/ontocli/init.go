package ontocli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

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
