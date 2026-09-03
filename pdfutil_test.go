package main

import (
	"os"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
)

// TestPrepareInputRepairsMissingAcroFormDA reproduces the reported bug:
// reading Residential_Rent_To_Own_Agreement_d8f5077622.pdf fails validation
// with "dict=formFieldDict required entry=DA missing" because its AcroForm
// dict has no /DA and one of its Tx fields doesn't either. prepareInput
// should silently hand back a repaired temp copy that validates cleanly.
func TestPrepareInputRepairsMissingAcroFormDA(t *testing.T) {
	const in = "testing/Residential_Rent_To_Own_Agreement_d8f5077622.pdf"

	if _, err := api.ReadContextFile(in); err == nil {
		t.Fatalf("expected %s to fail validation unrepaired (test fixture assumption no longer holds)", in)
	}

	path, cleanup, err := prepareInput(in)
	if err != nil {
		t.Fatalf("prepareInput(%s): %v", in, err)
	}
	defer cleanup()

	if path == in {
		t.Fatalf("expected a repaired temp copy, got the original path back")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("repaired file missing: %v", err)
	}

	if _, err := api.ReadContextFile(path); err != nil {
		t.Fatalf("repaired file still fails validation: %v", err)
	}
}

// TestPrepareInputLeavesValidPDFAlone guards against prepareInput doing
// unnecessary work (or breaking anything) for a PDF that already validates.
func TestPrepareInputLeavesValidPDFAlone(t *testing.T) {
	const in = "testing/testAcroForm.pdf"

	path, cleanup, err := prepareInput(in)
	if err != nil {
		t.Fatalf("prepareInput(%s): %v", in, err)
	}
	defer cleanup()

	if path != in {
		t.Fatalf("expected the original path back for an already-valid PDF, got %s", path)
	}
}
