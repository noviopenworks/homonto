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
			if c.Action != "update" {
				continue
			}
			if _, ok := e.State.Get(cs.Tool, c.Key); ok {
				lines = append(lines, fmt.Sprintf("%s %s drifted (will reset on apply)", cs.Tool, c.Key))
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
	for _, name := range e.Cfg.Skills.Own {
		p := filepath.Join(e.ContentDir, "skills", name)
		if _, err := os.Stat(p); err != nil {
			out = append(out, fmt.Sprintf("warn: skill %q missing from %s", name, p))
		} else {
			out = append(out, fmt.Sprintf("ok: skill %q present", name))
		}
	}
	return out
}
