package adapter

import "testing"

func TestAction_Valid(t *testing.T) {
	for _, a := range []Action{ActionCreate, ActionUpdate, ActionDelete, ActionNoop, ActionAdopt} {
		if !a.Valid() {
			t.Errorf("%q should be valid", a)
		}
	}
	if Action("bogus").Valid() {
		t.Errorf("bogus should be invalid")
	}
	if Action("").Valid() {
		t.Errorf("empty should be invalid")
	}
}

func TestChangeSet_Validate(t *testing.T) {
	known := map[string]bool{"claude": true, "opencode": true}

	ok := ChangeSet{Tool: "claude", Changes: []Change{{Action: ActionCreate, Key: "mcp.x"}}}
	if err := ok.Validate(known); err != nil {
		t.Errorf("legal set should validate: %v", err)
	}

	unknownTool := ChangeSet{Tool: "ghost", Changes: []Change{{Action: ActionNoop, Key: "k"}}}
	if err := unknownTool.Validate(known); err == nil {
		t.Errorf("unknown tool should error")
	}

	unknownAction := ChangeSet{Tool: "claude", Changes: []Change{{Action: Action("frobnicate"), Key: "k"}}}
	if err := unknownAction.Validate(known); err == nil {
		t.Errorf("unknown action should error")
	}
}
