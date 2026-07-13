package ontocli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"

	"github.com/noviopenworks/homonto/internal/ontostate"
	"github.com/spf13/cobra"
)

// graphNode is one change in the traceability graph.
type graphNode struct {
	ID       string `json:"id"`
	Change   string `json:"change"`
	Phase    string `json:"phase"`
	Archived bool   `json:"archived"`
}

// graphEdge is a typed relationship between changes. Today the only edge type is
// "depends-on" (from a change to each of its declared deps).
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
			edgesFrom := map[string][]string{}
			for _, e := range edges {
				edgesFrom[e.From] = append(edgesFrom[e.From], e.To)
			}
			for _, n := range nodes {
				suffix := ""
				if n.Archived {
					suffix = ", archived"
				}
				id := n.ID
				if id == "" {
					id = "no-id"
				}
				cmd.Printf("%s (%s, %s%s)\n", n.Change, id, n.Phase, suffix)
				for _, to := range edgesFrom[n.Change] {
					cmd.Printf("  → depends-on %s\n", to)
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

	add := func(dir, fallbackName string, archived bool) {
		st, class, _ := ontostate.Classify(dir)
		name := st.Change
		if class != "valid" || name == "" {
			name = fallbackName
		}
		nodes = append(nodes, graphNode{ID: st.ID, Change: name, Phase: st.Phase, Archived: archived || st.Archived})
		for _, dep := range st.Deps {
			edges = append(edges, graphEdge{From: name, To: dep, Type: "depends-on"})
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

	sort.Slice(nodes, func(i, j int) bool { return nodes[i].Change < nodes[j].Change })
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].From != edges[j].From {
			return edges[i].From < edges[j].From
		}
		return edges[i].To < edges[j].To
	})
	return nodes, edges, nil
}
