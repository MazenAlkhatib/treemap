package treemap

import (
	"strings"

	"github.com/schollz/progressbar/v3"
)

// CollapseLongPaths will collapse all long chains in tree.
func CollapseLongPaths(t *Tree) {
	if t == nil {
		return
	}

	// Create progress bar with total number of nodes
	bar := progressbar.Default(int64(len(t.Nodes)))
	bar.Describe("Collapsing long paths")

	// Process nodes
	CollapseLongPathsFromNode(t, t.Root, bar)
}

// CollapseLongPathsFromNode will collapse current node into children as long as it has single child.
// Will set name of this node to joined path from roots.
// Will set size to this child's size.
// Expecting Name containing either single value for current node.
func CollapseLongPathsFromNode(t *Tree, nodeName string, bar *progressbar.ProgressBar) {
	if t == nil {
		return
	}

	bar.Add(1)

	parts := []string{}
	q := nodeName
	for children := t.To[q]; len(children) == 1; children = t.To[q] {
		nextChild := children[0]

		parts = append(parts, t.Nodes[q].Name)
		delete(t.Nodes, q)
		delete(t.To, q)

		q = nextChild
	}

	// if we deleted some children
	if q != nodeName {
		// redirect edges from current node to last child
		t.To[nodeName] = make([]string, len(t.To[q]))
		copy(t.To[nodeName], t.To[q])

		node := t.Nodes[q]

		// add last child node name to path
		parts = append(parts, node.Name)

		// copy fields from child to current node
		t.Nodes[nodeName] = Node{
			Path: node.Path,
			Name: strings.Join(parts, "/"),
			Size: node.Size,
		}

		// delete last child, since it is unreachable now
		delete(t.Nodes, q)
		delete(t.To, q)
	}

	// recursively collapse
	for _, node := range t.To[nodeName] {
		CollapseLongPathsFromNode(t, node, bar)
	}
}
