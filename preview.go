package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func runPreview(args []string) error {
	fs := flag.NewFlagSet("preview", flag.ExitOnError)
	page := fs.Int("page", 1, "page number to preview (1-based)")
	grid := fs.Float64("grid", 50, "grid line spacing in PDF points")
	out := fs.String("out", "", "output PDF path (default: <input>_preview.pdf)")
	fs.Usage = func() {
		fmt.Println("Usage: qwiksi preview <input.pdf> --page N [--grid 50] [--out preview.pdf]")
	}

	inFile, rest, err := takeInputFile(args)
	if err != nil {
		fs.Usage()
		return err
	}
	fs.Parse(rest)

	readFile, cleanup, err := prepareInput(inFile)
	if err != nil {
		return fmt.Errorf("reading %s: %w", inFile, err)
	}
	defer cleanup()

	if *out == "" {
		ext := filepath.Ext(inFile)
		*out = strings.TrimSuffix(inFile, ext) + "_preview" + ext
	}

	dims, err := api.PageDimsFile(readFile)
	if err != nil {
		return fmt.Errorf("reading page dimensions: %w", err)
	}
	if *page < 1 || *page > len(dims) {
		return fmt.Errorf("page %d out of range (document has %d pages)", *page, len(dims))
	}
	dim := dims[*page-1]

	gridPNG, err := os.CreateTemp("", "qwiksi-grid-*.png")
	if err != nil {
		return err
	}
	gridPNGPath := gridPNG.Name()
	gridPNG.Close()
	defer os.Remove(gridPNGPath)

	if err := writeGridPNG(dim.Width, dim.Height, *grid, gridPNGPath); err != nil {
		return fmt.Errorf("generating grid overlay: %w", err)
	}

	wm, err := api.ImageWatermark(gridPNGPath, stampDescriptor(0, 0, 1), true, false, types.POINTS)
	if err != nil {
		return fmt.Errorf("building grid overlay watermark: %w", err)
	}

	if err := api.AddWatermarksFile(readFile, *out, []string{fmt.Sprintf("%d", *page)}, wm, nil); err != nil {
		return fmt.Errorf("stamping grid overlay onto page %d: %w", *page, err)
	}

	fmt.Printf("Wrote %s - open it in any PDF viewer and read off page-point coordinates (origin: bottom-left) for `qwiksi sign --x --y`.\n", *out)
	return nil
}
