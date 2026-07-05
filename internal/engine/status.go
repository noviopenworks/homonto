package engine

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Drift returns lines describing on-disk managed values that diverge from the
// last-applied snapshot. It re-uses each adapter's Plan: an update on a key the
// state already recorded is drift. Secret-key drift is reported without printing
// the value (Plan already redacts it).
func (e *Engine) Drift() ([]string, error) {
	sets, err := e.Plan()
	if err != nil {
		return nil, err
	}
	var lines []string
	for _, cs := range sets {
		for _, c := range cs.Changes {
			if _, ok := e.State.Get(cs.Tool, c.Key); !ok {
				continue
			}
			switch c.Action {
			case "update":
				lines = append(lines, fmt.Sprintf("%s %s drifted (will reset on apply)", cs.Tool, c.Key))
			case "create":
				// A create on a state-recorded key means the managed value was
				// deleted out of band — that is drift too, not "No drift".
				lines = append(lines, fmt.Sprintf("%s %s missing (will recreate on apply)", cs.Tool, c.Key))
			}
		}
	}
	return lines, nil
}

// Doctor runs environment health checks.
func (e *Engine) Doctor() []string {
	var out []string
	if _, err := exec.LookPath("pass"); err != nil {
		out = append(out, "warn: `pass` not found on PATH (pass: references will fail)")
	} else {
		out = append(out, "ok: pass found")
	}
	for _, loc := range []struct{ label, path string }{
		{".claude (Claude Code)", filepath.Join(e.Home, ".claude")},
		{".config/opencode (OpenCode)", filepath.Join(e.Home, ".config", "opencode")},
	} {
		if _, err := os.Stat(loc.path); err != nil {
			out = append(out, fmt.Sprintf("warn: %s config location %s not found", loc.label, loc.path))
		} else {
			out = append(out, fmt.Sprintf("ok: %s config location present", loc.label))
		}
	}
	for _, name := range e.Cfg.Skills.Own {
		p := filepath.Join(e.ContentDir, "skills", name)
		if _, err := os.Stat(p); err != nil {
			out = append(out, fmt.Sprintf("warn: skill %q missing from %s", name, p))
			continue
		}
		// Content alone is not enough — the tool only sees the skill through
		// its symlink, so verify the link exists and points at the content.
		dst := filepath.Join(e.Home, ".claude", "skills", name)
		if target, err := os.Readlink(dst); err == nil && target == p {
			out = append(out, fmt.Sprintf("ok: skill %q linked", name))
		} else {
			out = append(out, fmt.Sprintf("warn: skill %q content present, not linked (run apply)", name))
		}
	}
	return out
}
