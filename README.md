# qwiksi

qwiksi is a CLI tool for signing PDFs without opening a UI. Point it at a PDF and it signs it.

- Detects and lists fillable form fields
- Manual x/y coordinates for flat or scanned PDFs with no form fields
- Can generate a signature image from your name in a cursive font

## Table of Contents

- [Install](#install)
  - [Download a prebuilt binary](#download-a-prebuilt-binary)
  - [Build from source](#build-from-source)
- [Usage](#usage)
  - [Basic Operation](#basic-operation)
  - [1. Don't have a signature image? Generate one](#1-dont-have-a-signature-image-generate-one)
  - [2. Check for existing form fields](#2-check-for-existing-form-fields)
  - [3. Find coordinates](#3-find-coordinates)
  - [4. Sign](#4-sign)
    - [Singing using manual coordinate mode](#singing-using-manual-coordinate-mode)
- [Notes](#notes)

## Install

### Download a prebuilt binary

Download the archive for your platform from the [latest release](https://github.com/krisraven/qwiksi/releases/latest), extract it and put `qwiksi` in your `$PATH`.

```
# macOS (Apple Silicon)
curl -L https://github.com/krisraven/qwiksi/releases/latest/download/qwiksi_darwin_arm64.tar.gz | tar xz
sudo mv qwiksi /usr/local/bin/

# macOS (Intel)
curl -L https://github.com/krisraven/qwiksi/releases/latest/download/qwiksi_darwin_amd64.tar.gz | tar xz
sudo mv qwiksi /usr/local/bin/

# Linux (x86_64)
curl -L https://github.com/krisraven/qwiksi/releases/latest/download/qwiksi_linux_amd64.tar.gz | tar xz
sudo mv qwiksi /usr/local/bin/
```
`arm64` archives are also published for Linux, and Windows builds (`amd64`/`arm64`) are available as `.zip` - see the [releases page](https://github.com/krisraven/qwiksi/releases) for the full list. Each release includes a `checksums.txt` to verify your download against.

### Build from source

Requires Go 1.25+.

```
go build -o qwiksi .
```

Produces a single static binary with no runtime dependencies.

## Usage

### Basic Operation

Most modern PDF documents use AcroForm fields, and hopefully the fields are labelled sensibly (a box that needs a signature is named **Signature**, and so on).

If your PDF is using this format, it means that you can just type `qwiksi` and then follow the prompts. 

If the boxes are not named sensibly, then you'll have to [get the fields](#2-check-for-existing-form-fields).

If the PDF doesn't have AcroFields (if is flat or a scanned document), then you can use [manual mode](#singing-using-manual-coordinate-mode). You'll first need to [get the coordinates](#3-find-coordinates).

### 1. Don't have a signature image? Generate one

```
qwiksi addsig --text "Your Name" [--font 1-6] [--size 100] [--color 000000] [--out signature.png]
```

Renders your name in a cursive font as a transparent-background PNG, ready to
feed into `sign --signature` below. Six bundled fonts are embedded in the
binary, no external files needed. Or skip this step entirely - `sign` below
accepts `--text` directly and renders the signature on the fly, no PNG file
in between.

| `--font` | Font | Style |
|---|---|---|
| `1` (default) | Sacramento | casual, flowing |
| `2` | Great Vibes | elegant, formal |
| `3` | Qwigley | tall, looping monoline |
| `4` | Satisfy | relaxed marker script |
| `5` | Allura | delicate, calligraphic |
| `6` | Alex Brush | brush-pen calligraphy |

`--color` is a 6-digit hex RGB value (default `000000`, black).

Not sure which one you want? Generate a one-page specimen PDF with every
bundled font rendering its own name in itself:

```
qwiksi fontdemo [--out fontdemo.pdf] [--size 44] [--color 000000]
```

### 2. Check for existing form fields

If the PDF is already a fillable form, it may have a signature field you can
target by name:

```
qwiksi fields input.pdf
```

This prints each field's name, type, page numer and rect (the field's bounding box on the page ). 

If there are none, you should use the coordinate-based flow, out-lined below, instead.

### 3. Find coordinates

```
qwiksi preview input.pdf --page 1 [--grid 50] [--out preview.pdf]
```

Stamps a labeled coordinate grid onto a copy of the given page and writes it
to `preview.pdf` (default: `<input>_preview.pdf`). Open it in any PDF viewer
to read off where you want the signature. Coordinates are in PDF points,
measured from the **bottom-left** corner of the page, matching what `sign
--x --y` expects below.

### 4. Sign

If the PDF has AcroSign fields `qwiksi` will place the image into the named AcroForm field. The signature is scaled to fit and centered within it:

```
qwiksi sign input.pdf --signature sig.png --field "Signature" [--output signed.pdf]
```

#### Singing using manual coordinate mode

Using this the image can be placed at an absolute position:

```
qwiksi sign input.pdf --signature sig.png --page 1 --x 150 --y 100 [--width 150] [--height 100] [--output signed.pdf]
```

`--width`/`--height` are optional; default is 150pt wide with height derived
from the signature image's aspect ratio. Output defaults to
`<input>_signed.pdf`.


## Notes

- `--signature` accepts PNG or JPEG. For best results, use a PNG signature image with a transparent background (or use `addsig` to [create one](#1-dont-have-a-signature-image-generate-one))
- Coordinates and dimensions are all in PDF points (1/72 inch), origin at
  the page's bottom-left corner. The same convention is used in `preview` when the grid is drawn, so numbers in the preview PDF map directly onto `sign` flags,
- Built using [pdfcpu](https://pdfcpu.io),
- The bundled fonts are Google Fonts. Sacramento, Great Vibes, Qwigley,
  Allura and Alex Brush are under the SIL Open Font License 1.1; Satisfy is
  under the Apache License 2.0.
