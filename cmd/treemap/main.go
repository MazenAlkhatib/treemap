package main

import (
	"flag"
	"fmt"
	"image/color"
	"log"
	"os"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/MazenAlkhatib/treemap"
	"github.com/MazenAlkhatib/treemap/parser"
	"github.com/MazenAlkhatib/treemap/render"
	"github.com/MazenAlkhatib/treemap/tracker"
)

const doc string = `
Generate treemaps from STDIN in header-less CSV.

</ delimitered path>,<size>

Command options:
`

var grey = color.RGBA{128, 128, 128, 255}

func main() {
	// Set GOGC to trigger GC more frequently
	debug.SetGCPercent(20)

	var (
		w             float64
		h             float64
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
	flag.Float64Var(&w, "w", 1024, "width of output")
	flag.Float64Var(&h, "h", 1024, "height of output")
	flag.Float64Var(&marginBox, "margin-box", 4, "margin between boxes")
	flag.Float64Var(&paddingBox, "padding-box", 4, "padding between box border and content")
	flag.Float64Var(&padding, "padding", 32, "padding around root content")
	flag.StringVar(&colorScheme, "color", "balance", "color scheme (RdBu, balance, none)")
	flag.StringVar(&colorBorder, "color-border", "auto", "color of borders (light, dark, auto)")
	flag.StringVar(&outputPath, "output-path", "treemap", "The output path of the rendered image")
	flag.BoolVar(&keepLongPaths, "long-paths", false, "keep long paths when paren has single child")
	flag.StringVar(&inputFile, "input", "", "Input CSV file path (if not provided, reads from stdin)")
	flag.Parse()

	defer tracker.TrackTime(time.Now(), "Full Execution")
	fmt.Printf("Processing has been started at %s\n", time.Now().Format("15:04:05"))

	// Print initial memory stats
	printMemStats("Initial")

	parser := parser.CSVTreeParser{}
	var tree *treemap.Tree
	var err error

	if inputFile != "" {
		fmt.Printf("Starting batch parsing of %s...\n", inputFile)
		tree, err = parser.ParseFileBatchedArray(inputFile, 5_000_000)
	} else {
		tree, err = parser.ParseReader(os.Stdin)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "can not parse: %v\n", err)
		os.Exit(1)
	}

	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Parsing completed. Nodes: %d\n", len(tree.Nodes))
	printMemStats("After parsing")

	// Force GC before heavy processing
	runtime.GC()

	treemap.SetNamesFromPaths(tree)
	printMemStats("After setting names")

	if !keepLongPaths {
		treemap.CollapseLongPaths(tree)
		printMemStats("After collapsing paths")
	}

	sizeImputer := treemap.SumSizeImputer{EmptyLeafSize: 1}
	sizeImputer.ImputeSize(*tree)
	printMemStats("After size imputation")

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

	renderTreemapStreaming(tree, w, h, uiBuilder, outputPath, marginBox, paddingBox, padding)
	runtime.GC()
	printMemStats(fmt.Sprintf("After %dx%d render", int(w), int(h)))
}

func printMemStats(stage string) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Printf("[%s] Memory usage: Alloc = %v MiB, TotalAlloc = %v MiB, Sys = %v MiB, NumGC = %v\n",
		stage,
		m.Alloc/1024/1024,
		m.TotalAlloc/1024/1024,
		m.Sys/1024/1024,
		m.NumGC)
}

func renderTreemapStreaming(tree *treemap.Tree, w, h float64, uiBuilder render.UITreeMapBuilder, outputPath string, marginBox, paddingBox, padding float64) {
	defer tracker.TrackTime(time.Now(), "RenderTreemapStreaming")
	startTime := time.Now()
	fmt.Printf("Streaming treemap %dx%d started at %s\n", int(w), int(h), startTime.Format("15:04:05"))

	spec := uiBuilder.NewUITreeMap(*tree, w, h, marginBox, paddingBox, padding)
	renderer := render.StreamingSVGRenderer{}

	fileName := fmt.Sprintf("%s_%d_%d_stream.svg", outputPath, int(w), int(h))
	if err := renderer.RenderStream(spec, w, h, fileName); err != nil {
		fmt.Printf("Error streaming to file: %v\n", err)
		return
	}

	// Clean up the spec after rendering
	spec.Children = nil

	elapsed := time.Since(startTime)
	fmt.Printf("Streaming %dx%d completed in %s\n", int(w), int(h), elapsed)
}
