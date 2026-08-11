package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		if err := runInteractive(); err != nil {
			fmt.Fprintf(os.Stderr, "qwiksi: %v\n", err)
			os.Exit(1)
		}
		return
	}

	var err error
	switch os.Args[1] {
	case "fields":
		err = runFields(os.Args[2:])
	case "preview":
		err = runPreview(os.Args[2:])
	case "sign":
		err = runSign(os.Args[2:])
	case "addsig":
		err = runAddSig(os.Args[2:])
	case "-h", "--help", "help":
		usage()
		return
	default:
		fmt.Fprintf(os.Stderr, "qwiksi: unknown command %q\n\n", os.Args[1])
		usage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "qwiksi: %v\n", err)
		os.Exit(1)
	}
}

// takeInputFile pulls the leading positional <input.pdf> argument out of
// args so the rest can be handed to flag.FlagSet.Parse, which stops
// parsing at the first non-flag token - it can't handle "<file> --flags"
// order on its own.
func takeInputFile(args []string) (file string, rest []string, err error) {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return "", nil, fmt.Errorf("missing input.pdf")
	}
	return args[0], args[1:], nil
}

func usage() {
	fmt.Fprint(os.Stderr, `qwiksi - stamp a signature image onto a PDF from the command line

Usage:
  qwiksi sign    <input.pdf> --text "Your Name" [--font 1|2] [--size 100] [--color 000000] --field "FieldName" [--output out.pdf]
  qwiksi sign    <input.pdf> --signature sig.png --field "FieldName" [--output out.pdf]
  qwiksi sign    <input.pdf> --signature sig.png --page N --x X --y Y [--width W] [--height H] [--output out.pdf]
  qwiksi addsig  --text "Your Name" [--font 1|2] [--size 100] [--color 000000] [--out signature.png]
  qwiksi fields  <input.pdf>
  qwiksi preview <input.pdf> --page N [--grid 50] [--out preview.pdf]

Run "qwiksi <command> -h" for command-specific flags.
`)
}
