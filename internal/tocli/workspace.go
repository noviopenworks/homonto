package tocli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/noviopenworks/homonto/internal/tostate"
	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/cobra"
)

// todayFn returns today's date for created/finished stamps and archive
// prefixes; a variable so tests can pin it.
var todayFn = func() string { return time.Now().Format("2006-01-02") }

// tasksDir/archiveDir are to's territory. Fully disjoint from onto's
// docs/changes/ so a mixed repo never confuses either tool's commands —
// though homonto refuses to declare both frameworks anyway.
func tasksDir(root string) string   { return filepath.Join(root, "docs", "tasks") }
func archiveDir(root string) string { return filepath.Join(root, "docs", "tasks", "archive") }

func changeDir(root, name string) string { return filepath.Join(tasksDir(root), name) }
func statePath(root, name string) string {
	return filepath.Join(changeDir(root, name), tostate.FileName)
}
func planPath(root, name string) string { return filepath.Join(changeDir(root, name), "plan.md") }

// homontoConfig is the minimal shape of homonto.toml that the gate needs:
// just enough to detect whether a [frameworks.to] table is declared. It is
// intentionally a standalone struct, not homonto's own config type, so that
// to stays isolated from homonto's projection pipeline packages.
type homontoConfig struct {
	Frameworks map[string]any `toml:"frameworks"`
}

// gate enforces the framework-install precondition every mutating command
// (init, new, phase, done, abandon) requires: the project must have declared
// and applied [frameworks.to] through Homonto. The skills are the product —
// the gate guarantees no agent works inside the framework without them. It
// checks, in order, and returns on the first failure:
//
//  1. <root>/homonto.toml exists.
//  2. it declares a [frameworks.to] table.
//  3. <root>/.homonto/catalog/skills/to exists as a directory (i.e. the
//     declaration has been applied).
//
// Read-only commands (status, handoff) never call gate. gate performs no
// writes.
func gate(root string) error {
	tomlPath := filepath.Join(root, "homonto.toml")

	data, err := os.ReadFile(tomlPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("to: no homonto.toml found in %s; run `homonto init` first", root)
		}
		return fmt.Errorf("to: reading %s: %w", tomlPath, err)
	}

	var cfg homontoConfig
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("to: parsing %s: %w", tomlPath, err)
	}

	if _, ok := cfg.Frameworks["to"]; !ok {
		return fmt.Errorf("to: %s has no [frameworks.to] table; declare [frameworks.to] and run `homonto apply`", tomlPath)
	}

	catalogPath := filepath.Join(root, ".homonto", "catalog", "skills", "to")
	info, err := os.Stat(catalogPath)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("to: %s not found; run `homonto apply` to install the to framework", catalogPath)
	}

	return nil
}

// changeNamePattern is the accepted shape for a change name: one or more
// lowercase-alphanumeric segments joined by single hyphens. "archive" is
// additionally reserved (it is the archive directory itself).
var changeNamePattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

func validChangeName(name string) error {
	if name == "" {
		return fmt.Errorf("to: change name must not be empty")
	}
	if name != filepath.Base(name) || strings.Contains(name, "..") {
		return fmt.Errorf("to: change name %q must not contain path separators or \"..\"", name)
	}
	if !changeNamePattern.MatchString(name) {
		return fmt.Errorf("to: change name %q must match %s", name, changeNamePattern.String())
	}
	if name == "archive" {
		return fmt.Errorf("to: change name %q is reserved", name)
	}
	return nil
}

// loadChange loads an active (non-archived) change's state, with an error
// that distinguishes "never existed" from "already archived".
func loadChange(root, name string) (tostate.State, error) {
	if err := validChangeName(name); err != nil {
		return tostate.State{}, err
	}
	st, err := tostate.Load(statePath(root, name))
	if err == nil {
		return st, nil
	}
	if archived := findArchived(root, name); archived != "" {
		return tostate.State{}, fmt.Errorf("to: change %q is archived at %s", name, archived)
	}
	return tostate.State{}, err
}

