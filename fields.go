package main

import (
	"flag"
	"fmt"

	"github.com/pdfcpu/pdfcpu/pkg/api"
)

func runFields(args []string) error {
	fs := flag.NewFlagSet("fields", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Println("Usage: qwiksi fields <input.pdf>")
	}
	fs.Parse(args)

	if fs.NArg() != 1 {
		fs.Usage()
		return fmt.Errorf("missing input.pdf")
	}
	inFile := fs.Arg(0)

	ctx, err := api.ReadContextFile(inFile)
	if err != nil {
		return fmt.Errorf("reading %s: %w", inFile, err)
	}

	fields, err := loadFormFields(ctx)
	if err != nil {
		return err
	}

	if len(fields) == 0 {
		fmt.Println("No AcroForm fields found - this looks like a flat PDF. Use `qwiksi preview` + `qwiksi sign --page --x --y` instead.")
		return nil
	}

	printFieldsTable(ctx, fields)
	return nil
}
