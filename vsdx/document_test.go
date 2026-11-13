package vsdx

import "testing"

func TestOpenFixture(t *testing.T) {
	doc, err := Open("../testdata/vsdx/bpmn-sample.vsdx")
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Pages) != 1 {
		t.Fatalf("pages = %d, want 1", len(doc.Pages))
	}
	if len(doc.Pages[0].Shapes) != 9 {
		t.Fatalf("shapes = %d, want 9", len(doc.Pages[0].Shapes))
	}
	width, ok := doc.Pages[0].Shapes[0].Number("Width")
	if !ok || width <= 0 {
		t.Fatalf("first shape width = %v, want positive", width)
	}
}
