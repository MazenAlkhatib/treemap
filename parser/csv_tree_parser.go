package parser

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/MazenAlkhatib/treemap"
	"github.com/MazenAlkhatib/treemap/tracker"
)

// CSVTreeParser handles parsing of CSV data into a tree structure
type CSVTreeParser struct {
	Comma rune
}

// if duplicates, then sum size
// TODO: policies for duplicates
func (s CSVTreeParser) ParseReader(reader io.Reader) (*treemap.Tree, error) {
	defer tracker.TrackTime(time.Now(), "CSV Parsing")

	tree := &treemap.Tree{
		Nodes: make(map[string]treemap.Node),
		To:    make(map[string][]string),
	}

	// for finding roots
	hasParent := make(map[string]bool)

	r := csv.NewReader(reader)
	r.LazyQuotes = true

	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("can not parse: %w", err)
		}

		if len(record) == 0 {
			return nil, errors.New("no values in row")
		}

		node := treemap.Node{Path: record[0]}

		if len(record) >= 2 {
			v, err := strconv.ParseFloat(record[1], 64)
			if err != nil {
				return nil, fmt.Errorf("size(%s) is not float: %w", record[1], err)
			}
			node.Size = v
		}

		// Process node immediately instead of collecting all nodes
		if existingNode, ok := tree.Nodes[node.Path]; ok {
			tree.Nodes[node.Path] = treemap.Node{
				Path: existingNode.Path,
				Name: existingNode.Name,
				Size: existingNode.Size + node.Size,
			}
		} else {
			tree.Nodes[node.Path] = node
		}

		parts := strings.Split(node.Path, "/")
		hasParent[parts[0]] = false

		for parent, i := parts[0], 1; i < len(parts); i++ {
			child := parent + "/" + parts[i]

			if _, ok := tree.Nodes[parent]; !ok {
				tree.Nodes[parent] = treemap.Node{
					Path: parent,
				}
			}
			tree.To[parent] = append(tree.To[parent], child)
			hasParent[child] = true

			parent = child
		}
	}

	// Deduplicate edges
	for node, v := range tree.To {
		tree.To[node] = unique(v)
	}

	// Find roots
	var roots []string
	for node, has := range hasParent {
		if !has {
			roots = append(roots, node)
		}
	}

	switch {
	case len(roots) == 0:
		return nil, errors.New("no roots, possible cycle in graph")
	case len(roots) > 1:
		tree.Root = "some-secret-string"
		tree.To[tree.Root] = roots
	default:
		tree.Root = roots[0]
	}

	return tree, nil
}

func (s CSVTreeParser) ParseString(in string) (*treemap.Tree, error) {
	return s.ParseReader(strings.NewReader(in))
}

func (s CSVTreeParser) WriteTreeToFile(tree treemap.Tree, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Start writing from the root node
	writeNode(tree, tree.Root, 0, file)

	fmt.Println("Tree successfully written to", filename)
	return nil
}

// Recursive function to print tree hierarchy
func writeNode(tree treemap.Tree, path string, level int, file *os.File) {
	node, exists := tree.Nodes[path]
	if !exists {
		return
	}

	// Indentation based on depth level
	indent := ""
	for i := 0; i < level; i++ {
		indent += "  "
	}

	// Format node details
	line := fmt.Sprintf("%s- %s (Path: %s, Size: %.2f)\n",
		indent, node.Name, node.Path, node.Size)

	// Write to file
	file.WriteString(line)

	// Recur for children
	for _, child := range tree.To[path] {
		writeNode(tree, child, level+1, file)
	}
}

func parseNodes(in string) ([]treemap.Node, error) {
	var nodes []treemap.Node
	r := csv.NewReader(strings.NewReader(in))
	r.LazyQuotes = true
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("can not parse: %w", err)
		}

		if len(record) == 0 {
			return nil, errors.New("no values in row")
		}

		node := treemap.Node{Path: record[0]}

		if len(record) >= 2 {
			v, err := strconv.ParseFloat(record[1], 64)
			if err != nil {
				return nil, fmt.Errorf("size(%s) is not float: %w", record[1], err)
			}
			node.Size = v
		}

		nodes = append(nodes, node)
	}
	return nodes, nil
}

// If node is in path, but not present, then it will be in To but not will have entry in Nodes.
// This is not terribly efficient, but should do its job for small graphs.
func makeTree(nodes []treemap.Node) (*treemap.Tree, error) {
	tree := treemap.Tree{
		Nodes: map[string]treemap.Node{},
		To:    map[string][]string{},
	}

	// for finding roots
	hasParent := map[string]bool{}

	for _, node := range nodes {
		if existingNode, ok := tree.Nodes[node.Path]; ok {
			tree.Nodes[node.Path] = treemap.Node{
				Path: existingNode.Path,
				Name: existingNode.Name,
				Size: existingNode.Size + node.Size,
			}
		}
		tree.Nodes[node.Path] = node

		parts := strings.Split(node.Path, "/")
		hasParent[parts[0]] = false

		for parent, i := parts[0], 1; i < len(parts); i++ {
			child := parent + "/" + parts[i]

			if _, ok := tree.Nodes[parent]; !ok {
				tree.Nodes[parent] = treemap.Node{
					Path: parent,
				}
			}
			tree.To[parent] = append(tree.To[parent], child)
			hasParent[child] = true

			parent = child
		}
	}

	for node, v := range tree.To {
		tree.To[node] = unique(v)
	}

	var roots []string
	for node, has := range hasParent {
		if !has {
			roots = append(roots, node)
		}
	}

	switch {
	case len(roots) == 0:
		return nil, errors.New("no roots, possible cycle in graph")
	case len(roots) > 1:
		tree.Root = "some-secret-string"
		tree.To[tree.Root] = roots
	default:
		tree.Root = roots[0]
	}

	return &tree, nil
}

