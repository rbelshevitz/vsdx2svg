package main

import (
	"bytes"
	"flag"
	"fmt"
	"html"
	"os"
	"strconv"
	"strings"

	"github.com/beevik/etree"
	native "github.com/wijnberg-net/vsdx-go/vsdx"
)

func main() {
	input := flag.String("input", "", "input .vsdx file")
	output := flag.String("output", "", "output .svg file")
	pageIndex := flag.Int("page", 1, "1-based page number")
	noRounding := flag.Bool("no-rounding", false, "disable Visio corner rounding")
	flag.Parse()
	if *input == "" || *output == "" {
		fmt.Fprintln(os.Stderr, "usage: vsdx2svg -input diagram.vsdx -output diagram.svg [-page 1]")
		os.Exit(2)
	}

	vis, err := native.Open(*input)
	if err != nil {
		fatal(err)
	}
	defer vis.Close()
	page := vis.GetPage(*pageIndex - 1)
	if page == nil {
		fatal(fmt.Errorf("page %d is out of range (document has %d pages)", *pageIndex, len(vis.GetPageNames())))
	}
	svg, err := renderPage(page, *noRounding)
	if err != nil {
		fatal(err)
	}
	if err := os.WriteFile(*output, svg, 0644); err != nil {
		fatal(err)
	}
}

func renderPage(page *native.Page, noRounding bool) ([]byte, error) {
	const scale = 100.0
	pageWidth, pageHeight := page.Width(), page.Height()
	minX, minY := 0.0, 0.0
	if pageWidth <= 0 || pageHeight <= 0 {
		pageWidth, pageHeight, minX, minY = fallbackPageBounds(page)
	}
	if pageWidth <= 0 || pageHeight <= 0 {
		return nil, fmt.Errorf("invalid page size %.3fx%.3f", pageWidth, pageHeight)
	}

	var out bytes.Buffer
	width, height := pageWidth*scale, pageHeight*scale
	fmt.Fprintf(&out, `<svg xmlns="http://www.w3.org/2000/svg" xmlns:inkscape="http://www.inkscape.org/namespaces/inkscape" viewBox="0 0 %.3f %.3f" width="%.0f" height="%.0f">`, width, height, width, height)

	for _, shape := range page.ChildShapes() {
		if noRounding {
			shape.SetRounding(0)
			for _, child := range shape.AllShapes() {
				child.SetRounding(0)
			}
		}
		result, err := native.ShapeToSVG(shape,
			native.WithSize(abs(shape.Width())*scale, abs(shape.Height())*scale),
			native.WithPrecision(3),
		)
		if err != nil {
			return nil, fmt.Errorf("render shape %s: %w", shape.ID, err)
		}
		left := shape.X() - abs(shape.Width())/2 - minX
		top := pageHeight - (shape.Y() + abs(shape.Height())/2 - minY)
		inner, err := editableShapeSVG(result.SVG, left*scale, top*scale, result.Width, result.Height, shape.ID)
		if err != nil {
			return nil, fmt.Errorf("prepare SVG for shape %s: %w", shape.ID, err)
		}
		out.WriteString(inner)
	}

	out.WriteString(`</svg>`)
	return out.Bytes(), nil
}

func fallbackPageBounds(page *native.Page) (width, height, minX, minY float64) {
	const margin = 0.25
	first := true
	var maxX, maxY float64
	for _, shape := range page.ChildShapes() {
		left := shape.X() - abs(shape.Width())/2
		right := shape.X() + abs(shape.Width())/2
		bottom := shape.Y() - abs(shape.Height())/2
		top := shape.Y() + abs(shape.Height())/2
		if first {
			minX, minY, maxX, maxY = left, bottom, right, top
			first = false
			continue
		}
		if left < minX {
			minX = left
		}
		if bottom < minY {
			minY = bottom
		}
		if right > maxX {
			maxX = right
		}
		if top > maxY {
			maxY = top
		}
	}
	if first {
		return 0, 0, 0, 0
	}
	minX -= margin
	minY -= margin
	maxX += margin
	maxY += margin
	return maxX - minX, maxY - minY, minX, minY
}

func editableShapeSVG(svg []byte, x, y, width, height float64, shapeID string) (string, error) {
	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(svg); err != nil {
		return "", err
	}
	root := doc.Root()
	if root == nil || root.Tag != "svg" {
		return "", fmt.Errorf("missing SVG root")
	}

	graphics := make([]*etree.Element, 0)
	for _, child := range root.ChildElements() {
		if child.Tag != "defs" {
			graphics = append(graphics, child)
		}
	}
	if len(graphics) == 0 {
		return "", nil
	}

	group := root.CreateElement("g")
	group.CreateAttr("transform", fmt.Sprintf("translate(%s %s)", format(x), format(y)))
	group.CreateAttr("id", fmt.Sprintf("visio-%s", shapeID))
	group.CreateAttr("inkscape:label", fmt.Sprintf("Visio shape %s", shapeID))
	group.CreateAttr("pointer-events", "all")

	fill, stroke := "", ""
	for _, child := range graphics {
		group.AddChild(child)
		if fill == "" {
			if value, ok := firstPaint(child, "fill"); ok {
				fill = value
			}
		}
		if stroke == "" {
			if value, ok := firstPaint(child, "stroke"); ok {
				stroke = value
			}
		}
		normalizePaintInheritance(child)
	}
	if fill == "" {
		fill = "none"
	}
	if stroke == "" {
		stroke = "none"
	}
	hitbox := group.CreateElement("rect")
	hitbox.CreateAttr("x", "0")
	hitbox.CreateAttr("y", "0")
	hitbox.CreateAttr("width", format(width))
	hitbox.CreateAttr("height", format(height))
	hitbox.CreateAttr("fill", "none")
	hitbox.CreateAttr("stroke", "none")
	hitbox.CreateAttr("pointer-events", "all")
	group.CreateAttr("fill", fill)
	group.CreateAttr("stroke", stroke)

	result, err := doc.WriteToString()
	if err != nil {
		return "", err
	}
	rootStart := strings.Index(result, "<svg")
	if rootStart < 0 {
		return "", fmt.Errorf("missing serialized SVG root")
	}
	contentStart := strings.IndexByte(result[rootStart:], '>')
	contentEnd := strings.LastIndex(result, "</svg>")
	if contentStart < 0 || contentEnd < 0 {
		return "", fmt.Errorf("invalid serialized SVG")
	}
	contentStart += rootStart + 1
	return result[contentStart:contentEnd], nil
}

func firstPaint(element *etree.Element, name string) (string, bool) {
	if attr := element.SelectAttr(name); attr != nil {
		return attr.Value, true
	}
	for _, child := range element.ChildElements() {
		if value, ok := firstPaint(child, name); ok {
			return value, true
		}
	}
	return "", false
}

func normalizePaintInheritance(element *etree.Element) {
	switch element.Tag {
	case "path", "rect", "ellipse", "circle", "polygon", "polyline":
		if element.SelectAttr("fill") != nil {
			element.RemoveAttr("fill")
		}
		if element.SelectAttr("stroke") != nil {
			element.RemoveAttr("stroke")
		}
	case "text", "tspan":
		element.CreateAttr("stroke", "none")
	}
	for _, child := range element.ChildElements() {
		normalizePaintInheritance(child)
	}
}

func abs(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}

func format(value float64) string {
	return html.EscapeString(strconv.FormatFloat(value, 'f', 3, 64))
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
