// Package conformance holds a shared, table-driven conformance suite that
// exercises the adapter.Adapter contract uniformly against every registered
// adapter, rather than relying only on per-adapter ad-hoc tests. A new adapter
// (or a regression in an existing one) that diverges from the core contract is
// caught here. The first slice (ROADMAP E3 / finding F55) asserted the create /
// observe-clean / idempotent-replan / unmanaged-preservation core. This second
// slice extends the same shared harness with two further properties every
// adapter must honor: drift detection + reset (an out-of-band change to a
// managed file is seen by ObserveHashes and reset by a re-Apply) and
// malformed-doc safety (a pre-existing malformed tool document never panics
// Plan or Apply — they error or recover).
package conformance

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/noviopenworks/homonto/internal/adapter"
	"github.com/noviopenworks/homonto/internal/adapter/claude"
	"github.com/noviopenworks/homonto/internal/adapter/opencode"
	"github.com/noviopenworks/homonto/internal/config"
	"github.com/noviopenworks/homonto/internal/jsonutil"
	"github.com/noviopenworks/homonto/internal/secret"
	"github.com/noviopenworks/homonto/internal/state"
)

// adapterCase describes one adapter under test. Every field mirrors the exact
// construction path the adapter's own _test.go files use, so the shared harness
// drives real adapters rather than a re-invented setup.
type adapterCase struct {
	name string
	// newAdapter builds the adapter at user scope, given a $HOME root and a
	// content dir (owned skills). Mirrors each package's New(home, content).
	newAdapter func(home, content string) adapter.Adapter
	// newConfig declares at least one simple managed resource the adapter
	// projects (an MCP targeting the tool plus a setting), so a fresh Plan
	// yields creates.
	newConfig func() *config.Config
	// seed pre-creates the adapter's base config files when the adapter expects
	// them to exist (mirrors the dominant per-adapter test setup). May be nil.
	seed func(t *testing.T, home string)
	// managedDir is the adapter's managed target tree, where the
	// unmanaged-preservation check plants its file (claude -> ~/.claude,
	// opencode -> ~/.config/opencode).
	managedDir func(home string) string
	// driftKey is the state key whose backing on-disk value driftMutate changes
	// out-of-band; the drift check asserts ObserveHashes flags it and a re-Apply
	// resets it. Leave driftMutate nil to skip the drift check for this adapter
	// (the check t.Skips with an explanation).
	driftKey string
	// driftMutate changes the bytes of the file that backs driftKey, out-of-band
	// (as a user hand-editing the tool's config would). It must alter driftKey's
	// value away from the managed value so ObserveHashes reports drift.
	driftMutate func(t *testing.T, home string)
	// malformed plants a pre-existing MALFORMED (unparseable) tool document where
	// the adapter reads/writes, before any Plan. The malformed-doc check then runs
	// Plan+Apply and asserts neither panics. Leave nil to skip (with explanation).
	malformed func(t *testing.T, home string)
}

// noSecret is a resolver with no secret material; the conformance configs
// declare no ${...} secret tokens, so resolution is a no-op.
func noSecret() *secret.Resolver {
	return &secret.Resolver{Getenv: os.Getenv, Pass: func(string) (string, error) { return "", nil }}
}

