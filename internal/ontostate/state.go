// Package ontostate models the onto-state.yaml file used by the onto
// binary's workflow tracking. It is the only package in this module
// permitted to import gopkg.in/yaml.v3.
package ontostate

import (
	"fmt"
	"os"

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
	return Parse(b)
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
