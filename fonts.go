package main

import (
	"fmt"

	_ "embed"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

//go:embed assets/fonts/Sacramento-Regular.ttf
var fontSacramento []byte

//go:embed assets/fonts/GreatVibes-Regular.ttf
var fontGreatVibes []byte

//go:embed assets/fonts/Qwigley-Regular.ttf
var fontQwigley []byte

//go:embed assets/fonts/Satisfy-Regular.ttf
var fontSatisfy []byte

//go:embed assets/fonts/Allura-Regular.ttf
var fontAllura []byte

//go:embed assets/fonts/AlexBrush-Regular.ttf
var fontAlexBrush []byte

type cursiveFont struct {
	id    string
	label string // plain font name, e.g. "Great Vibes"
	name  string // label plus a short style descriptor, for user-facing messages
	data  []byte
}

// cursiveFonts are bundled directly into the binary. Sacramento, Great
// Vibes, Qwigley, Allura and Alex Brush are SIL Open Font License 1.1
// (assets/fonts/OFL-*.txt); Satisfy is Apache License 2.0
// (assets/fonts/LICENSE-Satisfy.txt).
var cursiveFonts = []cursiveFont{
	{id: "1", label: "Sacramento", name: "Sacramento (casual script)", data: fontSacramento},
	{id: "2", label: "Great Vibes", name: "Great Vibes (formal script)", data: fontGreatVibes},
	{id: "3", label: "Qwigley", name: "Qwigley (tall looping monoline)", data: fontQwigley},
	{id: "4", label: "Satisfy", name: "Satisfy (relaxed marker script)", data: fontSatisfy},
	{id: "5", label: "Allura", name: "Allura (delicate calligraphic)", data: fontAllura},
	{id: "6", label: "Alex Brush", name: "Alex Brush (brush-pen calligraphy)", data: fontAlexBrush},
}

func lookupFont(id string) (*cursiveFont, error) {
	for i := range cursiveFonts {
		if cursiveFonts[i].id == id {
			return &cursiveFonts[i], nil
		}
	}
	return nil, fmt.Errorf("unknown font %q - choices: 1 (Sacramento), 2 (Great Vibes), 3 (Qwigley), 4 (Satisfy), 5 (Allura), 6 (Alex Brush)", id)
}

// loadCursiveFace looks up fontID and parses it into a font.Face at the
// given size, rendered at 72 DPI so 1pt == 1px, matching the point
// convention used throughout qwiksi. Callers must Close() the face.
func loadCursiveFace(fontID string, size float64) (face font.Face, cf *cursiveFont, err error) {
	cf, err = lookupFont(fontID)
	if err != nil {
		return nil, nil, err
	}

	sfntFont, err := opentype.Parse(cf.data)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing font: %w", err)
	}
	face, err = opentype.NewFace(sfntFont, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("creating font face: %w", err)
	}

	return face, cf, nil
}