// findArchived returns the newest archive directory holding the named change
// ("" if none). Archive dirs are date-prefixed (<YYYY-MM-DD>-<name>) so a
// name can be reused across changes; pre-v0.5.0 unprefixed dirs are matched
// too.
func findArchived(root, name string) string {
	newest := ""
	if _, err := os.Stat(filepath.Join(archiveDir(root), name)); err == nil {
		newest = filepath.Join(archiveDir(root), name)
	}
	matches, _ := filepath.Glob(filepath.Join(archiveDir(root), "*-"+name))
	for _, m := range matches {
		// The date prefix sorts lexically, so the last match is the newest.
		if newest == "" || filepath.Base(m) > filepath.Base(newest) {
			newest = m
		}
	}
	return newest
}

// archiveDest is where a change finishing on the given date archives to. The
// date prefix frees the change name for reuse (a recurring chore can run
// again next time); a same-day reuse gets a numeric suffix so finishing can
// never collide into a wedge.
func archiveDest(root, name, finished string) string {
	base := filepath.Join(archiveDir(root), finished+"-"+name)
	dest := base
	for n := 2; ; n++ {
		if _, err := os.Stat(dest); os.IsNotExist(err) {
			return dest
		}
		dest = fmt.Sprintf("%s-%d", base, n)
	}
}

// archive moves an active change directory to dest. It refuses to clobber an
// existing archive; callers pre-check the destination before writing terminal
// state so a collision cannot strand a terminal change in the active tree.
func archive(root, name, dest string) (string, error) {
	if _, err := os.Stat(dest); err == nil {
		return "", fmt.Errorf("to: archive destination %s already exists", dest)
	}
	if err := os.MkdirAll(archiveDir(root), 0o755); err != nil {
		return "", fmt.Errorf("to: creating %s: %w", archiveDir(root), err)
	}
	if err := os.Rename(changeDir(root, name), dest); err != nil {
		return "", fmt.Errorf("to: archiving %s: %w", name, err)
	}
	return dest, nil
}

// finishAndArchive is the shared terminal move for done and abandon: pick a
// free archive destination, write the terminal state, then rename. A crash
// (or kill) between the write and the rename leaves a terminal change in the
// active tree; re-running the same command converges it (completeArchive), and
// `to doctor` reports it.
func finishAndArchive(root string, st tostate.State) (string, error) {
	dest := archiveDest(root, st.Change, st.Finished)
	if err := tostate.Save(statePath(root, st.Change), st); err != nil {
		return "", err
	}
	return archive(root, st.Change, dest)
}

// completeArchive finishes the interrupted half of a terminal-but-active
// change: the state already says done/abandoned, only the move into the
// archive is missing.
func completeArchive(root string, st tostate.State) (string, error) {
	finished := st.Finished
	if finished == "" {
		// A wedged pre-Finished state; date the archive by completion instead.
		finished = todayFn()
	}
	return archive(root, st.Change, archiveDest(root, st.Change, finished))
}

// lock takes an exclusive per-workspace lock for a mutating command, so two
// concurrent sessions cannot interleave writes on the same change
// (last-writer-wins with no diagnostic). Same O_EXCL pattern as homonto's
// applylock: portable, fail-fast, and a SIGKILLed holder leaves a lockfile
// whose content names the stale pid for hand cleanup.
func lock(root string) (func(), error) {
	if err := os.MkdirAll(tasksDir(root), 0o755); err != nil {
		return nil, fmt.Errorf("to: lock: %w", err)
	}
	path := filepath.Join(tasksDir(root), ".to.lock")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		if os.IsExist(err) {
			return nil, fmt.Errorf("to: another to command is in progress (lock held at %s); wait for it, or remove the file if none is running", path)
		}
		return nil, fmt.Errorf("to: lock: %w", err)
	}
	fmt.Fprintf(f, "pid=%d\n", os.Getpid())
	_ = f.Close()
	return func() { _ = os.Remove(path) }, nil
}

// printJSON marshals v with indentation to the command's stdout.
func printJSON(cmd *cobra.Command, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("to: encoding json: %w", err)
	}
	cmd.Println(string(b))
	return nil
}
