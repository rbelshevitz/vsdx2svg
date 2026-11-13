package vsdx

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"
)

// Document is the readable subset of a VSDX package used by the renderer.
type Document struct {
	Pages []Page
}

// Open reads page XML from a VSDX ZIP package. Masters and relationships are
// intentionally left for the next layer; page-local values are enough for the
// first renderer spike.
func Open(filename string) (*Document, error) {
	zr, err := zip.OpenReader(filename)
	if err != nil {
		return nil, err
	}
	defer zr.Close()

	files := make(map[string]*zip.File)
	var names []string
	for _, file := range zr.File {
		name := path.Clean(file.Name)
		files[name] = file
		if strings.HasPrefix(name, "visio/pages/page") && strings.HasSuffix(name, ".xml") {
			base := path.Base(name)
			if base != "pages.xml" {
				names = append(names, name)
			}
		}
	}
	sort.Strings(names)

	doc := &Document{Pages: make([]Page, 0, len(names))}
	for _, name := range names {
		page, err := decodePage(files[name])
		if err != nil {
			return nil, fmt.Errorf("decode %s: %w", name, err)
		}
		page.Name = name
		doc.Pages = append(doc.Pages, page)
	}
	return doc, nil
}

func decodePage(file *zip.File) (Page, error) {
	rc, err := file.Open()
	if err != nil {
		return Page{}, err
	}
	defer rc.Close()

	var raw pageXML
	if err := xml.NewDecoder(rc).Decode(&raw); err != nil && err != io.EOF {
		return Page{}, err
	}
	return Page{Shapes: raw.Shapes, Connects: raw.Connects}, nil
}

type pageXML struct {
	Shapes   []Shape   `xml:"Shapes>Shape"`
	Connects []Connect `xml:"Connects>Connect"`
}
