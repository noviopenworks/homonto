// Package ontostate models the onto-state.yaml file used by the onto
// binary's workflow tracking. It is the only package in this module
// permitted to import gopkg.in/yaml.v3.
package ontostate

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// validPhases enumerates the onto workflow phase set. The terminal phase is
// "close", not "archive".
var validPhases = map[string]bool{
	"open":   true,
	"design": true,
	"build":  true,
	"verify": true,
	"close":  true,
}

// State models the contents of onto-state.yaml.
type State struct {
	Change   string   `yaml:"change"`
	Workflow string   `yaml:"workflow,omitempty"`
	Phase    string   `yaml:"phase"`
	Created  string   `yaml:"created,omitempty"`
	BaseRef  string   `yaml:"base_ref,omitempty"`
	Deps     []string `yaml:"deps,omitempty"`
	Archived bool     `yaml:"archived,omitempty"`
}

// Parse decodes raw YAML bytes into a State. It never panics; malformed
// input is returned as an error whose message identifies the onto-state
// source.
func Parse(b []byte) (state State, err error) {
	defer func() {
		if r := recover(); r != nil {
			state = State{}
			err = fmt.Errorf("onto-state: parse panic: %v", r)
		}
	}()

	if unmarshalErr := yaml.Unmarshal(b, &state); unmarshalErr != nil {
		return State{}, fmt.Errorf("onto-state: %w", unmarshalErr)
	}
	return state, nil
}

// Load reads the file at path and parses it as a State.
func Load(path string) (State, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return State{}, fmt.Errorf("onto-state: failed to read %s: %w", path, err)
	}
	state, err := Parse(b)
	if err != nil {
		return State{}, fmt.Errorf("%s: %w", path, err)
	}
	return state, nil
}

// Validate checks that the State satisfies the invariants required to
// derive a workflow phase: Change must be non-empty and Phase must be one
// of the known onto workflow phases.
func (s State) Validate() error {
	if s.Change == "" {
		return fmt.Errorf("onto-state: change is required")
	}
	if !validPhases[s.Phase] {
		return fmt.Errorf("onto-state: phase %q is not one of open|design|build|verify|close", s.Phase)
	}
	return nil
}

// DerivePhase validates the State and returns its Phase.
func (s State) DerivePhase() (string, error) {
	if err := s.Validate(); err != nil {
		return "", err
	}
	return s.Phase, nil
}

// Marshal encodes a State to YAML bytes.
func Marshal(s State) ([]byte, error) {
	return yaml.Marshal(s)
}

// Save writes s to path as YAML, creating parent directories as needed. It
// writes to a temp file next to path and renames it into place, removing
// the temp file if any step fails.
func Save(path string, s State) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("onto-state: failed to create directory for %s: %w", path, err)
	}
	b, err := Marshal(s)
	if err != nil {
		return fmt.Errorf("onto-state: failed to marshal %s: %w", path, err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("onto-state: failed to write %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("onto-state: failed to rename %s to %s: %w", tmp, path, err)
	}
	return nil
}

// orderedPhases lists the onto workflow phases in traversal order. "close"
// is the terminal phase.
var orderedPhases = []string{"open", "design", "build", "verify", "close"}

// RequiredArtifacts returns the filenames that must exist in a change
// directory for the given phase. Requirements are cumulative: each phase
// requires the base skeleton set plus every artifact introduced by earlier
// phases. Unknown phases fall back to the base skeleton set. The returned
// slice is a fresh copy on every call.
func RequiredArtifacts(phase string) []string {
	base := []string{"onto-state.yaml", "proposal.md", "tasks.md"}
	switch phase {
	case "design":
		return append(base, "design.md")
	case "build":
		return append(base, "design.md", "plan.md")
	case "verify", "close":
		return append(base, "design.md", "plan.md", "verification.md")
	default:
		return base
	}
}

// NextPhase returns the phase that follows phase in the ordered onto
// workflow sequence ["open","design","build","verify","close"], and true.
// It returns ("", false) when phase is "close" (the terminal phase) or is
// not a recognized phase.
func NextPhase(phase string) (string, bool) {
	for i, p := range orderedPhases {
		if p == phase {
			if i+1 < len(orderedPhases) {
				return orderedPhases[i+1], true
			}
			return "", false
		}
	}
	return "", false
}

// TasksAllChecked reads the file at tasksPath and reports whether every
// checkbox line ("- [ ]" or "- [x]"/"- [X]", after trimming leading
// whitespace) is checked. It returns false if the file contains no
// checkbox lines. It returns an error if the file cannot be read.
func TasksAllChecked(tasksPath string) (bool, error) {
	f, err := os.Open(tasksPath)
	if err != nil {
		return false, fmt.Errorf("onto-state: failed to read %s: %w", tasksPath, err)
	}
	defer f.Close()

	sawCheckbox := false
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		switch {
		case strings.HasPrefix(line, "- [ ]"):
			return false, nil
		case strings.HasPrefix(line, "- [x]"), strings.HasPrefix(line, "- [X]"):
			sawCheckbox = true
		}
	}
	if err := scanner.Err(); err != nil {
		return false, fmt.Errorf("onto-state: failed to read %s: %w", tasksPath, err)
	}
	return sawCheckbox, nil
}

// DepsResolved reports which of deps are not yet archived under root. A dep
// is resolved iff filepath.Glob(filepath.Join(root,"docs","changes","archive","*-"+dep))
// finds at least one match — i.e. the dep was archived under a date-prefixed
// directory such as docs/changes/archive/2026-07-10-<dep>/. The returned
// slice contains the unresolved subset of deps, in input order. A nil or
// empty deps returns an empty (len 0) slice.
func DepsResolved(root string, deps []string) []string {
	unresolved := make([]string, 0, len(deps))
	for _, dep := range deps {
		pattern := filepath.Join(root, "docs", "changes", "archive", "*-"+dep)
		matches, _ := filepath.Glob(pattern)
		if len(matches) == 0 {
			unresolved = append(unresolved, dep)
		}
	}
	return unresolved
}

// ValidateSkeleton loads onto-state.yaml from changeDir, derives its phase,
// and checks that every artifact RequiredArtifacts(phase) names is present
// in changeDir. It returns an error naming the first missing artifact, or
// nil if all are present.
func ValidateSkeleton(changeDir string) error {
	state, err := Load(filepath.Join(changeDir, "onto-state.yaml"))
	if err != nil {
		return err
	}
	phase, err := state.DerivePhase()
	if err != nil {
		return err
	}
	for _, name := range RequiredArtifacts(phase) {
		if _, err := os.Stat(filepath.Join(changeDir, name)); err != nil {
			return fmt.Errorf("onto-state: missing required artifact %s in %s: %w", name, changeDir, err)
		}
	}
	return nil
}
