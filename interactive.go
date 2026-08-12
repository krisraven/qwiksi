package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/form"
)

const interactiveDefaultFont = "1"
const interactiveDefaultSize = 100
const interactiveDefaultColor = "000000"

func runInteractive() error {
	fmt.Print(`qwiksi - quick PDF signing

Other commands: fields, preview, sign, addsig (run "qwiksi <command> -h" for details)

`)

	r := bufio.NewReader(os.Stdin)

	inFile, err := prompt(r, "PDF file to sign: ")
	if err != nil {
		return err
	}
	if inFile == "" {
		return fmt.Errorf("no PDF file given")
	}
	if _, err := os.Stat(inFile); err != nil {
		return fmt.Errorf("can't read %s: %w", inFile, err)
	}

	ctx, err := api.ReadContextFile(inFile)
	if err != nil {
		return fmt.Errorf("reading %s: %w", inFile, err)
	}

	fields, err := loadFormFields(ctx)
	if err != nil {
		return err
	}

	if len(fields) == 0 {
		fmt.Printf(`No AcroForm fields found in %s - this looks like a flat PDF.
Use the coordinate-based flow instead, e.g.:

  qwiksi preview %s --page 1 [--grid 50] [--out preview.pdf]
  qwiksi sign    %s --signature sig.png --page N --x X --y Y [--width W] [--height H] [--output out.pdf]
`, inFile, inFile, inFile)
		return nil
	}

	fmt.Println()
	printFieldsTable(ctx, fields)
	fmt.Println()

	fieldName, err := prompt(r, "Which field do you want to sign? ")
	if err != nil {
		return err
	}
	if !hasField(fields, fieldName) {
		return fmt.Errorf("no such form field: %q", fieldName)
	}

	pageInput, err := prompt(r, "Page number (blank if the field only appears on one page): ")
	if err != nil {
		return err
	}
	desiredPage := 0
	if pageInput != "" {
		n, err := strconv.Atoi(pageInput)
		if err != nil || n < 1 {
			return fmt.Errorf("invalid page number: %q", pageInput)
		}
		desiredPage = n
	}

	text, err := prompt(r, "Signature text: ")
	if err != nil {
		return err
	}
	if text == "" {
		return fmt.Errorf("signature text can't be empty")
	}

	sigPath, err := writeTempSignaturePNG(text, interactiveDefaultFont, interactiveDefaultSize, interactiveDefaultColor)
	if err != nil {
		return err
	}
	defer os.Remove(sigPath)

	imgW, imgH, err := imageDimsPx(sigPath)
	if err != nil {
		return err
	}

	dx, dy, scale, pageNr, widgetObjNr, err := fieldPlacement(inFile, fieldName, desiredPage, imgW, imgH)
	if err != nil {
		return err
	}

	ext := filepath.Ext(inFile)
	outFile := strings.TrimSuffix(inFile, ext) + "_signed" + ext

	if err := stampSignature(inFile, sigPath, outFile, dx, dy, scale, pageNr, true, widgetObjNr); err != nil {
		return err
	}

	fmt.Printf("Wrote %s\n", outFile)
	return nil
}

func hasField(fields []form.Field, name string) bool {
	for _, f := range fields {
		if f.Name == name {
			return true
		}
	}
	return false
}

func prompt(r *bufio.Reader, label string) (string, error) {
	fmt.Print(label)
	line, err := r.ReadString('\n')
	if err != nil && line == "" {
		return "", err
	}
	return strings.TrimSpace(line), nil
}
