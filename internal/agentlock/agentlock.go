// Package agentlock is the ground-truth lockfile for lifecycle-managed agents
// (.homonto/agents-lock.json). It records, per agent, what was installed where
// and the content hash, so later update/pin/doctor/migrate can reason about
// drift without re-deriving it. It is intentionally separate from state.json
// (the plan/apply model) because agent lifecycle needs installed-version truth.
package agentlock

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/noviopenworks/homonto/internal/fsutil"
)

// Install records one materialized copy of an agent: the destination path and
// the content hash written there (for copy) or of the linked source (for link).
type Install struct {
	Path string `json:"path"`
	Hash string `json:"hash"`
}

// Agent is the recorded install of one declared agent across its targets.
type Agent struct {
	Source    string             `json:"source"`
	Version   string             `json:"version,omitempty"`
	Mode      string             `json:"mode"`
	Targets   []string           `json:"targets"`
	Installed map[string]Install `json:"installed"` // tool -> install
}

// Lock is the whole lockfile: every agent we have installed, keyed by name.
type Lock struct {
	Agents map[string]Agent `json:"agents"`
}

func file(homontoDir string) string { return filepath.Join(homontoDir, "agents-lock.json") }

// Load reads <homontoDir>/agents-lock.json, returning an empty Lock when the
// file is absent. A malformed file surfaces its parse error rather than being
// silently discarded.
func Load(homontoDir string) (*Lock, error) {
	data, err := os.ReadFile(file(homontoDir))
	if errors.Is(err, os.ErrNotExist) {
		return &Lock{Agents: map[string]Agent{}}, nil
	}
	if err != nil {
		return nil, err
	}
	var l Lock
	if err := json.Unmarshal(data, &l); err != nil {
		return nil, err
	}
	if l.Agents == nil {
		l.Agents = map[string]Agent{}
	}
	return &l, nil
}

// Save writes the lockfile atomically. Marshaling with sorted map keys and
// stable 2-space indentation makes re-saves byte-deterministic and diff-friendly.
func (l *Lock) Save(homontoDir string) error {
	data, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return err
	}
	return fsutil.WriteAtomic(file(homontoDir), data)
}

// HashContent returns the sha256 hex digest of b.
func HashContent(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
