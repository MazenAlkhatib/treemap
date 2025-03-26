package treemap

import (
	"strings"
)

type Node struct {
	Path string
	Name string
	Size float64
}

type Tree struct {
	Nodes map[string]Node     // node identifier (path) -> Node
	To    map[string][]string // node identifier (path) -> list of node identifiers (paths) for edges from it (to children)
	Root  string
}

// SetNamesFromPaths will update each node to its path leaf as name.
func SetNamesFromPaths(t *Tree) {
	if t == nil {
		return
	}

	for path, node := range t.Nodes {
		parts := strings.Split(node.Path, "/")
		if len(parts) == 0 {
			continue
		}

		t.Nodes[path] = Node{
			Path: node.Path,
			Name: parts[len(parts)-1],
			Size: node.Size,
		}
	}
}
