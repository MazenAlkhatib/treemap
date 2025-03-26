package treemap

import (
	"strings"

	"github.com/schollz/progressbar/v3"
)

// SumSizeImputer will set sum of children into empty parents and fill children with contant.
type SumSizeImputer struct {
	EmptyLeafSize float64
}

func (s SumSizeImputer) ImputeSize(t Tree) {
	// Create progress bar with total number of nodes
	bar := progressbar.Default(int64(len(t.Nodes)))
	bar.Describe("Imputing sizes")

	s.ImputeSizeNode(t, t.Root, bar)
}

func (s SumSizeImputer) ImputeSizeNode(t Tree, node string, bar *progressbar.ProgressBar) {
	var sum float64
	for _, child := range t.To[node] {
		s.ImputeSizeNode(t, child, bar)
		sum += t.Nodes[child].Size
	}

	if n, ok := t.Nodes[node]; !ok || n.Size == 0 {
		v := s.EmptyLeafSize
		if len(t.To[node]) > 0 {
			v = sum
		}

		var name string
		if parts := strings.Split(node, "/"); len(parts) > 0 {
			name = parts[len(parts)-1]
		}

		t.Nodes[node] = Node{
			Path: node,
			Name: name,
			Size: v,
		}
	}
	bar.Add(1)
}
