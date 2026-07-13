// Package conformance holds a shared, table-driven conformance suite that
// exercises the adapter.Adapter contract uniformly against every registered
// adapter, rather than relying only on per-adapter ad-hoc tests. A new adapter
// (or a regression in an existing one) that diverges from the core contract is
// caught here. This is the first slice (ROADMAP E3 / finding F55): it asserts
// the create / observe-clean / idempotent-replan / unmanaged-preservation core.
package conformance

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/noviopenworks/homonto/internal/adapter"
	"github.com/noviopenworks/homonto/internal/adapter/claude"
	"github.com/noviopenworks/homonto/internal/adapter/opencode"
	"github.com/noviopenworks/homonto/internal/config"
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