func unique(a []string) []string {
	u := map[string]bool{}
	var b []string
	for _, q := range a {
		if _, ok := u[q]; !ok {
			u[q] = true
			b = append(b, q)
		}
	}
	return b
}

func (s CSVTreeParser) ParseFile(filepath string) (*treemap.Tree, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("cannot open file: %w", err)
	}
	defer file.Close()

	return s.ParseReader(file)
}

// ParseFileBatched parses a CSV file in batches to build the tree structure
func (s CSVTreeParser) ParseFileBatched(filename string, batchSize int) (*treemap.Tree, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	return s.ParseReaderBatched(file, batchSize)
}

// ParseReaderBatched parses a CSV reader in batches to build the tree structure
func (s *CSVTreeParser) ParseReaderBatched(reader io.Reader, batchSize int) (*treemap.Tree, error) {
	return s.ParseReaderBatchedArray(reader, batchSize)
}

// ParseReaderBatchedArray parses a CSV reader in batches to build the tree structure using the array-based structure
func (s *CSVTreeParser) ParseReaderBatchedArray(reader io.Reader, batchSize int) (*treemap.Tree, error) {
	defer tracker.TrackTime(time.Now(), "CSV Parsing")

	// Initialize the final tree structure
	tree := &treemap.Tree{
		Nodes: make(map[string]treemap.Node),
		To:    make(map[string][]string),
	}

	// Create CSV reader
	csvReader := csv.NewReader(reader)
	if s.Comma != 0 {
		csvReader.Comma = s.Comma
	}

	recordCount := 0
	startTime := time.Now()

	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error reading CSV: %v", err)
		}

		// Parse size
		size, err := strconv.ParseFloat(record[1], 64)
		if err != nil {
			return nil, fmt.Errorf("error parsing size: %v", err)
		}

		// Process record immediately
		path := record[0]
		if existingNode, ok := tree.Nodes[path]; ok {
			tree.Nodes[path] = treemap.Node{
				Path: existingNode.Path,
				Name: existingNode.Name,
				Size: existingNode.Size + size,
			}
		} else {
			tree.Nodes[path] = treemap.Node{
				Path: path,
				Size: size,
			}
		}

		// Process parent-child relationships
		parts := strings.Split(path, "/")
		for parent, i := parts[0], 1; i < len(parts); i++ {
			child := parent + "/" + parts[i]

			if _, ok := tree.Nodes[parent]; !ok {
				tree.Nodes[parent] = treemap.Node{
					Path: parent,
				}
			}
			tree.To[parent] = append(tree.To[parent], child)

			parent = child
		}

		recordCount++
		if recordCount%1000 == 0 {
			// Print progress every 1000 records
			elapsed := time.Since(startTime)
			recordsPerSecond := float64(recordCount) / elapsed.Seconds()
			fmt.Printf("\rProcessed %d records (%.2f records/sec)", recordCount, recordsPerSecond)
		}
	}

	// Deduplicate edges
	for node, v := range tree.To {
		tree.To[node] = unique(v)
	}

	// Find root nodes (nodes without parents)
	hasParent := make(map[string]bool)
	for _, children := range tree.To {
		for _, child := range children {
			hasParent[child] = true
		}
	}

	// Set root
	var roots []string
	for node := range tree.Nodes {
		if !hasParent[node] {
			roots = append(roots, node)
		}
	}

	switch {
	case len(roots) == 0:
		return nil, errors.New("no roots, possible cycle in graph")
	case len(roots) > 1:
		tree.Root = "some-secret-string"
		tree.To[tree.Root] = roots
	default:
		tree.Root = roots[0]
	}

	fmt.Printf("\nFinished processing %d records in %v\n", recordCount, time.Since(startTime))
	return tree, nil
}

// ArrayNode represents a node in the array-based tree structure
type ArrayNode struct {
	Path         string
	Size         float64
	ParentIndex  int
	ChildIndices []int
}

// ArrayTree represents the tree structure using arrays
type ArrayTree struct {
	Nodes       []ArrayNode
	RootIndex   int
	PathToIndex map[string]int // Maps path to node index for quick lookups
}

// NewArrayTree creates a new ArrayTree
func NewArrayTree() *ArrayTree {
	return &ArrayTree{
		Nodes:       make([]ArrayNode, 0),
		PathToIndex: make(map[string]int),
	}
}

// ToTreemapTree converts the array-based tree to a treemap.Tree
func (t *ArrayTree) ToTreemapTree() *treemap.Tree {
	tree := &treemap.Tree{
		Nodes: make(map[string]treemap.Node),
		To:    make(map[string][]string),
	}

	// First pass: create all nodes
	for _, node := range t.Nodes {
		tree.Nodes[node.Path] = treemap.Node{
			Path: node.Path,
			Size: node.Size,
		}
	}

	// Second pass: create edges
	for _, node := range t.Nodes {
		if node.ParentIndex != -1 {
			parentPath := t.Nodes[node.ParentIndex].Path
			tree.To[parentPath] = append(tree.To[parentPath], node.Path)
		}
	}

	// Find root (node with no parent)
	for _, node := range t.Nodes {
		if node.ParentIndex == -1 {
			tree.Root = node.Path
			break
		}
	}

	return tree
}

// ParseFileBatchedArray parses a CSV file in batches
func (s *CSVTreeParser) ParseFileBatchedArray(filename string, batchSize int) (*treemap.Tree, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("can not open file: %w", err)
	}
	defer file.Close()

	return s.ParseReaderBatchedArray(file, batchSize)
}
