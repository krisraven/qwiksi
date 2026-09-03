package main

import (
	"fmt"
	"image"
	_ "image/jpeg"
	"image/png"
	"os"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/form"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// imageDimsPx returns an image file's native pixel dimensions without
// fully decoding it. pdfcpu treats those pixel counts as points when it
// builds an image watermark's bounding box, so this is what "scalefactor"
// math below is relative to.
func imageDimsPx(path string) (w, h int, err error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()

	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return 0, 0, fmt.Errorf("reading image %s: %w", path, err)
	}
	return cfg.Width, cfg.Height, nil
}

// prepareInput returns the path qwiksi's pdfcpu calls should actually read:
// inFile itself when it already validates, or a repaired temp copy when the
// only problem is a missing/empty AcroForm-level /DA. Real-world PDF
// generators routinely omit it (relying on viewers to fall back to a
// default appearance), but pdfcpu's validator rejects it even in relaxed
// mode - see https://github.com/pdfcpu/pdfcpu/issues/1274 for the same gap
// with the sibling /FT entry. Any other validation failure is returned
// unchanged so real problems still surface. The returned cleanup always
// runs safely, removing the temp file if one was created.
func prepareInput(inFile string) (path string, cleanup func(), err error) {
	noop := func() {}

	f, err := os.Open(inFile)
	if err != nil {
		return "", noop, err
	}
	defer f.Close()

	ctx, err := api.ReadContext(f, model.NewDefaultConfiguration())
	if err != nil {
		return "", noop, err
	}

	origErr := api.ValidateContext(ctx)
	if origErr == nil {
		return inFile, noop, nil
	}

	root, catErr := ctx.XRefTable.Catalog()
	if catErr != nil {
		return "", noop, origErr
	}
	acroFormObj, ok := root.Find("AcroForm")
	if !ok {
		return "", noop, origErr
	}
	acroForm, derefErr := ctx.XRefTable.DereferenceDict(acroFormObj)
	if derefErr != nil || acroForm == nil {
		return "", noop, origErr
	}

	sl := acroForm.StringLiteralEntry("DA")
	hasUsableDA := acroForm.HasEntry("DA") && !(sl != nil && string(*sl) == "")
	if hasUsableDA {
		// AcroForm already has a DA, so this isn't the gap we know how to
		// repair - the failure is something else.
		return "", noop, origErr
	}
	acroForm["DA"] = types.StringLiteral("/Helv 0 Tf 0 g")

	if err := api.ValidateContext(ctx); err != nil {
		// Repair didn't help - report the original failure.
		return "", noop, origErr
	}

	tmp, err := os.CreateTemp("", "qwiksi-repaired-*.pdf")
	if err != nil {
		return "", noop, origErr
	}
	tmpPath := tmp.Name()
	cleanup = func() { os.Remove(tmpPath) }

	if err := api.WriteContext(ctx, tmp); err != nil {
		tmp.Close()
		cleanup()
		return "", noop, origErr
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return "", noop, origErr
	}

	return tmpPath, cleanup, nil
}

// fieldWidget holds one candidate widget annotation for a field name: its
// dict (carrying /Rect) and that dict's own object number, which identifies
// the widget annotation itself - callers need it to strip the annotation
// after stamping (see sign.go: otherwise the widget's own appearance
// renders on top of the page content and hides the stamp).
type fieldWidget struct {
	dict  types.Dict
	objNr int
}

