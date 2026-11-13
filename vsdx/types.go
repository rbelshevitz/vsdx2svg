package vsdx

import "strconv"

// Page is a Visio drawing page.
type Page struct {
	Name     string
	Shapes   []Shape
	Connects []Connect
}

// Shape contains page-local ShapeSheet values and geometry rows.
type Shape struct {
	ID       string    `xml:"ID,attr"`
	Name     string    `xml:"Name,attr"`
	NameU    string    `xml:"NameU,attr"`
	Type     string    `xml:"Type,attr"`
	Master   string    `xml:"Master,attr"`
	Cells    []Cell    `xml:"Cell"`
	Sections []Section `xml:"Section"`
	Text     string    `xml:"Text"`
	Children []Shape   `xml:"Shapes>Shape"`
}

type Cell struct {
	Name    string `xml:"N,attr"`
	Value   string `xml:"V,attr"`
	Unit    string `xml:"U,attr"`
	Formula string `xml:"F,attr"`
}

type Section struct {
	Name string `xml:"N,attr"`
	Rows []Row  `xml:"Row"`
}

type Row struct {
	Type  string `xml:"T,attr"`
	Index string `xml:"IX,attr"`
	Cells []Cell `xml:"Cell"`
}

type Connect struct {
	FromSheet string `xml:"FromSheet,attr"`
	FromCell  string `xml:"FromCell,attr"`
	ToSheet   string `xml:"ToSheet,attr"`
	ToCell    string `xml:"ToCell,attr"`
}

func (s Shape) Cell(name string) (Cell, bool) {
	for _, cell := range s.Cells {
		if cell.Name == name {
			return cell, true
		}
	}
	return Cell{}, false
}

func (s Shape) Number(name string) (float64, bool) {
	cell, ok := s.Cell(name)
	if !ok {
		return 0, false
	}
	n, err := strconv.ParseFloat(cell.Value, 64)
	return n, err == nil
}

func (s Shape) Geometry() []Section {
	var result []Section
	for _, section := range s.Sections {
		if section.Name == "Geometry" {
			result = append(result, section)
		}
	}
	return result
}
