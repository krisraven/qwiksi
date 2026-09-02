package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

const (
	fontDemoPageWidth = 595 // A4 width in points, treated 1:1 as px
	fontDemoMargin    = 48
	fontDemoRowHeight = 110
)

func runFontDemo(args []string) error {
	fs := flag.NewFlagSet("fontdemo", flag.ExitOnError)
	out := fs.String("out", "fontdemo.pdf", "output PDF path")
	size := fs.Float64("size", 44, "font size in points for each rendered name")
	colorHex := fs.String("color", "000000", "ink color as a hex RGB triple")
	fs.Usage = func() {
		fmt.Println(`Usage: qwiksi fontdemo [--out fontdemo.pdf] [--size 44] [--color 000000]

Renders every bundled cursive font's own name set in that font (e.g. "Allura"
set in Allura), one per line, onto a single-page PDF - a quick way to compare
them before picking a --font id for addsig/sign.`)
	}
	fs.Parse(args)

	col, err := parseHexColor(*colorHex)
	if err != nil {
		return err
	}

	img, err := renderFontDemoImage(*size, col)
	if err != nil {
		return err
	}

	pngFile, err := os.CreateTemp("", "qwiksi-fontdemo-*.png")
	if err != nil {
		return err
	}
	pngPath := pngFile.Name()
	defer os.Remove(pngPath)
	if err := png.Encode(pngFile, img); err != nil {
		pngFile.Close()
		return fmt.Errorf("encoding demo image: %w", err)
	}
	if err := pngFile.Close(); err != nil {
		return fmt.Errorf("writing demo image: %w", err)
	}

	// ImportImagesFile appends to an existing outFile rather than replacing
	// it, so start from a clean slate to keep repeated runs idempotent.
	if err := os.Remove(*out); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing existing %s: %w", *out, err)
	}
	if err := api.ImportImagesFile([]string{pngPath}, *out, nil, nil); err != nil {
		return fmt.Errorf("building %s: %w", *out, err)
	}

	fmt.Printf("Wrote %s (%d fonts)\n", *out, len(cursiveFonts))
	return nil
}

// renderFontDemoImage draws one row per bundled cursive font: a plain
// caption (name and --font id) followed by the font's own name rendered in
// itself. Pixel dimensions are treated as PDF points 1:1 (see
// fontDemoPageWidth), matching the convention used throughout qwiksi.
func renderFontDemoImage(size float64, col color.RGBA) (*image.RGBA, error) {
	w := fontDemoPageWidth
	h := fontDemoMargin*2 + fontDemoRowHeight*len(cursiveFonts)

	img := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), image.White, image.Point{}, draw.Src)

	capFace := basicfont.Face7x13
	capDrawer := &font.Drawer{Dst: img, Src: image.NewUniform(color.Black), Face: capFace}

	for i, cf := range cursiveFonts {
		rowTop := fontDemoMargin + i*fontDemoRowHeight

		capDrawer.Dot = fixed.Point26_6{
			X: fixed.I(fontDemoMargin),
			Y: fixed.I(rowTop + 16),
		}
		capDrawer.DrawString(fmt.Sprintf("%s - --font %s", cf.name, cf.id))

		face, _, err := loadCursiveFace(cf.id, size)
		if err != nil {
			return nil, fmt.Errorf("rendering %s: %w", cf.label, err)
		}
		d := &font.Drawer{
			Dst:  img,
			Src:  image.NewUniform(col),
			Face: face,
			Dot: fixed.Point26_6{
				X: fixed.I(fontDemoMargin),
				Y: fixed.I(rowTop + 16 + int(size)),
			},
		}
		d.DrawString(cf.label)
		face.Close()
	}

	return img, nil
}
