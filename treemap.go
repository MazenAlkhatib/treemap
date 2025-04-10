package treemap

import (
	"fmt"
	"strings"

	"github.com/schollz/progressbar/v3"
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

	// Create progress bar with total number of nodes
	bar := progressbar.Default(int64(len(t.Nodes)))
	bar.Describe("Updating node names")

	for path, node := range t.Nodes {
		parts := strings.Split(node.Path, "/")
		if len(parts) == 0 {
			fmt.Println("no parts", node.Path)
			continue
		}

		t.Nodes[path] = Node{
			Path: node.Path,
			Name: parts[len(parts)-1],
			Size: node.Size,
		}
		bar.Add(1)
	}

	// Finish the progress bar
	bar.Finish()
}