// fieldWidgetDict locates the widget for the named AcroForm field: either
// the field dict itself (common case: a single, unsplit field), or one of
// its /Kids entries (a field split into one widget per page it appears on).
// When a field has more than one widget, desiredPage (1-based) selects
// which one; 0 picks the first one found.
func fieldWidgetDict(xRefTable *model.XRefTable, fieldName string, desiredPage int) (types.Dict, int, error) {
	arr, err := form.Fields(xRefTable)
	if err != nil {
		return nil, 0, fmt.Errorf("reading form fields: %w", err)
	}

	for _, obj := range arr {
		ir, ok := obj.(types.IndirectRef)
		if !ok {
			continue
		}
		d, err := xRefTable.DereferenceDict(ir)
		if err != nil || d == nil {
			continue
		}

		name, err := d.StringOrHexLiteralEntry("T")
		if err != nil || name == nil || *name != fieldName {
			continue
		}

		var candidates []fieldWidget

		if d.HasEntry("Rect") {
			candidates = append(candidates, fieldWidget{d, ir.ObjectNumber.Value()})
		}

		if kidsObj, err := xRefTable.DereferenceDictEntry(d, "Kids"); err == nil {
			if kids, err := xRefTable.DereferenceArray(kidsObj); err == nil {
				for _, k := range kids {
					kidIR, ok := k.(types.IndirectRef)
					if !ok {
						continue
					}
					if kd, err := xRefTable.DereferenceDict(kidIR); err == nil && kd.HasEntry("Rect") {
						candidates = append(candidates, fieldWidget{kd, kidIR.ObjectNumber.Value()})
					}
				}
			}
		}

		if len(candidates) == 0 {
			return nil, 0, fmt.Errorf("field %q has no widget annotation with a /Rect", fieldName)
		}
		if desiredPage <= 0 {
			return candidates[0].dict, candidates[0].objNr, nil
		}

		for _, c := range candidates {
			if _, pn, err := rectAndPage(xRefTable, c.dict, c.objNr); err == nil && pn == desiredPage {
				return c.dict, c.objNr, nil
			}
		}
		return nil, 0, fmt.Errorf("field %q has no widget on page %d", fieldName, desiredPage)
	}

	return nil, 0, fmt.Errorf("no such form field: %q", fieldName)
}

// rectAndPage resolves a widget dict's /Rect and the 1-based page number it
// sits on. It tries the widget's /P entry first, then - since many PDF
// generators omit /P even though it's only ever been optional - falls back
// to scanning every page's /Annots array for the widget's object number.
func rectAndPage(xRefTable *model.XRefTable, d types.Dict, objNr int) (*types.Rectangle, int, error) {
	rectObj, err := xRefTable.DereferenceDictEntry(d, "Rect")
	if err != nil {
		return nil, 0, fmt.Errorf("field has no /Rect: %w", err)
	}
	rectArr, err := xRefTable.DereferenceArray(rectObj)
	if err != nil {
		return nil, 0, fmt.Errorf("field /Rect is not an array: %w", err)
	}
	rect := types.RectForArray(rectArr)

	if ir, ok := d["P"].(types.IndirectRef); ok {
		if n, err := xRefTable.PageNumber(ir.ObjectNumber.Value()); err == nil && n > 0 {
			return rect, n, nil
		}
	}

	if err := xRefTable.EnsurePageCount(); err != nil {
		return nil, 0, fmt.Errorf("could not determine field's page: %w", err)
	}
	for p := 1; p <= xRefTable.PageCount; p++ {
		pd, _, _, err := xRefTable.PageDict(p, false)
		if err != nil || pd == nil {
			continue
		}
		annotsObj, found := pd.Find("Annots")
		if !found {
			continue
		}
		annots, err := xRefTable.DereferenceArray(annotsObj)
		if err != nil {
			continue
		}
		for _, a := range annots {
			if air, ok := a.(types.IndirectRef); ok && air.ObjectNumber.Value() == objNr {
				return rect, p, nil
			}
		}
	}

	return nil, 0, fmt.Errorf("could not determine field's page (no /P entry, and not listed in any page's /Annots)")
}

// loadFormFields returns a PDF's AcroForm fields, treating "no form
// available" as zero fields rather than an error - shared by `fields` and
// interactive mode, both of which need to tell a flat PDF from a form apart.
func loadFormFields(ctx *model.Context) ([]form.Field, error) {
	fields, _, err := form.FormFields(ctx)
	if err != nil && !strings.Contains(err.Error(), "no form available") {
		return nil, fmt.Errorf("reading form fields: %w", err)
	}
	return fields, nil
}

// printFieldsTable prints the same NAME/TYPE/PAGE/RECT listing used by the
// `fields` command - shared with interactive mode.
func printFieldsTable(ctx *model.Context, fields []form.Field) {
	fmt.Printf("%-24s %-16s %-6s %s\n", "NAME", "TYPE", "PAGE", "RECT (llx lly urx ury)")
	for _, f := range fields {
		rectStr := "?"
		if d, objNr, err := fieldWidgetDict(ctx.XRefTable, f.Name, 0); err == nil {
			if rect, _, err := rectAndPage(ctx.XRefTable, d, objNr); err == nil {
				rectStr = fmt.Sprintf("%.0f %.0f %.0f %.0f", rect.LL.X, rect.LL.Y, rect.UR.X, rect.UR.Y)
			}
		}
		fmt.Printf("%-24s %-16s %-6v %s\n", f.Name, f.Typ.String(), f.Pages, rectStr)
	}
}

