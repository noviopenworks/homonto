package ontocli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/noviopenworks/homonto/internal/buildinfo"
	"github.com/noviopenworks/homonto/internal/ontostate"
	"github.com/spf13/cobra"
)

// homontoAppliedVersion reads only the homontoVersion field from
// <root>/.homonto/state.json, returning "" for any absence or parse error. It
// deliberately does NOT import homonto's state package, keeping onto decoupled
// from the projection side (the doctor reads one opaque JSON field).
func homontoAppliedVersion(root string) string {
	data, err := os.ReadFile(filepath.Join(root, ".homonto", "state.json"))
	if err != nil {
		return ""
	}
	var s struct {
		HomontoVersion string `json:"homontoVersion"`
	}
	if json.Unmarshal(data, &s) != nil {
		return ""
	}
	return s.HomontoVersion
}

// normalizeVersion strips a leading "v" and any build metadata (from "+") so a
// dirty local build of both binaries compares equal on its release core.
func normalizeVersion(v string) string {
	v = strings.TrimPrefix(v, "v")
	if i := strings.IndexByte(v, '+'); i >= 0 {
		v = v[:i]
	}
	return v
}

// ErrQuietFindings is what `onto doctor --quiet` returns when there are
// findings: the caller (cmd/onto/main.go) must exit non-zero WITHOUT printing —
// quiet mode's whole contract is "exit code only", and hooks that capture
// stderr were getting a leaked error line.
var ErrQuietFindings = errors.New("onto doctor: findings (quiet)")

// doctorCmd builds the "onto doctor" subcommand: a strictly read-only,
// config-independent workspace-health diagnostic. Unlike init/new/close it is
// NOT gated on the framework install — a missing docs layout is a finding, not
// a refusal. It writes nothing and imports none of homonto's projection
// packages.
func doctorCmd() *cobra.Command {
	var (
		dir   string
		quiet bool
	)

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Report onto workflow/project health (read-only)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if quiet {
				// Hook-friendly: suppress ALL output, communicate only via exit
				// code (non-zero when there are findings). Used by an editor/tool
				// Stop hook to fail loudly on a workflow-integrity problem.
				// SilenceErrors/SilenceUsage stop cobra's own printing; main.go
				// additionally recognizes the quiet sentinel so its error line is
				// suppressed too — --quiet previously still leaked
				// "error: onto doctor: N problem(s) found" to stderr.
				cmd.SetOut(io.Discard)
				cmd.SilenceErrors = true
				cmd.SilenceUsage = true
				if err := runDoctor(cmd, dir); err != nil {
					return ErrQuietFindings
				}
				return nil
			}
			return runDoctor(cmd, dir)
		},
	}
	cmd.Flags().StringVar(&dir, "dir", ".", "workspace root to inspect")
	cmd.Flags().BoolVar(&quiet, "quiet", false, "print nothing; signal health via exit code only (for hooks)")
	return cmd
}

// runDoctor accumulates health findings in a fixed order — docs layout, active
// changes, archive layout — printing each to stdout. It performs zero writes
// and never calls gate(). On a healthy workspace it prints "healthy" and
// returns nil; otherwise it prints every finding and returns a summary error so
// main exits non-zero.
func runDoctor(cmd *cobra.Command, root string) error {
	var findings []string

	// 1. docs layout: every directory in docsLayout must exist as a directory.
	for _, d := range docsLayout {
		info, err := os.Stat(filepath.Join(root, d))
		if err != nil || !info.IsDir() {
			findings = append(findings, "docs layout: missing directory "+d)
		}
	}

	// 2. active changes: enumerate change directories first (excluding
	// archive/), then classify. A missing-state or malformed directory is a
	// finding — a deleted state file is reported, never silently skipped (F14).
	changesDir := filepath.Join(root, "docs", "changes")
	if entries, readErr := os.ReadDir(changesDir); readErr == nil {
		for _, e := range entries {
			if !e.IsDir() || e.Name() == "archive" {
				continue
			}
			name := e.Name()
			changeDir := filepath.Join(changesDir, name)
			st, class, classErr := ontostate.Classify(changeDir)
			switch class {
			case "missing-state":
				findings = append(findings, name+": missing-state (change directory has no state file)")
				continue
			case "malformed":
				findings = append(findings, fmt.Sprintf("%s: malformed state: %v", name, classErr))
				continue
			}
			// An abandoned change is a parked terminal state, not a health
			// problem: its missing artifacts, unresolved deps, and verify-round
			// count are exactly why it was abandoned. Counting them made a
			// `doctor --quiet` Stop hook fail forever with no clearing action.
			if st.Abandoned {
				continue
			}
			phase := st.Phase
			if skErr := ontostate.ValidateSkeleton(changeDir); skErr != nil {
				findings = append(findings, fmt.Sprintf("%s: phase %s missing artifact: %v", name, phase, skErr))
			}
			if unresolved := ontostate.DepsResolved(root, st.Deps); len(unresolved) > 0 {
				findings = append(findings, fmt.Sprintf("%s: unresolved dependencies: %v", name, unresolved))
			}
			if st.Archived {
				findings = append(findings, name+": active change marked archived: true (belongs under docs/changes/archive/)")
			}
			// A change that has failed verification 3+ times needs a decision, not
			// another silent retry (accept the deviation or keep fixing).
			if st.Observed.VerifyRounds >= 3 {
				findings = append(findings, fmt.Sprintf("%s: %d failed verify rounds — decide accept-deviation or continue", name, st.Observed.VerifyRounds))
			}
		}
	}

	// 3. archive layout: each archive/<name> directory must hold a valid
	// onto-state.yaml marked archived:true. Stray non-directory entries are
	// ignored.
	entries, _ := filepath.Glob(filepath.Join(root, "docs", "changes", "archive", "*"))
	for _, entry := range entries {
		info, err := os.Stat(entry)
		if err != nil || !info.IsDir() {
			continue
		}
		name := filepath.Base(entry)
		st, err := ontostate.Load(filepath.Join(entry, "onto-state.yaml"))
		if err != nil {
			findings = append(findings, fmt.Sprintf("archive/%s: invalid or missing onto-state.yaml: %v", name, err))
			continue
		}
		if !st.Archived {
			findings = append(findings, "archive/"+name+": not marked archived: true")
		}
	}

	// 4. version skew: the onto binary and the homonto that projected the onto
	// framework are released together and should match. When they diverge, the
	// installed skills/commands may not match this binary's behavior — tell the
	// user to re-sync. Best-effort and boundary-preserving: read only the
	// homontoVersion field from .homonto/state.json (no import of homonto's
	// projection packages); a missing file or field is silently skipped, and
	// build metadata (+dirty, etc.) is ignored so a homogeneous dev build of both
	// binaries does not report a false skew.
	if applied := homontoAppliedVersion(root); applied != "" {
		onto := buildinfo.Resolve(Version, devVersion)
		if onto != "" && normalizeVersion(onto) != normalizeVersion(applied) {
			findings = append(findings, fmt.Sprintf(
				"version skew: onto %s, but the onto framework was last applied by homonto %s — run `homonto update` (or align the two binaries)",
				onto, applied))
		}
	}

	// verdict
	if len(findings) == 0 {
		cmd.Println("healthy")
		return nil
	}
	for _, f := range findings {
		cmd.Println(f)
	}
	return fmt.Errorf("onto doctor: %d problem(s) found", len(findings))
}