func cases() []adapterCase {
	return []adapterCase{
		{
			name:       "claude",
			newAdapter: func(home, content string) adapter.Adapter { return claude.New(home, content) },
			newConfig: func() *config.Config {
				return &config.Config{
					MCPs: map[string]config.MCP{
						"codegraph": {Command: []string{"codegraph", "serve"}, Targets: []string{"claude"}},
					},
					Settings: config.Settings{Claude: map[string]any{"model": "opus"}},
				}
			},
			seed: func(t *testing.T, home string) {
				t.Helper()
				mustWrite(t, filepath.Join(home, ".claude.json"), "{}")
				mustMkdir(t, filepath.Join(home, ".claude"))
				mustWrite(t, filepath.Join(home, ".claude", "settings.json"), "{}")
			},
			managedDir: func(home string) string { return filepath.Join(home, ".claude") },
			// setting.model lives on disk as the "model" key in ~/.claude/settings.json.
			driftKey: "setting.model",
			driftMutate: func(t *testing.T, home string) {
				t.Helper()
				// Hand-edit the managed setting to a different value out-of-band.
				mustWrite(t, filepath.Join(home, ".claude", "settings.json"), `{"model":"sonnet"}`)
			},
			malformed: func(t *testing.T, home string) {
				t.Helper()
				// A pre-existing, unparseable ~/.claude.json (truncated object).
				mustWrite(t, filepath.Join(home, ".claude.json"), `{"mcpServers": {`)
			},
		},
		{
			name:       "opencode",
			newAdapter: func(home, content string) adapter.Adapter { return opencode.New(home, content) },
			newConfig: func() *config.Config {
				return &config.Config{
					MCPs: map[string]config.MCP{
						"codegraph": {Command: []string{"codegraph", "serve"}, Targets: []string{"opencode"}},
					},
					Settings: config.Settings{OpenCode: map[string]any{"theme": "dark"}},
				}
			},
			seed:       nil, // opencode.Apply creates ~/.config/opencode/opencode.jsonc on a real create.
			managedDir: func(home string) string { return filepath.Join(home, ".config", "opencode") },
			// setting.theme lives on disk as the "theme" key in ~/.config/opencode/opencode.jsonc.
			driftKey: "setting.theme",
			driftMutate: func(t *testing.T, home string) {
				t.Helper()
				p := filepath.Join(home, ".config", "opencode", "opencode.jsonc")
				raw, err := os.ReadFile(p)
				if err != nil {
					t.Fatalf("read opencode config for drift mutation: %v", err)
				}
				doc, err := jsonutil.Standardize(raw)
				if err != nil {
					t.Fatalf("standardize opencode config: %v", err)
				}
				out, err := jsonutil.SetJSON(doc, "theme", "light")
				if err != nil {
					t.Fatalf("set theme out-of-band: %v", err)
				}
				mustWrite(t, p, string(out))
			},
			malformed: func(t *testing.T, home string) {
				t.Helper()
				dir := filepath.Join(home, ".config", "opencode")
				mustMkdir(t, dir)
				// A pre-existing, unparseable opencode.jsonc (dangling value).
				mustWrite(t, filepath.Join(dir, "opencode.jsonc"), `{"theme": }`)
			},
		},
	}
}

