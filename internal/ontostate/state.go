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

// CurrentSchemaVersion is the schema_version every write emits.
const CurrentSchemaVersion = 1

// enum membership sets for optional gated core fields. An empty value is
// always allowed (legacy-tolerant); a non-empty value must be a member.
var (
	validWorkflows     = map[string]bool{"full": true, "fix": true, "tweak": true}
	validIsolations    = map[string]bool{"branch": true, "worktree": true}
	validBuildModes    = map[string]bool{"direct": true, "subagent": true}
	validTDDModes      = map[string]bool{"tdd": true, "direct": true}
	validVerifyScales  = map[string]bool{"light": true, "full": true}
	validVerifyResults = map[string]bool{"pending": true, "pass": true, "fail": true}
)

// ValidWorkflow reports whether w is a recognized workflow value.
func ValidWorkflow(w string) bool { return validWorkflows[w] }

// GuidesResolved reports whether the guides obligation is discharged: either
// "updated" or a "waived:<reason>". Empty and "pending" are unresolved. It
// answers the close-phase question "were the guides dealt with", distinct from
// ValidGuides which only checks the value is well-formed.
func GuidesResolved(v string) bool {
	return v == "updated" || strings.HasPrefix(v, "waived:")
}

// ValidGuides reports whether v is a recognized guides value: empty (unset),
// "pending", "updated", or any "waived:<reason>". The waived form is a prefix,
// not a fixed member, so guides cannot use the enum-setter machinery.
func ValidGuides(v string) bool {
	if r, ok := strings.CutPrefix(v, "waived:"); ok {
		return strings.TrimSpace(r) != "" // a waiver must carry a reason
	}
	return v == "" || v == "pending" || v == "updated"
}

// Verify holds the gated verify-phase fields.
type Verify struct {
	Scale  string `yaml:"scale,omitempty" json:"scale,omitempty"`   // light | full | ""
	Result string `yaml:"result,omitempty" json:"result,omitempty"` // pending | pass | fail
}

// Close holds the gated close-phase progress fields.
type Close struct {
	Merged bool `yaml:"merged,omitempty" json:"merged,omitempty"`
}

// Observed carries observational state. It is never gated: no field here may
// block a transition, and unknown values in it can never break a gate.
type Observed struct {
	Metrics         map[string]string `yaml:"metrics,omitempty" json:"metrics,omitempty"` // phase -> YYYY-MM-DD
	TasksTotal      int               `yaml:"tasks_total,omitempty" json:"tasks_total,omitempty"`
	VerifyRounds    int               `yaml:"verify_rounds,omitempty" json:"verify_rounds,omitempty"`
	PresetEscalated bool              `yaml:"preset_escalated,omitempty" json:"preset_escalated,omitempty"`
}

// State models onto-state.yaml: a schema_version, a flat gated core, and a
// carried observational group. Validation inspects the core only.
type State struct {
	SchemaVersion int `yaml:"schema_version,omitempty" json:"schema_version,omitempty"`

	// gated core
	Change string `yaml:"change" json:"change"`
	// ID is a stable, name-independent identifier assigned once at `onto new` and
	// never rewritten by any later command, so a change's identity survives a
	// rename. Absent on legacy states (never retro-minted on read).
	ID       string   `yaml:"id,omitempty" json:"id,omitempty"`
	Workflow string   `yaml:"workflow,omitempty" json:"workflow,omitempty"`
	Phase    string   `yaml:"phase" json:"phase"`
	Created  string   `yaml:"created,omitempty" json:"created,omitempty"`
	BaseRef  string   `yaml:"base_ref,omitempty" json:"base_ref,omitempty"`
	Deps     []string `yaml:"deps,omitempty" json:"deps,omitempty"`
	// Supersedes lists change names this change replaces/obsoletes (a traceability
	// relationship surfaced by `onto graph`). Ungated: never blocks a transition.
	Supersedes []string `yaml:"supersedes,omitempty" json:"supersedes,omitempty"`
	// DeviatesFrom lists targets (decisions, specs, or prior changes) this change
	// knowingly diverges from — an honest record of implementation divergence
	// surfaced by `onto graph`. Ungated: never blocks a transition.
	DeviatesFrom []string `yaml:"deviates_from,omitempty" json:"deviates_from,omitempty"`
	Isolation    string   `yaml:"isolation,omitempty" json:"isolation,omitempty"`
	BuildMode    string   `yaml:"build_mode,omitempty" json:"build_mode,omitempty"`
	TDDMode      string   `yaml:"tdd_mode,omitempty" json:"tdd_mode,omitempty"`
	Verify       Verify   `yaml:"verify,omitempty" json:"verify,omitempty"`
	Close        Close    `yaml:"close,omitempty" json:"close,omitempty"`
	Directive    string   `yaml:"directive,omitempty" json:"directive,omitempty"`
	Guides       string   `yaml:"guides,omitempty" json:"guides,omitempty"` // "" | pending | updated | waived:<reason>
	Archived     bool     `yaml:"archived,omitempty" json:"archived,omitempty"`

	// observational (carried, never gated)
	Observed Observed `yaml:"observed,omitempty" json:"observed,omitempty"`
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
	state, err := parseAndMigrate(b, path)
	if err != nil {
		return State{}, fmt.Errorf("%s: %w", path, err)
	}
	return state, nil
}

// Validate checks presence and shape of the gated core: Change and Phase must
// be present/known, and every optional enum, when non-empty, must be a member
// of its set. It never inspects Observed (B1: shape, not judgment).
func (s State) Validate() error {
	if s.Change == "" {
		return fmt.Errorf("onto-state: change is required")
	}
	if !validPhases[s.Phase] {
		return fmt.Errorf("onto-state: phase %q is not one of open|design|build|verify|close", s.Phase)
	}
	if s.Workflow != "" && !validWorkflows[s.Workflow] {
		return fmt.Errorf("onto-state: workflow %q is not one of full|fix|tweak", s.Workflow)
	}
	if s.Isolation != "" && !validIsolations[s.Isolation] {
		return fmt.Errorf("onto-state: isolation %q is not one of branch|worktree", s.Isolation)
	}
	if s.BuildMode != "" && !validBuildModes[s.BuildMode] {
		return fmt.Errorf("onto-state: build_mode %q is not one of direct|subagent", s.BuildMode)
	}
	if s.TDDMode != "" && !validTDDModes[s.TDDMode] {
		return fmt.Errorf("onto-state: tdd_mode %q is not one of tdd|direct", s.TDDMode)
	}
	if s.Verify.Scale != "" && !validVerifyScales[s.Verify.Scale] {
		return fmt.Errorf("onto-state: verify.scale %q is not one of light|full", s.Verify.Scale)
	}
	if s.Verify.Result != "" && !validVerifyResults[s.Verify.Result] {
		return fmt.Errorf("onto-state: verify.result %q is not one of pending|pass|fail", s.Verify.Result)
	}
	if !ValidGuides(s.Guides) {
		return fmt.Errorf("onto-state: guides %q is not one of pending|updated|waived:<reason>", s.Guides)
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
	s.SchemaVersion = CurrentSchemaVersion
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
