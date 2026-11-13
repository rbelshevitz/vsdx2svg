# vsdx2svg

`vsdx2svg` converts Microsoft Visio `.vsdx` files to editable SVG for Inkscape.

## Build

Requires Go 1.22 or newer.

```bash
go mod download
go build -o vsdx2svg ./cmd/vsdx2svg
```

## Usage

```bash
./vsdx2svg -input diagram.vsdx -output diagram.svg
```

Options:

- `-page N` — convert page `N` (default: `1`)
- `-no-rounding` — disable rounded corners from Visio shape settings

The generated SVG keeps shapes, text, fills, and strokes as separate objects for editing in Inkscape.

## Test

```bash
go test ./...
```

MIT License.
