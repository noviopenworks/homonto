// Package subagentpath is the single source of truth for where each tool's
// owned subagents are linked, as a function of the install scope. It parallels
// commandpath/skillpath; a future change may unify them into a
// resourcepath.Dir(kind, …). Scope flipping (for inactive-scope pruning) reuses
// skillpath.Other, so this package exposes only Dir.
package subagentpath

import "path/filepath"

// Dir returns the directory a tool's owned subagents are linked into.
//
//	claude   + user     -> <home>/.claude/agents
//	claude   + project  -> <projectRoot>/.claude/agents
//	opencode + user     -> <home>/.config/opencode/agent
//	opencode + project  -> <projectRoot>/.opencode/agent
//
// Claude Code uses the PLURAL "agents" directory at both scopes; OpenCode uses
// the SINGULAR "agent" directory at both scopes (consistent with its singular
// "command"). Any scope other than "project" is treated as "user". An unknown
// tool returns "".
func Dir(tool, scope, home, projectRoot string) string {
	project := scope == "project"
	switch tool {
	case "claude":
		if project {
			return filepath.Join(projectRoot, ".claude", "agents")
		}
		return filepath.Join(home, ".claude", "agents")
	case "opencode":
		if project {
			return filepath.Join(projectRoot, ".opencode", "agent")
		}
		return filepath.Join(home, ".config", "opencode", "agent")
	}
	return ""
}
