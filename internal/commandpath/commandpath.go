// Package commandpath is the single source of truth for where each tool's owned
// commands are linked, as a function of the install scope. It parallels
// skillpath; a future change may unify both into a resourcepath.Dir(kind, …).
// Scope flipping (for inactive-scope pruning) reuses skillpath.Other, so this
// package exposes only Dir.
package commandpath

import "path/filepath"

// Dir returns the directory a tool's owned commands are linked into.
//
//	claude   + user     -> <home>/.claude/commands
//	claude   + project  -> <projectRoot>/.claude/commands
//	opencode + user     -> <home>/.config/opencode/command
//	opencode + project  -> <projectRoot>/.opencode/command
//
// OpenCode uses the SINGULAR "command" directory (unlike its plural "skills").
// Any scope other than "project" is treated as "user". An unknown tool
// returns "".
func Dir(tool, scope, home, projectRoot string) string {
	project := scope == "project"
	switch tool {
	case "claude":
		if project {
			return filepath.Join(projectRoot, ".claude", "commands")
		}
		return filepath.Join(home, ".claude", "commands")
	case "opencode":
		if project {
			return filepath.Join(projectRoot, ".opencode", "command")
		}
		return filepath.Join(home, ".config", "opencode", "command")
	}
	return ""
}
