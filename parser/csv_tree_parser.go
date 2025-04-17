package parser

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/MazenAlkhatib/treemap"
	"github.com/schollz/progressbar/v3"
)

// CSVTreeParser handles parsing of CSV data into a tree structure
type CSVTreeParser struct {
	Comma rune
}

// ParseReader parses CSV data from a reader into a tree structure
func (s *CSVTreeParser) ParseReader(reader io.Reader) (*treemap.Tree, error) {
	tree := &treemap.Tree{
		Nodes: make(map[string]treemap.Node),
		To:    make(map[string][]string),
	}

	// for finding roots
	hasParent := make(map[string]bool)
	// for tracking unique children
	uniqueChildren := make(map[string]map[string]bool)

	r := csv.NewReader(reader)
	if s.Comma != 0 {
		r.Comma = s.Comma
	}
	r.LazyQuotes = true

	// Create progress bar with unknown total
	bar := progressbar.Default(-1)
	bar.Describe("Parsing CSV records")
	count := 0
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error reading CSV: %w", err)
		}

		if len(record) == 0 {
			return nil, errors.New("no values in row")
		}

		if (record[0] == "path" || record[0] == "full_path") && count == 0 {
			// skip header
			count++
			continue
		}

		count++
		path := record[0]
		var size float64
		if len(record) >= 2 {
			size, err = strconv.ParseFloat(record[1], 64)
			if err != nil {
				return nil, fmt.Errorf("size(%s) is not float: %w", record[1], err)
			}
		}

		if strings.HasSuffix(path, "/") {
			continue
		}

		// Get node name from path
		parts := strings.Split(path, "/")
		name := parts[len(parts)-1]

		// Process node
		if existingNode, ok := tree.Nodes[path]; ok {
			tree.Nodes[path] = treemap.Node{
				Path: existingNode.Path,
				Name: existingNode.Name,
				Size: existingNode.Size + size,
			}
		} else {
			tree.Nodes[path] = treemap.Node{
				Path: path,
				Name: name,
				Size: size,
			}
		}

		// Build parent-child relationships
		hasParent[parts[0]] = false

		for parent, i := parts[0], 1; i < len(parts); i++ {
			child := parent + "/" + parts[i]

			if _, ok := tree.Nodes[parent]; !ok {
				parentName := parts[i-1]
				tree.Nodes[parent] = treemap.Node{
					Path: parent,
					Name: parentName,
				}
			}

			// Initialize the unique children map for this parent if needed
			if _, ok := uniqueChildren[parent]; !ok {
				uniqueChildren[parent] = make(map[string]bool)
			}

			// Only add the child if we haven't seen it before for this parent
			if !uniqueChildren[parent][child] {
				tree.To[parent] = append(tree.To[parent], child)
				uniqueChildren[parent][child] = true
			}
			hasParent[child] = true

			parent = child
		}

		bar.Add(1)
	}

	// Finish the progress bar
	bar.Finish()

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

// ParseFile parses a CSV file into a tree structure
func (s *CSVTreeParser) ParseFile(filepath string) (*treemap.Tree, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("cannot open file: %w", err)
	}
	defer file.Close()

	return s.ParseReader(file)
}