// TestAdaptersPassCoreContract runs the SAME core-contract assertions against
// every adapter in the table:
//
//	(a) Plan on a fresh config + empty state yields at least one "create";
//	(b) Apply writes them without error;
//	(c) ObserveHashes then reports every applied key clean (hash == Entry.Applied);
//	(d) a second Plan (updated state) yields no changes (idempotent);
//	(e) an unmanaged file planted in the managed tree survives Apply byte-for-byte.
func TestAdaptersPassCoreContract(t *testing.T) {
	for _, tc := range cases() {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			home, content := t.TempDir(), t.TempDir()
			if tc.seed != nil {
				tc.seed(t, home)
			}
			a := tc.newAdapter(home, content)
			st, err := state.Load(t.TempDir())
			if err != nil {
				t.Fatalf("state.Load: %v", err)
			}

			// (e-setup) Plant an unmanaged file in the managed tree BEFORE apply.
			managedDir := tc.managedDir(home)
			mustMkdir(t, managedDir)
			unmanagedPath := filepath.Join(managedDir, "unmanaged-keep.txt")
			const unmanagedBody = "hand-written, homonto must not touch this\n"
			mustWrite(t, unmanagedPath, unmanagedBody)

			// (a) Fresh Plan yields at least one create.
			cs, err := a.Plan(tc.newConfig(), st)
			if err != nil {
				t.Fatalf("plan: %v", err)
			}
			creates := 0
			for _, ch := range cs.Changes {
				if ch.Action == "create" {
					creates++
				}
			}
			if creates == 0 {
				t.Fatalf("expected at least one create on fresh plan, got %+v", cs.Changes)
			}

			// (b) Apply writes without error.
			if err := a.Apply(cs, noSecret(), st); err != nil {
				t.Fatalf("apply: %v", err)
			}

			// (c) Every applied key hashes back to its recorded Entry.Applied.
			obs, err := a.ObserveHashes(st)
			if err != nil {
				t.Fatalf("observe: %v", err)
			}
			keys := st.Keys(tc.name)
			if len(keys) == 0 {
				t.Fatalf("apply recorded no state keys for %q", tc.name)
			}
			for _, key := range keys {
				e, ok := st.Get(tc.name, key)
				if !ok {
					t.Fatalf("key %q missing from state after apply", key)
				}
				h, ok := obs[key]
				if !ok {
					t.Fatalf("applied key %q missing from ObserveHashes (should be clean/on-disk)", key)
				}
				if h != e.Applied {
					t.Fatalf("key %q not clean: observed %q != Applied %q", key, h, e.Applied)
				}
			}

			// (d) Second plan on the updated state is a no-op.
			cs2, err := a.Plan(tc.newConfig(), st)
			if err != nil {
				t.Fatalf("second plan: %v", err)
			}
			for _, ch := range cs2.Changes {
				if ch.Action != "noop" {
					t.Fatalf("second plan not idempotent, got %+v", ch)
				}
			}

			// (e) The unmanaged file still exists with unchanged bytes.
			got, err := os.ReadFile(unmanagedPath)
			if err != nil {
				t.Fatalf("unmanaged file lost after apply: %v", err)
			}
			if string(got) != unmanagedBody {
				t.Fatalf("unmanaged file mutated by apply.\nwant: %q\ngot:  %q", unmanagedBody, string(got))
			}
		})
	}
}

// applyFresh builds the adapter for tc on fresh temp dirs, seeds it, and runs
// one Plan+Apply of tc.newConfig() so every managed key is on disk and recorded
// in state. It returns the adapter, its $HOME, and the populated state — the
// starting point the drift check mutates. It mirrors the core test's own setup.
func applyFresh(t *testing.T, tc adapterCase) (adapter.Adapter, string, *state.State) {
	t.Helper()
	home, content := t.TempDir(), t.TempDir()
	if tc.seed != nil {
		tc.seed(t, home)
	}
	a := tc.newAdapter(home, content)
	st, err := state.Load(t.TempDir())
	if err != nil {
		t.Fatalf("state.Load: %v", err)
	}
	cs, err := a.Plan(tc.newConfig(), st)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if err := a.Apply(cs, noSecret(), st); err != nil {
		t.Fatalf("apply: %v", err)
	}
	return a, home, st
}

