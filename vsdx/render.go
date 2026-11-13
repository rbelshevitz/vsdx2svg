package vsdx

import (
	"bytes"
	"fmt"
	"html"
	"math"
	"strconv"
	"strings"
)

// RenderPage renders the currently supported page-local geometry to SVG.
// Coordinates are kept in Visio's millimetre-like drawing units.
func RenderPage(page Page) ([]byte, error) {
	width, height := pageBounds(page)
	var out bytes.Buffer
	fmt.Fprintf(&out, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %.4f %.4f" width="%.4f" height="%.4f">`, width, height, width, height)
	fmt.Fprintf(&out, `<rect width="100%%" height="100%%" fill="white"/>`)
	for _, shape := range page.Shapes {
		renderShape(&out, shape, height)
	}
	out.WriteString(`</svg>`)
	return out.Bytes(), nil
}

func pageBounds(page Page) (float64, float64) {
	maxX, maxY := 1.0, 1.0
	for _, shape := range page.Shapes {
		x, _ := shape.Number("PinX")
		y, _ := shape.Number("PinY")
		w, _ := shape.Number("Width")
		h, _ := shape.Number("Height")
		locX, ok := shape.Number("LocPinX")
		if !ok {
			locX = w / 2
		}
		locY, ok := shape.Number("LocPinY")
		if !ok {
			locY = h / 2
		}
		maxX = math.Max(maxX, x-locX+w)
		maxY = math.Max(maxY, y-locY+h)
	}
	return maxX + 0.25, maxY + 0.25
}

func renderShape(out *bytes.Buffer, shape Shape, canvasHeight float64) {
	x, _ := shape.Number("PinX")
	y, _ := shape.Number("PinY")
	w, _ := shape.Number("Width")
	h, _ := shape.Number("Height")
	locX, ok := shape.Number("LocPinX")
	if !ok {
		locX = w / 2
	}
	locY, ok := shape.Number("LocPinY")
	if !ok {
		locY = h / 2
	}
	left, bottom := x-locX, y-locY
	fill := cellValue(shape, "FillForegnd", "FillBkgnd")
	line := cellValue(shape, "LineColor")
	if fill == "" {
		fill = "none"
	}
	if line == "" {
		line = "black"
	}
	strokeWidth := 1.0
	if value, ok := shape.Number("LineWeight"); ok && value > 0 {
		strokeWidth = value
	}

	geometry := shape.Geometry()
	if len(geometry) == 0 {
		writeRect(out, left, bottom, w, h, canvasHeight, fill, line, strokeWidth)
	} else {
		for _, section := range geometry {
			if len(section.Rows) == 0 {
				continue
			}
			if section.Rows[0].Type == "Ellipse" {
				writeEllipse(out, left, bottom, w, h, canvasHeight, fill, line, strokeWidth)
			} else {
				writePath(out, section.Rows, left, bottom, canvasHeight, fill, line, strokeWidth)
			}
		}
	}

	text := strings.TrimSpace(shape.Text)
	if text != "" {
		textX := left + w/2
		textY := canvasHeight - (bottom + h/2)
		fmt.Fprintf(out, `<text x="%.4f" y="%.4f" text-anchor="middle" dominant-baseline="middle" font-family="Arial, sans-serif" font-size="0.16" fill="%s">%s</text>`, textX, textY, html.EscapeString(textColor(shape)), html.EscapeString(text))
	}
	for _, child := range shape.Children {
		renderShape(out, child, canvasHeight)
	}
}

func writeRect(out *bytes.Buffer, x, y, w, h, canvasHeight float64, fill, line string, strokeWidth float64) {
	fmt.Fprintf(out, `<rect x="%.4f" y="%.4f" width="%.4f" height="%.4f" fill="%s" stroke="%s" stroke-width="%.4f"/>`, x, canvasHeight-y-h, w, h, html.EscapeString(fill), html.EscapeString(line), strokeWidth)
}

func writeEllipse(out *bytes.Buffer, x, y, w, h, canvasHeight float64, fill, line string, strokeWidth float64) {
	fmt.Fprintf(out, `<ellipse cx="%.4f" cy="%.4f" rx="%.4f" ry="%.4f" fill="%s" stroke="%s" stroke-width="%.4f"/>`, x+w/2, canvasHeight-y-h/2, w/2, h/2, html.EscapeString(fill), html.EscapeString(line), strokeWidth)
}

func writePath(out *bytes.Buffer, rows []Row, left, bottom, canvasHeight float64, fill, line string, strokeWidth float64) {
	var d strings.Builder
	for _, row := range rows {
		x, _ := row.Number("X")
		y, _ := row.Number("Y")
		px, py := left+x, canvasHeight-(bottom+y)
		switch row.Type {
		case "MoveTo":
			fmt.Fprintf(&d, "M %.4f %.4f ", px, py)
		case "LineTo":
			fmt.Fprintf(&d, "L %.4f %.4f ", px, py)
		case "Ellipse":
			// Ellipse rows are handled as a shape-level primitive.
			return
		}
	}
	if d.Len() == 0 {
		return
	}
	fmt.Fprintf(out, `<path d="%s" fill="%s" stroke="%s" stroke-width="%.4f"/>`, strings.TrimSpace(d.String()), html.EscapeString(fill), html.EscapeString(line), strokeWidth)
}

func cellValue(shape Shape, names ...string) string {
	for _, name := range names {
		if cell, ok := shape.Cell(name); ok && cell.Value != "" {
			return cell.Value
		}
	}
	return ""
}

func textColor(shape Shape) string {
	for _, section := range shape.Sections {
		if section.Name != "Character" {
			continue
		}
		for _, row := range section.Rows {
			if cell, ok := row.Cell("Color"); ok && cell.Value != "" {
				return cell.Value
			}
		}
	}
	return "black"
}

func (r Row) Number(name string) (float64, bool) {
	cell, ok := r.Cell(name)
	if !ok {
		return 0, false
	}
	n, err := strconv.ParseFloat(cell.Value, 64)
	return n, err == nil
}

func (r Row) Cell(name string) (Cell, bool) {
	for _, cell := range r.Cells {
		if cell.Name == name {
			return cell, true
		}
	}
	return Cell{}, false
}
