package render

import (
	"fmt"
	"image/color"
	"io"
	"os"
	"runtime"
	"strings"
	"time"
)

var xmlEscaper = strings.NewReplacer(
	"&", "&amp;",
	"<", "&lt;",
	">", "&gt;",
	"\"", "&quot;",
	"'", "&apos;",
)

// StreamingSVGRenderer is an optimized renderer that writes SVG directly to file
type StreamingSVGRenderer struct{}

// RenderStream renders the treemap directly to a file with optimized memory usage
func (r StreamingSVGRenderer) RenderStream(root UIBox, w, h float64, filename string) error {
	if !root.IsRoot {
		return fmt.Errorf("not a root node")
	}

	start := time.Now()
	fmt.Printf("Rendering SVG tree map at %v...\n", start)

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Write SVG header
	if _, err := fmt.Fprintf(file, `
<svg 
	xmlns="http://www.w3.org/2000/svg" 
	xmlns:xlink="http://www.w3.org/1999/xlink" 
	viewBox="0 0 %f %f" 
	style="background: white none repeat scroll 0%% 0%%;"
>`, w, h); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	// Process boxes in batches to control memory usage
	const batchSize = 1000
	que := make([]UIBox, 0, batchSize)
	que = append(que, root)

	var processed int
	var currentBatch []UIBox

	for len(que) > 0 {
		// Take up to batchSize boxes from queue
		batchEnd := batchSize
		if batchEnd > len(que) {
			batchEnd = len(que)
		}
		currentBatch = que[:batchEnd]
		que = que[batchEnd:]

		// Process current batch
		for _, q := range currentBatch {
			// Add children to queue before we process the box
			if len(q.Children) > 0 {
				que = append(que, q.Children...)
			}

			// Write box SVG directly to file
			if !q.IsInvisible {
				if err := streamBoxSVG(file, q); err != nil {
					return fmt.Errorf("failed to write box: %w", err)
				}
			}

			// Clear children after processing to free memory
			q.Children = nil
		}

		// Clear batch slice
		currentBatch = currentBatch[:0]

		// Periodic GC
		processed += batchEnd
		if processed%10000 == 0 {
			runtime.GC()
		}
	}

	// Write SVG footer
	if _, err := io.WriteString(file, "\n</svg>"); err != nil {
		return fmt.Errorf("failed to write footer: %w", err)
	}

	fmt.Printf("SVG tree map rendering completed in %v\n", time.Since(start))
	return nil
}

// streamBoxSVG writes a single box's SVG directly to the file
func streamBoxSVG(file *os.File, q UIBox) error {
	// Get box colors
	r, g, b, a := color.White.RGBA()
	if q.Color != color.Opaque {
		r, g, b, a = q.Color.RGBA()
	}
	r = r >> 8
	g = g >> 8
	b = b >> 8
	o := float64(a>>8) / 255.0

	br, bg, bb, ba := color.White.RGBA()
	if q.BorderColor != color.Opaque {
		br, bg, bb, ba = q.BorderColor.RGBA()
	}
	br = br >> 8
	bg = bg >> 8
	bb = bb >> 8
	bo := float64(ba>>8) / 255.0

	// Write box opening
	if _, err := io.WriteString(file, "\n<g>"); err != nil {
		return err
	}

	// Write rectangle
	if _, err := fmt.Fprintf(file, `
	<rect x="%f" y="%f" width="%f" height="%f" style="fill: rgb(%d, %d, %d);opacity:1;fill-opacity:%.2f;stroke:rgb(%d,%d,%d);stroke-width:1px;stroke-opacity:%.2f;" />`,
		q.X, q.Y, q.W, q.H,
		r, g, b, o,
		br, bg, bb, bo); err != nil {
		return err
	}

	// Write text if present
	if q.Title != nil {
		if err := streamTextSVG(file, q.Title); err != nil {
			return err
		}
	}

	// Write box closing
	if _, err := io.WriteString(file, "\n</g>\n"); err != nil {
		return err
	}

	return nil
}

// streamTextSVG writes text SVG directly to the file
func streamTextSVG(file *os.File, t *UIText) error {
	if t == nil {
		return nil
	}

	r, g, b, a := color.Black.RGBA()
	if t.Color != color.Opaque {
		r, g, b, a = t.Color.RGBA()
	}
	r = r >> 8
	g = g >> 8
	b = b >> 8
	o := float64(a>>8) / 255.0

	_, err := fmt.Fprintf(file, `
	<text 
		data-notex="1" 
		text-anchor="start"
		transform="translate(%f,%f) scale(%f)"
		style="font-family: Open Sans, verdana, arial, sans-serif !important; font-size: %dpx; fill: rgb(%d, %d, %d); fill-opacity: %.2f; white-space: pre;" 
		data-math="N">%s</text>`,
		t.X,
		t.Y+t.H,
		t.Scale,
		fontSize,
		r, g, b, o,
		xmlEscaper.Replace(t.Text))

	return err
}
