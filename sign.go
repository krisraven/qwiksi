package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const defaultSignatureWidthPt = 150

func runSign(args []string) error {
	fs := flag.NewFlagSet("sign", flag.ExitOnError)
	sig := fs.String("signature", "", "path to an existing signature image (PNG/JPEG)")
	text := fs.String("text", "", "signature text to render on the fly, instead of --signature")
	fontID := fs.String("font", "1", "cursive font for --text: 1 (Sacramento) or 2 (Great Vibes)")
	size := fs.Float64("size", 100, "font size in points for --text")
	colorHex := fs.String("color", "000000", "signature ink color as a hex RGB triple for --text")
	out := fs.String("output", "", "output PDF path (default: <input>_signed.pdf)")
	field := fs.String("field", "", "name of an existing AcroForm field to sign into")
	page := fs.Int("page", 0, "page number to sign (manual mode)")
	x := fs.Float64("x", 0, "x offset in PDF points from bottom-left (manual mode)")
	y := fs.Float64("y", 0, "y offset in PDF points from bottom-left (manual mode)")
	width := fs.Float64("width", 0, "signature width in PDF points (manual mode, default 150)")
	height := fs.Float64("height", 0, "signature height in PDF points (manual mode, default derived from aspect ratio)")
	fs.Usage = func() {
		fmt.Println(`Usage:
  qwiksi sign <input.pdf> --signature sig.png --field "FieldName" [--output out.pdf]
  qwiksi sign <input.pdf> --signature sig.png --page N --x X --y Y [--width W] [--height H] [--output out.pdf]

Instead of --signature, pass --text "Your Name" [--font 1|2] [--size 100] [--color 000000]
to render the signature on the fly - no separate addsig step needed.`)
	}
	inFile, rest, err := takeInputFile(args)
	if err != nil {
		fs.Usage()
		return err
	}
	fs.Parse(rest)

	if (*sig == "") == (*text == "") {
		return fmt.Errorf("specify exactly one of --signature or --text")
	}

	if *text != "" {
		path, err := writeTempSignaturePNG(*text, *fontID, *size, *colorHex)
		if err != nil {
			return err
		}
		defer os.Remove(path)
		*sig = path
	}

	imgW, imgH, err := imageDimsPx(*sig)
	if err != nil {
		return err
	}

	if *out == "" {
		ext := filepath.Ext(inFile)
		*out = strings.TrimSuffix(inFile, ext) + "_signed" + ext
	}

	fieldMode := *field != ""
	manualMode := *page != 0 || *x != 0 || *y != 0
	if fieldMode == manualMode {
		return fmt.Errorf("specify exactly one of --field, or --page/--x/--y")
	}

	var dx, dy, scale float64
	var pageNr, widgetObjNr int

	if fieldMode {
		dx, dy, scale, pageNr, widgetObjNr, err = fieldPlacement(inFile, *field, 0, imgW, imgH)
		if err != nil {
			return err
		}
	} else {
		if *page < 1 {
			return fmt.Errorf("--page is required (and must be >= 1) in manual mode")
		}
		pageNr = *page
		dx, dy = *x, *y

		switch {
		case *width > 0:
			scale = *width / float64(imgW)
		case *height > 0:
			scale = *height / float64(imgH)
		default:
			scale = defaultSignatureWidthPt / float64(imgW)
		}
	}

	if err := stampSignature(inFile, *sig, *out, dx, dy, scale, pageNr, fieldMode, widgetObjNr); err != nil {
		return err
	}

	fmt.Printf("Wrote %s\n", *out)
	return nil
}