// fieldPlacement resolves the named AcroForm field's page and the
// dx/dy/scale needed to stamp an imgW x imgH image scaled to fit and
// centered within the field's rect. desiredPage (1-based, 0 for "don't
// care") disambiguates a field that has a widget on more than one page.
func fieldPlacement(inFile, fieldName string, desiredPage, imgW, imgH int) (dx, dy, scale float64, pageNr, widgetObjNr int, err error) {
	ctx, err := api.ReadContextFile(inFile)
	if err != nil {
		return 0, 0, 0, 0, 0, fmt.Errorf("reading %s: %w", inFile, err)
	}
	d, objNr, err := fieldWidgetDict(ctx.XRefTable, fieldName, desiredPage)
	if err != nil {
		return 0, 0, 0, 0, 0, err
	}
	rect, pageNr, err := rectAndPage(ctx.XRefTable, d, objNr)
	if err != nil {
		return 0, 0, 0, 0, 0, err
	}

	scale = rect.Width() / float64(imgW)
	if hScale := rect.Height() / float64(imgH); hScale < scale {
		scale = hScale
	}
	finalW := scale * float64(imgW)
	finalH := scale * float64(imgH)
	dx = rect.LL.X + (rect.Width()-finalW)/2
	dy = rect.LL.Y + (rect.Height()-finalH)/2

	return dx, dy, scale, pageNr, objNr, nil
}

// stampSignature stamps sigPath onto inFile at the given position/scale and
// writes the result to outFile. When fieldMode is set, it also strips the
// signed field's widget annotation (object number widgetObjNr) afterward -
// otherwise the widget's own appearance renders on top of the page content
// and hides the stamp (see sign.go's field-mode comment for the full story).
func stampSignature(inFile, sigPath, outFile string, dx, dy, scale float64, pageNr int, fieldMode bool, widgetObjNr int) error {
	wm, err := api.ImageWatermark(sigPath, stampDescriptor(dx, dy, scale), true, false, types.POINTS)
	if err != nil {
		return fmt.Errorf("building signature watermark: %w", err)
	}

	if err := api.AddWatermarksFile(inFile, outFile, []string{fmt.Sprintf("%d", pageNr)}, wm, nil); err != nil {
		return fmt.Errorf("stamping signature onto page %d: %w", pageNr, err)
	}

	if fieldMode {
		if err := api.RemoveAnnotationsFile(outFile, outFile, nil, nil, []int{widgetObjNr}, nil, false); err != nil {
			return fmt.Errorf("removing signed field's widget annotation: %w", err)
		}
	}

	return nil
}

// writeTempSignaturePNG renders text as a signature image (see
// renderSignatureImage) and writes it to a temp PNG file, returning its
// path. Caller is responsible for os.Remove'ing it.
func writeTempSignaturePNG(text, fontID string, size float64, colorHex string) (path string, err error) {
	col, err := parseHexColor(colorHex)
	if err != nil {
		return "", err
	}
	img, _, err := renderSignatureImage(text, fontID, size, col)
	if err != nil {
		return "", err
	}
	f, err := os.CreateTemp("", "qwiksi-sig-*.png")
	if err != nil {
		return "", fmt.Errorf("creating temp signature file: %w", err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		return "", fmt.Errorf("encoding rendered signature: %w", err)
	}
	if err := f.Close(); err != nil {
		return "", fmt.Errorf("writing temp signature file: %w", err)
	}
	return f.Name(), nil
}

// stampDescriptor builds a pdfcpu image-watermark description string that
// places an image at an absolute page position (bottom-left-origin, in
// points) and absolute size derived from scale x the image's native pixel
// dimensions (pdfcpu treats 1 native pixel as 1 point for "scalefactor ... abs").
func stampDescriptor(x, y, scale float64) string {
	// rotation:0 is required - pdfcpu's watermark default is a diagonal
	// banner placement (Diagonal: DiagonalLLToUR) unless a rotation or
	// diagonal setting is explicitly given.
	return fmt.Sprintf("position:bl, offset:%g %g, scalefactor:%g abs, opacity:1, rotation:0", x, y, scale)
}
