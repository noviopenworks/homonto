package ontostate

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// legacyState is a permissive superset that captures BOTH legacy shapes: the
// flat 7-field binary onto-state.yaml and the rich skill state.yaml. Fields
// absent from a given file simply stay zero.
type legacyState struct {
	Change   string   `yaml:"change"`
	Workflow string   `yaml:"workflow"`
	Phase    string   `yaml:"phase"`
	Created  string   `yaml:"created"`
	BaseRef  string   `yaml:"base_ref"`
	Deps     []string `yaml:"deps"`
	Archived bool     `yaml:"archived"`

	Decisions struct {
		Isolation string `yaml:"isolation"`
		Execution string `yaml:"execution"`
		TDD       string `yaml:"tdd"`
		Directive string `yaml:"directive"`
	} `yaml:"decisions"`

	Verify struct {
		Mode   string `yaml:"mode"`
		Result string `yaml:"result"`
	} `yaml:"verify"`

	Close struct {
		Merged bool `yaml:"merged"`
	} `yaml:"close"`

	Metrics struct {
		Phases       map[string]string `yaml:"phases"`
		TasksTotal   int               `yaml:"tasks_total"`
		VerifyRounds int               `yaml:"verify_rounds"`
		Upgraded     bool              `yaml:"upgraded"`
	} `yaml:"metrics"`
}

// isLegacy reports whether b carries no schema_version key.
func isLegacy(b []byte) bool {
	var probe struct {
		SchemaVersion *int `yaml:"schema_version"`
	}
	if err := yaml.Unmarshal(b, &probe); err != nil {
		return false // let parseAndMigrate surface the parse error
	}
	return probe.SchemaVersion == nil
}

// migrateLegacy maps a legacy shape onto the current State. Fields the legacy
// file lacks map to their zero value, which the schema treats as "unset".
// Renames: execution->build_mode, tdd->tdd_mode, verify.mode->verify.scale,
// metrics.upgraded->preset_escalated. The observational-only "guides" field is
// intentionally not carried (change B re-derives it; it is never gated).
func migrateLegacy(l legacyState) State {
	return State{
		SchemaVersion: CurrentSchemaVersion,
		Change:        l.Change,
		Workflow:      l.Workflow,
		Phase:         l.Phase,
		Created:       l.Created,
		BaseRef:       l.BaseRef,
		Deps:          l.Deps,
		Isolation:     l.Decisions.Isolation,
		BuildMode:     l.Decisions.Execution,
		TDDMode:       l.Decisions.TDD,
		Verify:        Verify{Scale: l.Verify.Mode, Result: l.Verify.Result},
		Close:         Close{Merged: l.Close.Merged},
		Directive:     l.Decisions.Directive,
		Archived:      l.Archived,
		Observed: Observed{
			Metrics:         l.Metrics.Phases,
			TasksTotal:      l.Metrics.TasksTotal,
			VerifyRounds:    l.Metrics.VerifyRounds,
			PresetEscalated: l.Metrics.Upgraded,
		},
	}
}

// parseAndMigrate decodes raw state bytes, up-migrating a legacy (unversioned)
// shape to the current schema. A current-version doc is decoded as-is. The
// result always carries the current schema_version. sourceName names the file
// in any error. Migration is ordered and idempotent.
func parseAndMigrate(b []byte, sourceName string) (State, error) {
	if isLegacy(b) {
		var l legacyState
		if err := yaml.Unmarshal(b, &l); err != nil {
			return State{}, fmt.Errorf("onto-state: %s: %w", sourceName, err)
		}
		return migrateLegacy(l), nil
	}
	st, err := Parse(b)
	if err != nil {
		return State{}, err
	}
	st.SchemaVersion = CurrentSchemaVersion
	return st, nil
}
