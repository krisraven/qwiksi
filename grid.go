package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// pxPerPt controls the resolution of the generated grid image: 1 image
// pixel per PDF point keeps the overlay pixel-aligned 1:1 with the page
// when stamped back at "scalefactor:1 abs", so no extra unit conversion
// is needed between the grid image and the coordinates a user later
// passes to `qwiksi sign --x --y`.
const pxPerPt = 1.0

// gridLineColor/labelColor are semi-transparent so the underlying page
// content stays legible under the overlay.
var (
	gridLineColor = color.RGBA{R: 220, G: 0, B: 0, A: 130}
	labelColor    = color.RGBA{R: 0, G: 0, B: 0, A: 255}
)

// writeGridPNG renders a coordinate grid sized to a page of widthPt x
// heightPt (PDF points), with lines every spacingPt points, labeled in
// the PDF's native bottom-left-origin coordinate space, to outPath.
func writeGridPNG(widthPt, heightPt, spacingPt float64, outPath string) error {
	if spacingPt <= 0 {
		return fmt.Errorf("grid spacing must be > 0")
	}

	w := int(widthPt*pxPerPt) + 1
	h := int(heightPt*pxPerPt) + 1

	img := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), image.Transparent, image.Point{}, draw.Src)

	face := basicfont.Face7x13

	for x := 0.0; x <= widthPt; x += spacingPt {
		px := int(x * pxPerPt)
		drawVLine(img, px, 0, h-1, gridLineColor)
		drawLabel(img, face, fmt.Sprintf("%d", int(x)), px+2, h-4, labelColor)
	}

	for y := 0.0; y <= heightPt; y += spacingPt {
		// Flip to image space (row 0 = top) while keeping the label text
		// itself in PDF bottom-left coordinates.
		py := h - 1 - int(y*pxPerPt)
		drawHLine(img, py, 0, w-1, gridLineColor)
		drawLabel(img, face, fmt.Sprintf("%d", int(y)), 2, py-3, labelColor)
	}

	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		return err
	}
	return f.Close()
}

func drawVLine(img *image.RGBA, x, y0, y1 int, c color.Color) {
	for y := y0; y <= y1; y++ {
		img.Set(x, y, c)
	}
}

func drawHLine(img *image.RGBA, y, x0, x1 int, c color.Color) {
	for x := x0; x <= x1; x++ {
		img.Set(x, y, c)
	}
}

func drawLabel(img *image.RGBA, face font.Face, s string, x, y int, c color.Color) {
	if y < 0 {
		y = 10
	}
	if y >= img.Bounds().Dy() {
		y = img.Bounds().Dy() - 4
	}
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(c),
		Face: face,
		Dot:  fixed.P(x, y),
	}
	d.DrawString(s)
}
