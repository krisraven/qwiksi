package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"strconv"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

const addSigPaddingPt = 20

func runAddSig(args []string) error {
	fs := flag.NewFlagSet("addsig", flag.ExitOnError)
	text := fs.String("text", "", "signature text to render (required)")
	fontID := fs.String("font", "1", "cursive font: 1 (Sacramento) or 2 (Great Vibes)")
	out := fs.String("out", "signature.png", "output PNG path")
	size := fs.Float64("size", 100, "font size in points")
	colorHex := fs.String("color", "000000", "signature ink color as a hex RGB triple")
	fs.Usage = func() {
		fmt.Println(`Usage: qwiksi addsig --text "Your Name" [--font 1|2] [--size 100] [--color 000000] [--out signature.png]

Fonts:
  1  Sacramento  - casual flowing script
  2  Great Vibes - elegant formal script`)
	}
	fs.Parse(args)

	if *text == "" {
		fs.Usage()
		return fmt.Errorf("--text is required")
	}

	col, err := parseHexColor(*colorHex)
	if err != nil {
		return err
	}

	img, fontName, err := renderSignatureImage(*text, *fontID, *size, col)
	if err != nil {
		return err
	}

	f, err := os.Create(*out)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		return fmt.Errorf("writing %s: %w", *out, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("writing %s: %w", *out, err)
	}

	b := img.Bounds()
	fmt.Printf("Wrote %s (%dx%d) using %s\n", *out, b.Dx(), b.Dy(), fontName)
	return nil
}

// renderSignatureImage draws text in the given cursive font as a
// transparent-background RGBA image. Shared by `addsig` (writes it straight
// to a PNG file) and `sign --text` (feeds it directly into the stamping
// step without an intermediate file).
func renderSignatureImage(text, fontID string, size float64, col color.RGBA) (*image.RGBA, string, error) {
	cf, err := lookupFont(fontID)
	if err != nil {
		return nil, "", err
	}

	sfntFont, err := opentype.Parse(cf.data)
	if err != nil {
		return nil, "", fmt.Errorf("parsing font: %w", err)
	}
	face, err := opentype.NewFace(sfntFont, &opentype.FaceOptions{
		Size:    size,
		DPI:     72, // 1pt == 1px, matching the point convention used elsewhere in qwiksi
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil, "", fmt.Errorf("creating font face: %w", err)
	}
	defer face.Close()

	d := &font.Drawer{Face: face}
	bounds, _ := d.BoundString(text)
	metrics := face.Metrics()

	textW := (bounds.Max.X - bounds.Min.X).Ceil()
	textH := (metrics.Ascent + metrics.Descent).Ceil()
	w := textW + 2*addSigPaddingPt
	h := textH + 2*addSigPaddingPt

	img := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), image.Transparent, image.Point{}, draw.Src)

	d.Dst = img
	d.Src = image.NewUniform(col)
	d.Dot = fixed.Point26_6{
		X: fixed.I(addSigPaddingPt) - bounds.Min.X,
		Y: fixed.I(addSigPaddingPt) + metrics.Ascent,
	}
	d.DrawString(text)

	return img, cf.name, nil
}

func parseHexColor(s string) (color.RGBA, error) {
	s = strings.TrimPrefix(s, "#")
	if len(s) != 6 {
		return color.RGBA{}, fmt.Errorf("color must be a 6-digit hex RGB value, got %q", s)
	}
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return color.RGBA{}, fmt.Errorf("invalid hex color %q: %w", s, err)
	}
	return color.RGBA{
		R: uint8(v >> 16),
		G: uint8(v >> 8),
		B: uint8(v),
		A: 255,
	}, nil
}