// TestAdaptersDetectAndResetDrift runs the drift-detection/reset contract check
// against every adapter: after Plan+Apply of a managed resource,
//
//	(a) the driftKey starts clean (ObserveHashes hash == Entry.Applied);
//	(b) an out-of-band change to its backing file makes ObserveHashes report the
//	    key as differing from Entry.Applied (drift detected);
//	(c) a second Plan yields a non-noop change for that key (a reset/update);
//	(d) after re-Apply, ObserveHashes reports the key clean again (reset to the
//	    managed value).
func TestAdaptersDetectAndResetDrift(t *testing.T) {
	for _, tc := range cases() {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if tc.driftMutate == nil {
				// No adapter in the table currently opts out; a future adapter
				// whose managed values are not file-backed would skip here.
				t.Skipf("adapter %q declares no driftMutate (no file-backed managed key to mutate)", tc.name)
			}
			a, home, st := applyFresh(t, tc)

			// (a) The key we are about to drift starts clean.
			e0, ok := st.Get(tc.name, tc.driftKey)
			if !ok {
				t.Fatalf("drift key %q not recorded in state after apply", tc.driftKey)
			}
			obs0, err := a.ObserveHashes(st)
			if err != nil {
				t.Fatalf("observe (pre-drift): %v", err)
			}
			if obs0[tc.driftKey] != e0.Applied {
				t.Fatalf("precondition: %q not clean before mutation: observed %q != Applied %q",
					tc.driftKey, obs0[tc.driftKey], e0.Applied)
			}

			// Mutate the backing file out-of-band.
			tc.driftMutate(t, home)

			// (b) ObserveHashes now flags the key as drifted.
			obs1, err := a.ObserveHashes(st)
			if err != nil {
				t.Fatalf("observe (post-drift): %v", err)
			}
			h1, ok := obs1[tc.driftKey]
			if !ok {
				t.Fatalf("drifted key %q vanished from ObserveHashes (expected present-but-differing)", tc.driftKey)
			}
			if h1 == e0.Applied {
				t.Fatalf("out-of-band change to %q not detected: observed hash still == Applied %q", tc.driftKey, e0.Applied)
			}

			// (c) A re-Plan proposes a non-noop reset for the drifted key.
			cs, err := a.Plan(tc.newConfig(), st)
			if err != nil {
				t.Fatalf("plan after drift: %v", err)
			}
			var reset *adapter.Change
			for i := range cs.Changes {
				if cs.Changes[i].Key == tc.driftKey && cs.Changes[i].Action != "noop" {
					reset = &cs.Changes[i]
					break
				}
			}
			if reset == nil {
				t.Fatalf("plan after drift proposes no reset for %q: %+v", tc.driftKey, cs.Changes)
			}

			// (d) Re-Apply resets the key; ObserveHashes is clean again.
			if err := a.Apply(cs, noSecret(), st); err != nil {
				t.Fatalf("re-apply: %v", err)
			}
			e2, ok := st.Get(tc.name, tc.driftKey)
			if !ok {
				t.Fatalf("drift key %q missing from state after reset", tc.driftKey)
			}
			obs2, err := a.ObserveHashes(st)
			if err != nil {
				t.Fatalf("observe (post-reset): %v", err)
			}
			h2, ok := obs2[tc.driftKey]
			if !ok {
				t.Fatalf("reset key %q missing from ObserveHashes (should be clean/on-disk)", tc.driftKey)
			}
			if h2 != e2.Applied {
				t.Fatalf("re-apply did not reset %q to clean: observed %q != Applied %q", tc.driftKey, h2, e2.Applied)
			}
		})
	}
}

// TestAdaptersSurviveMalformedDoc runs the malformed-doc safety contract check
// against every adapter: a pre-existing, unparseable tool document sitting where
// the adapter reads/writes must not crash Plan or Apply. Returning an error is
// the expected, acceptable outcome; recovering is acceptable; a panic is a
// conformance failure. The deferred recover converts any panic into a clear
// fatal instead of aborting the whole test binary.
func TestAdaptersSurviveMalformedDoc(t *testing.T) {
	for _, tc := range cases() {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if tc.malformed == nil {
				t.Skipf("adapter %q declares no malformed fixture (no single-doc read/write path to corrupt)", tc.name)
			}
			home, content := t.TempDir(), t.TempDir()
			tc.malformed(t, home)
			a := tc.newAdapter(home, content)
			st, err := state.Load(t.TempDir())
			if err != nil {
				t.Fatalf("state.Load: %v", err)
			}

			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("adapter %q panicked on a malformed tool doc (must error or recover, never panic): %v", tc.name, r)
				}
			}()

			cs, planErr := a.Plan(tc.newConfig(), st)
			if planErr != nil {
				// Erroring on a malformed doc is the expected, safe outcome.
				return
			}
			// Plan tolerated the malformed doc; Apply must also not panic. An
			// error from Apply is acceptable too — only a panic fails the check.
			_ = a.Apply(cs, noSecret(), st)
		})
	}
}

func mustWrite(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func mustMkdir(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
}
