package engine

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/noviopenworks/homonto/internal/commandpath"
	"github.com/noviopenworks/homonto/internal/config"
	"github.com/noviopenworks/homonto/internal/remote"
	"github.com/noviopenworks/homonto/internal/skillpath"
	"github.com/noviopenworks/homonto/internal/subagentpath"
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
	claudeSkills, cerr := e.Cfg.ExpandedSkillEntriesForTool("claude")
	if cerr != nil {
		out = append(out, fmt.Sprintf("warn: cannot expand claude skills: %v", cerr))
	} else {
		out = append(out, e.doctorSkills("claude", claudeSkills)...)
	}
	opencodeSkills, oerr := e.Cfg.ExpandedSkillEntriesForTool("opencode")
	if oerr != nil {
		out = append(out, fmt.Sprintf("warn: cannot expand opencode skills: %v", oerr))
	} else {
		out = append(out, e.doctorSkills("opencode", opencodeSkills)...)
	}
	claudeCommands, ccerr := e.Cfg.ExpandedCommandEntriesForTool("claude")
	if ccerr != nil {
		out = append(out, fmt.Sprintf("warn: cannot expand claude commands: %v", ccerr))
	} else {
		out = append(out, e.doctorCommands("claude", claudeCommands)...)
	}
	opencodeCommands, ocerr := e.Cfg.ExpandedCommandEntriesForTool("opencode")
	if ocerr != nil {
		out = append(out, fmt.Sprintf("warn: cannot expand opencode commands: %v", ocerr))
	} else {
		out = append(out, e.doctorCommands("opencode", opencodeCommands)...)
	}
	claudeSubagents, csaerr := e.Cfg.ExpandedSubagentEntriesForTool("claude")
	if csaerr != nil {
		out = append(out, fmt.Sprintf("warn: cannot expand claude subagents: %v", csaerr))
	} else {
		out = append(out, e.doctorSubagents("claude", claudeSubagents)...)
	}
	opencodeSubagents, osaerr := e.Cfg.ExpandedSubagentEntriesForTool("opencode")
	if osaerr != nil {
		out = append(out, fmt.Sprintf("warn: cannot expand opencode subagents: %v", osaerr))
	} else {
		out = append(out, e.doctorSubagents("opencode", opencodeSubagents)...)
	}
	out = append(out, e.doctorRemoteDigests()...)
	return out
}

// doctorRemoteDigests verifies that each locked remote subagent's materialized
// content still matches its pinned digest, catching on-disk tampering that the
// name-based link check cannot see (F30). It re-hashes the cache-backed content
// and compares the active materialized file's bytes against it.
func (e *Engine) doctorRemoteDigests() []string {
	lock, err := remote.LoadLock(e.remoteLockPath())
	if err != nil {
		return []string{fmt.Sprintf("warn: remote lock unreadable: %v", err)}
	}
	cache := &remote.Cache{Root: e.RemoteCacheRoot}
	var names []string
	for _, entry := range lock.Entries {
		if entry.Kind == "subagent" {
			names = append(names, entry.Name)
		}
	}
	sort.Strings(names)
	var out []string
	for _, name := range names {
		entry, _ := lock.Get("subagent", name)
		pin, perr := remote.ParseDigest(entry.Digest)
		if perr != nil {
			out = append(out, fmt.Sprintf("warn: remote subagent %q: unparseable locked digest: %v", name, perr))
			continue
		}
		if verr := cache.VerifyContent(pin); verr != nil {
			out = append(out, fmt.Sprintf("warn: remote subagent %q: %v", name, verr))
			continue
		}
		active := filepath.Join(e.remoteSubagentDir(), name+".md")
		ab, aerr := os.ReadFile(active)
		if aerr != nil {
			out = append(out, fmt.Sprintf("warn: remote subagent %q: materialized content missing (%v)", name, aerr))
			continue
		}
		cb, cerr := os.ReadFile(filepath.Join(cache.Dir(pin), name+".md"))
		if cerr != nil {
			out = append(out, fmt.Sprintf("warn: remote subagent %q: cached content missing (%v)", name, cerr))
			continue
		}
		if !bytes.Equal(ab, cb) {
			out = append(out, fmt.Sprintf("warn: remote subagent %q: materialized content does not match locked digest %s", name, pin))
			continue
		}
		out = append(out, fmt.Sprintf("ok: remote subagent %q digest verified", name))
	}
	return out
}

