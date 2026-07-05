package engine

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
)

// Status reports two independent facts about the managed surface:
//
//   - drift: state-recorded keys whose CURRENT on-disk value diverges from the
//     value last written by apply (Entry.Applied), or that are missing from disk
//     entirely. Drift comes ONLY from each adapter's ObserveHashes vs Applied —
//     never from the desired-vs-disk Plan comparison — so a pure homonto.toml
//     edit is never mistaken for disk drift.
//   - pending: visible config changes (create/update/delete) that Plan derived
//     from the current desired config and are still awaiting apply, EXCLUDING
//     any key already accounted for as drift.
//
// Plan also populates e.Warnings; a per-adapter ObserveHashes failure is
// appended there and that tool's keys are skipped rather than failing the run.
func (e *Engine) Status() (drift []string, pending int, err error) {
	sets, err := e.Plan()
	if err != nil {
		return nil, 0, err
	}

	// drifted tracks tool -> key -> true for every key reported as drift, so the
	// pending count can exclude them (a drifted key's Plan change is a reset, not
	// pending config work).
	drifted := map[string]map[string]bool{}
	mark := func(tool, key string) {
		if drifted[tool] == nil {
			drifted[tool] = map[string]bool{}
		}
		drifted[tool][key] = true
	}

	for _, a := range e.Adapters {
		observed, oerr := a.ObserveHashes(e.State)
		if oerr != nil {
			e.Warnings = append(e.Warnings, fmt.Sprintf("%s drift skipped: %v", a.Name(), oerr))
			continue
		}
		for _, key := range e.State.Keys(a.Name()) {
			h, ok := observed[key]
			if !ok {
				drift = append(drift, fmt.Sprintf("%s %s missing (deleted out of band)", a.Name(), key))
				mark(a.Name(), key)
				continue
			}
			entry, _ := e.State.Get(a.Name(), key)
			if h != entry.Applied {
				drift = append(drift, fmt.Sprintf("%s %s drifted (will reset on apply)", a.Name(), key))
				mark(a.Name(), key)
			}
		}
	}

	for _, cs := range sets {
		for _, c := range cs.Changes {
			switch c.Action {
			case "create", "update", "delete":
				if drifted[cs.Tool][c.Key] {
					continue
				}
				pending++
			}
		}
	}

	sort.Strings(drift)
	return drift, pending, nil
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
