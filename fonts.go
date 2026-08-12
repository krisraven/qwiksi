package main

import (
	"fmt"

	_ "embed"
)

//go:embed assets/fonts/Sacramento-Regular.ttf
var fontSacramento []byte

//go:embed assets/fonts/GreatVibes-Regular.ttf
var fontGreatVibes []byte

type cursiveFont struct {
	id   string
	name string
	data []byte
}

// cursiveFonts are bundled directly into the binary (SIL Open Font
// License - see assets/fonts/OFL-*.txt)
var cursiveFonts = []cursiveFont{
	{id: "1", name: "Sacramento (casual script)", data: fontSacramento},
	{id: "2", name: "Great Vibes (formal script)", data: fontGreatVibes},
}

func lookupFont(id string) (*cursiveFont, error) {
	for i := range cursiveFonts {
		if cursiveFonts[i].id == id {
			return &cursiveFonts[i], nil
		}
	}
	return nil, fmt.Errorf("unknown font %q - choices: 1 (Sacramento), 2 (Great Vibes)", id)
}