// doctorSkills reports, per skill, whether its content is present at the right
// source (builtin: from the materialized catalog, local: from the content dir)
// and whether it is linked into the tool's skills directory.
func (e *Engine) doctorSkills(tool string, entries []config.NamedResource) []string {
	var out []string
	for _, entry := range entries {
		name := entry.Name
		var p string
		if strings.HasPrefix(entry.Resource.Source, "builtin:") {
			p = filepath.Join(e.CatalogDir(), strings.TrimPrefix(entry.Resource.Source, "builtin:"))
		} else {
			sourceName := name
			if strings.HasPrefix(entry.Resource.Source, "local:") {
				sourceName = strings.TrimPrefix(entry.Resource.Source, "local:")
			}
			p = filepath.Join(e.ContentDir, "skills", sourceName)
		}
		if _, err := os.Stat(p); err != nil {
			out = append(out, fmt.Sprintf("warn: skill %q missing from %s (run apply)", name, p))
			continue
		}
		dst := filepath.Join(skillpath.Dir(tool, entry.Resource.Scope, e.Home, e.ProjectRoot), name)
		if target, err := os.Readlink(dst); err == nil && target == p {
			out = append(out, fmt.Sprintf("ok: skill %q linked (%s)", name, tool))
		} else {
			out = append(out, fmt.Sprintf("warn: skill %q content present, not linked for %s (run apply)", name, tool))
		}
	}
	return out
}

// doctorCommands reports, per command, whether its content file is present at
// the right source (builtin: from the materialized command root, local: from
// the content dir) and whether it is linked into the tool's command directory.
func (e *Engine) doctorCommands(tool string, entries []config.NamedResource) []string {
	var out []string
	for _, entry := range entries {
		name := entry.Name
		var p string
		if strings.HasPrefix(entry.Resource.Source, "builtin:") {
			p = filepath.Join(e.CommandDir(), strings.TrimPrefix(entry.Resource.Source, "builtin:")+".md")
		} else {
			sourceName := name
			if strings.HasPrefix(entry.Resource.Source, "local:") {
				sourceName = strings.TrimPrefix(entry.Resource.Source, "local:")
			}
			p = filepath.Join(e.ContentDir, "commands", sourceName+".md")
		}
		if _, err := os.Stat(p); err != nil {
			out = append(out, fmt.Sprintf("warn: command %q missing from %s (run apply)", name, p))
			continue
		}
		dst := filepath.Join(commandpath.Dir(tool, entry.Resource.Scope, e.Home, e.ProjectRoot), name+".md")
		if target, err := os.Readlink(dst); err == nil && target == p {
			out = append(out, fmt.Sprintf("ok: command %q linked (%s)", name, tool))
		} else {
			out = append(out, fmt.Sprintf("warn: command %q content present, not linked for %s (run apply)", name, tool))
		}
	}
	return out
}

// doctorSubagents reports, per subagent, whether its content file is present at
// the right source (builtin: from the materialized subagent root, local: from
// the content dir) and whether it is linked into the tool's agent directory.
func (e *Engine) doctorSubagents(tool string, entries []config.NamedResource) []string {
	var out []string
	for _, entry := range entries {
		name := entry.Name
		var p string
		switch {
		case strings.HasPrefix(entry.Resource.Source, "builtin:"):
			p = filepath.Join(e.SubagentDir(), strings.TrimPrefix(entry.Resource.Source, "builtin:")+".md")
		case strings.HasPrefix(entry.Resource.Source, "remote:"):
			p = filepath.Join(e.remoteSubagentDir(), name+".md")
		default:
			sourceName := name
			if strings.HasPrefix(entry.Resource.Source, "local:") {
				sourceName = strings.TrimPrefix(entry.Resource.Source, "local:")
			}
			p = filepath.Join(e.ContentDir, "subagents", sourceName+".md")
		}
		if _, err := os.Stat(p); err != nil {
			out = append(out, fmt.Sprintf("warn: subagent %q missing from %s (run apply)", name, p))
			continue
		}
		dst := filepath.Join(subagentpath.Dir(tool, entry.Resource.Scope, e.Home, e.ProjectRoot), name+".md")
		if target, err := os.Readlink(dst); err == nil && target == p {
			out = append(out, fmt.Sprintf("ok: subagent %q linked (%s)", name, tool))
		} else {
			out = append(out, fmt.Sprintf("warn: subagent %q content present, not linked for %s (run apply)", name, tool))
		}
	}
	return out
}
