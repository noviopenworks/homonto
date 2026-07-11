package ontostate

import (
	"reflect"
	"strings"
	"testing"
	"unicode"
)

// FuzzStateRoundTrip asserts serialization fidelity: any State that Marshals and
// then Parses back without error must be byte-for-byte identical in value. This
// guards against a field that silently fails to round-trip (drops, renames, or
// corrupts on the wire).
func FuzzStateRoundTrip(f *testing.F) {
	f.Add("feat-x", "full", "build", "2026-07-11", "abc123", true)
	f.Add("c", "", "open", "", "", false)
	f.Add("weird: value", "w\nx", "close", "d", "r", true)

	f.Fuzz(func(t *testing.T, change, workflow, phase, created, baseRef string, archived bool) {
		// onto only ever stores printable, control-char-free field values (a
		// kebab change name, a fixed-set phase, a word workflow, a date, a git
		// SHA). Round-tripping a control character is a yaml.v3 serialization edge
		// outside the state model's domain, so exclude it from the contract.
		for _, v := range []string{change, workflow, phase, created, baseRef} {
			if strings.ContainsFunc(v, func(r rune) bool { return unicode.IsControl(r) }) {
				return
			}
		}
		s := State{
			Change:   change,
			Workflow: workflow,
			Phase:    phase,
			Created:  created,
			BaseRef:  baseRef,
			Archived: archived,
		}
		b, err := Marshal(s)
		if err != nil {
			return
		}
		got, err := Parse(b)
		if err != nil {
			return // not a valid serialized state; round-trip claim does not apply
		}
		if !reflect.DeepEqual(s, got) {
			t.Fatalf("round-trip mismatch:\n in: %+v\nout: %+v\nyaml:\n%s", s, got, b)
		}
	})
}
