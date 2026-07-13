package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/noviopenworks/homonto/internal/fsutil"
)

// Entry is the last-applied record for one managed key. Desired holds the
// unresolved value (may contain ${...} tokens); Applied holds a non-secret
// sha256 of the resolved value that was written to disk. Neither field ever
// contains a plaintext secret, so state.json is safe to share.
type Entry struct {
	Desired string `json:"desired"`
	Applied string `json:"applied"`
}

// State is the last-applied snapshot, keyed tool -> managed key -> Entry.
// CatalogVersion is the embedded-catalog version last successfully materialized;
// it is global (not per-tool) and omitted when empty so pre-catalog state.json
// files stay backward-compatible (absent = "force materialize").
type State struct {
	// SchemaVersion is the state.json format version. Absent/0 means a legacy
	// (pre-versioning) file and is treated as the current version; a value greater
	// than CurrentStateSchemaVersion is rejected fail-closed at load.
	SchemaVersion  int                         `json:"schemaVersion,omitempty"`
	Managed        map[string]map[string]Entry `json:"managed"`
	CatalogVersion string                      `json:"catalogVersion,omitempty"`
}

// CurrentStateSchemaVersion is the state.json schema version this binary writes.
const CurrentStateSchemaVersion = 1

// CatalogVersionRecorded returns the catalog version last materialized, or "".
func (s *State) CatalogVersionRecorded() string { return s.CatalogVersion }

// SetCatalogVersion records the catalog version after a successful materialize.
func (s *State) SetCatalogVersion(v string) { s.CatalogVersion = v }

func newState() *State { return &State{Managed: map[string]map[string]Entry{}} }

func file(dir string) string { return filepath.Join(dir, "state.json") }

// Load reads <dir>/state.json, returning an empty State if the file is absent.
func Load(dir string) (*State, error) {
	data, err := os.ReadFile(file(dir))
	if errors.Is(err, os.ErrNotExist) {
		return newState(), nil
	}
	if err != nil {
		return nil, err
	}
	s := newState()
	if err := json.Unmarshal(data, s); err != nil {
		return nil, err
	}
	if s.SchemaVersion > CurrentStateSchemaVersion {
		return nil, fmt.Errorf("state: unknown state schema version %d (this binary supports up to %d) — upgrade homonto", s.SchemaVersion, CurrentStateSchemaVersion)
	}
	if s.Managed == nil {
		s.Managed = map[string]map[string]Entry{}
	}
	return s, nil
}

// Save writes the state atomically (temp + fsync + rename), creating dir if
// needed. state.json is one of homonto's own control-plane files, so it is
// written no-follow (a symlinked target is refused, never followed) at 0600.
func (s *State) Save(dir string) error {
	s.SchemaVersion = CurrentStateSchemaVersion
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return fsutil.WriteControlPlane(file(dir), data, 0o600)
}

// Set records the unresolved desired value and the applied-value hash for a key.
func (s *State) Set(tool, key, desired, appliedHash string) {
	if s.Managed[tool] == nil {
		s.Managed[tool] = map[string]Entry{}
	}
	s.Managed[tool][key] = Entry{Desired: desired, Applied: appliedHash}
}

// Get returns the recorded Entry for a key and whether it exists.
func (s *State) Get(tool, key string) (Entry, bool) {
	e, ok := s.Managed[tool][key]
	return e, ok
}

// Keys returns the sorted managed keys recorded for a tool.
func (s *State) Keys(tool string) []string {
	keys := make([]string, 0, len(s.Managed[tool]))
	for k := range s.Managed[tool] {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Delete drops the record for a key (after its on-disk value was pruned).
func (s *State) Delete(tool, key string) {
	delete(s.Managed[tool], key)
}
