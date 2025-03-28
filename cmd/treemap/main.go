package main

import (
	"flag"
	"fmt"
	"image/color"
	"log"
	"os"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/MazenAlkhatib/treemap"
	"github.com/MazenAlkhatib/treemap/parser"
	"github.com/MazenAlkhatib/treemap/render"
)

const doc string = `
Generate treemaps from file in header-less CSV.

Usage:
  treemap [options] -input data.csv

Input format:
  /delimitered/path,size

Example:
  treemap -input data.csv -sizes "1024x768,2048x1536" -output-path output

Command options:
`

var grey = color.RGBA{128, 128, 128, 255}

func main() {
	debug.SetGCPercent(20)

	var (
		sizes         []struct{ w, h float64 }
		marginBox     float64
		paddingBox    float64
		padding       float64
		colorScheme   string
		colorBorder   string
		outputPath    string
		keepLongPaths bool
		inputFile     string
	)

	flag.Usage = func() {
		fmt.Fprint(flag.CommandLine.Output(), doc)
		flag.PrintDefaults()
	}

	// Parse size pairs
	sizesStr := flag.String("sizes", "1024x1024", "comma-separated list of output sizes in format widthxheight (e.g., 1024x768,2048x1536)")
	flag.Float64Var(&marginBox, "margin-box", 4, "margin between boxes")
	flag.Float64Var(&paddingBox, "padding-box", 4, "padding between box border and content")
	flag.Float64Var(&padding, "padding", 32, "padding around root content")
	flag.StringVar(&colorScheme, "color", "balance", "color scheme (RdBu, balance, none)")
	flag.StringVar(&colorBorder, "color-border", "auto", "color of borders (light, dark, auto)")
	flag.StringVar(&outputPath, "output-path", "treemap", "The output path of the rendered image")
	flag.BoolVar(&keepLongPaths, "long-paths", false, "keep long paths when paren has single child")
	flag.StringVar(&inputFile, "input", "", "Input CSV file path (if not provided, reads from stdin)")
	flag.Parse()

	// Parse size pairs
	sizeStrs := strings.Split(*sizesStr, ",")
	sizes = make([]struct{ w, h float64 }, len(sizeStrs))

	for i, sizeStr := range sizeStrs {
		parts := strings.Split(strings.TrimSpace(sizeStr), "x")
		if len(parts) != 2 {
			log.Fatalf("invalid size format: %s (expected widthxheight)", sizeStr)
		}

		w, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		if err != nil {
			log.Fatalf("invalid width value: %v", err)
		}
		sizes[i].w = w

		h, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if err != nil {
			log.Fatalf("invalid height value: %v", err)
		}
		sizes[i].h = h
	}

	fmt.Printf("Processing has been started at %s\n", time.Now().Format("15:04:05"))

	parser := parser.CSVTreeParser{}
	var tree *treemap.Tree
	var err error

	tree, err = parser.ParseFile(inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "can not parse: %v\n", err)
		os.Exit(1)
	}

	if err != nil {
		log.Fatal(err)
	}

	// Force GC before heavy processing
	runtime.GC()

	treemap.SetNamesFromPaths(tree)

	if !keepLongPaths {
		treemap.CollapseLongPaths(tree)
	}

	sizeImputer := treemap.SumSizeImputer{EmptyLeafSize: 1}
	sizeImputer.ImputeSize(*tree)

	// Force GC before coloring setup
	runtime.GC()

	var colorer render.Colorer

	treeHueColorer := render.TreeHueColorer{
		Offset: 0,
		Hues:   map[string]float64{},
		C:      0.5,
		L:      0.5,
		DeltaH: 10,
		DeltaC: 0.3,
		DeltaL: 0.1,
	}

	var borderColor color.Color
	borderColor = color.White

	switch {
	case colorScheme == "none":
		colorer = render.NoneColorer{}
		borderColor = grey
	case colorScheme == "balanced":
		colorer = treeHueColorer
		borderColor = color.White
	default:
		colorer = treeHueColorer
	}

	switch {
	case colorBorder == "light":
		borderColor = color.White
	case colorBorder == "dark":
		borderColor = grey
	}

	uiBuilder := render.UITreeMapBuilder{
		Colorer:     colorer,
		BorderColor: borderColor,
	}

	// Render for each size pair
	for _, size := range sizes {
		renderTreemapStreaming(tree, size.w, size.h, uiBuilder, outputPath, marginBox, paddingBox, padding)
		runtime.GC()
	}
}

func renderTreemapStreaming(tree *treemap.Tree, w, h float64, uiBuilder render.UITreeMapBuilder, outputPath string, marginBox, paddingBox, padding float64) {

	spec := uiBuilder.NewUITreeMap(*tree, w, h, marginBox, paddingBox, padding)
	renderer := render.StreamingSVGRenderer{}

	fileName := fmt.Sprintf("%s_%d_%d_stream.svg", outputPath, int(w), int(h))
	if err := renderer.RenderStream(spec, w, h, fileName); err != nil {
		fmt.Printf("Error streaming to file: %v\n", err)
		return
	}

	// Clean up the spec after rendering
	spec.Children = nil
}
