package ontocli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"

	"github.com/noviopenworks/homonto/internal/ontostate"
	"github.com/spf13/cobra"
)

// graphNode is one node in the traceability graph — a change (kind "change",
// carrying id/phase/archived) or a capability (kind "capability", named in the
// Change field with id/phase empty).
type graphNode struct {
	ID       string `json:"id"`
	Change   string `json:"change"`
	Phase    string `json:"phase"`
	Archived bool   `json:"archived"`
	Kind     string `json:"kind"`
}

// graphEdge is a typed relationship between changes. Edge types: "depends-on"
// (a change → each declared dep), "implements" (a change → each capability its
// delta specs touch), "supersedes" (a change → each change it replaces), and
// "deviates-from" (a change → each target it knowingly diverges from).
type graphEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Type string `json:"type"`
}

// graphCmd builds the "onto graph" subcommand: a read-only, config-independent
// view of the change dependency graph over active and archived changes.
func graphCmd() *cobra.Command {
	var (
		dir    string
		asJSON bool
	)
	cmd := &cobra.Command{
		Use:   "graph",
		Short: "Emit the change dependency graph (read-only)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			nodes, edges, err := buildGraph(dir)
			if err != nil {
				return err
			}
			if asJSON {
				b, mErr := json.MarshalIndent(struct {
					Nodes []graphNode `json:"nodes"`
					Edges []graphEdge `json:"edges"`
				}{Nodes: nodes, Edges: edges}, "", "  ")
				if mErr != nil {
					return mErr
				}
				cmd.Println(string(b))
				return nil
			}
			type outEdge struct{ typ, to string }
			edgesFrom := map[string][]outEdge{}
			for _, e := range edges {
				edgesFrom[e.From] = append(edgesFrom[e.From], outEdge{e.Type, e.To})
			}
			for _, n := range nodes {
				if n.Kind == "capability" {
					cmd.Printf("%s (capability)\n", n.Change)
					continue
				}
				suffix := ""
				if n.Archived {
					suffix = ", archived"
				}
				id := n.ID
				if id == "" {
					id = "no-id"
				}
				cmd.Printf("%s (%s, %s%s)\n", n.Change, id, n.Phase, suffix)
				for _, e := range edgesFrom[n.Change] {
					cmd.Printf("  → %s %s\n", e.typ, e.to)
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&dir, "dir", ".", "workspace root to inspect")
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit {nodes, edges} JSON")
	return cmd
}

// buildGraph enumerates active (docs/changes/*) and archived
// (docs/changes/archive/*) changes into nodes + depends-on edges. A
// malformed/missing-state change still yields a node labeled by its directory
// (never silently dropped, mirroring status's F14 rule). Output is deterministic
// (nodes sorted by change name; edges by from,to).
func buildGraph(root string) ([]graphNode, []graphEdge, error) {
	var nodes []graphNode
	var edges []graphEdge

	capSeen := map[string]bool{}
	add := func(dir, fallbackName string, archived bool) {
		st, class, _ := ontostate.Classify(dir)
		name := st.Change
		if class != "valid" || name == "" {
			name = fallbackName
		}
		nodes = append(nodes, graphNode{ID: st.ID, Change: name, Phase: st.Phase, Archived: archived || st.Archived, Kind: "change"})
		for _, dep := range st.Deps {
			edges = append(edges, graphEdge{From: name, To: dep, Type: "depends-on"})
		}
		for _, sup := range st.Supersedes {
			edges = append(edges, graphEdge{From: name, To: sup, Type: "supersedes"})
		}
		for _, dev := range st.DeviatesFrom {
			edges = append(edges, graphEdge{From: name, To: dev, Type: "deviates-from"})
		}
		// implements: a change's delta specs live at specs/<capability>.md (onto's
		// flat delta-spec layout). Each names a capability the change implements.
		if specs, sErr := os.ReadDir(filepath.Join(dir, "specs")); sErr == nil {
			for _, sf := range specs {
				if sf.IsDir() || filepath.Ext(sf.Name()) != ".md" {
					continue
				}
				capName := sf.Name()[:len(sf.Name())-len(".md")]
				edges = append(edges, graphEdge{From: name, To: capName, Type: "implements"})
				if !capSeen[capName] {
					capSeen[capName] = true
					nodes = append(nodes, graphNode{Change: capName, Kind: "capability"})
				}
			}
		}
	}

	changesDir := filepath.Join(root, "docs", "changes")
	entries, err := os.ReadDir(changesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nodes, edges, nil // no changes: an empty graph, still read-only
		}
		return nil, nil, err
	}
	for _, e := range entries {
		if !e.IsDir() || e.Name() == "archive" {
			continue
		}
		add(filepath.Join(changesDir, e.Name()), e.Name(), false)
	}
	if archived, aErr := os.ReadDir(filepath.Join(changesDir, "archive")); aErr == nil {
		for _, e := range archived {
			if !e.IsDir() {
				continue
			}
			add(filepath.Join(changesDir, "archive", e.Name()), e.Name(), true)
		}
	}

	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].Kind != nodes[j].Kind {
			return nodes[i].Kind < nodes[j].Kind
		}
		return nodes[i].Change < nodes[j].Change
	})
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].Type != edges[j].Type {
			return edges[i].Type < edges[j].Type
		}
		if edges[i].From != edges[j].From {
			return edges[i].From < edges[j].From
		}
		return edges[i].To < edges[j].To
	})
	return nodes, edges, nil
}
