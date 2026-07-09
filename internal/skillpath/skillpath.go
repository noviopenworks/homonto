// Package skillpath is the single source of truth for where each tool's owned
// skills are linked, as a function of the install scope. Both the adapters and
// the engine's doctor call Dir so the path convention lives in exactly one
// place — important because the tools disagree on their subpaths and the two
// scopes do not share a common base directory.
package skillpath

import "path/filepath"

// Dir returns the directory a tool's owned skills are linked into.
//
//	claude   + user     -> <home>/.claude/skills
//	claude   + project  -> <projectRoot>/.claude/skills
//	opencode + user     -> <home>/.config/opencode/skills
//	opencode + project  -> <projectRoot>/.opencode/skills
//
// Any scope other than "project" is treated as "user" (config.Load rejects
// empty/invalid scope; this fallback only guards against an unnormalized value
// reaching here). An unknown tool returns "".
func Dir(tool, scope, home, projectRoot string) string {
	project := scope == "project"
	switch tool {
	case "claude":
		if project {
			return filepath.Join(projectRoot, ".claude", "skills")
		}
		return filepath.Join(home, ".claude", "skills")
	case "opencode":
		if project {
			// OpenCode reads project skills from <repo>/.opencode/skills — note this
			// differs from its global ~/.config/opencode/skills, so it is not a
			// base-directory swap. https://opencode.ai/docs/skills/
			return filepath.Join(projectRoot, ".opencode", "skills")
		}
		return filepath.Join(home, ".config", "opencode", "skills")
	}
	return ""
}

// Other returns the opposite scope, used to locate a skill's inactive-scope
// link so a scope switch can prune it. "project" maps to "user"; every other
// value (including "user" and "") maps to "project".
func Other(scope string) string {
	if scope == "project" {
		return "user"
	}
	return "project"
}
