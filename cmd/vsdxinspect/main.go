package main

import (
	"archive/zip"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"io"
	"path"
	"regexp"
	"sort"
	"strings"
)

var numberedPage = regexp.MustCompile(`^page[0-9]+\.xml$`)
var numberedMaster = regexp.MustCompile(`^master[0-9]+\.xml$`)

type report struct {
	pages   []string
	masters []string
	media   []string
	counts  map[string]int
}

func main() {
	flag.Usage = func() {
		fmt.Println("Usage: vsdxinspect file.vsdx")
	}
	flag.Parse()
	if flag.NArg() != 1 {
		flag.Usage()
		return
	}

	r, err := inspect(flag.Arg(0))
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return
	}

	fmt.Printf("Pages:   %d\n", len(r.pages))
	for _, name := range r.pages {
		fmt.Printf("  %s\n", name)
	}
	fmt.Printf("Masters: %d\n", len(r.masters))
	for _, name := range r.masters {
		fmt.Printf("  %s\n", name)
	}
	fmt.Printf("Media:   %d\n", len(r.media))
	for _, name := range r.media {
		fmt.Printf("  %s\n", name)
	}

	keys := make([]string, 0, len(r.counts))
	for key := range r.counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	fmt.Println("XML elements:")
	for _, key := range keys {
		fmt.Printf("  %-16s %d\n", key, r.counts[key])
	}
}

func inspect(filename string) (report, error) {
	zr, err := zip.OpenReader(filename)
	if err != nil {
		return report{}, err
	}
	defer zr.Close()

	r := report{counts: make(map[string]int)}
	for _, file := range zr.File {
		name := path.Clean(file.Name)
		base := path.Base(name)
		switch {
		case strings.HasPrefix(name, "visio/pages/") && numberedPage.MatchString(base):
			r.pages = append(r.pages, name)
		case strings.HasPrefix(name, "visio/masters/") && numberedMaster.MatchString(base):
			r.masters = append(r.masters, name)
		case strings.HasPrefix(name, "visio/media/"):
			r.media = append(r.media, name)
		}

		if strings.HasSuffix(name, ".xml") {
			if err := countXML(file, r.counts); err != nil {
				return report{}, fmt.Errorf("parse %s: %w", name, err)
			}
		}
	}

	sort.Strings(r.pages)
	sort.Strings(r.masters)
	sort.Strings(r.media)
	return r, nil
}

func countXML(file *zip.File, counts map[string]int) error {
	rc, err := file.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	decoder := xml.NewDecoder(rc)
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		if start, ok := token.(xml.StartElement); ok {
			counts[start.Name.Local]++
		}
	}
}
