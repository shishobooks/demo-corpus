package main

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/url"
	"path"
	"strings"
)

// epubBook is enough of an EPUB to turn "this far through" into the CFI the
// web reader saves for it. The reader (foliate) measures progress by the
// uncompressed size of each linear spine document, so the same measure
// lands a CFI where the reader would say that percentage is.
type epubBook struct {
	sections []epubSection
	total    int64
}

type epubSection struct {
	// step is the CFI path to the section's itemref in the package
	// document, such as "/6/14[chapter-3]".
	step string
	// size counts toward progress only for linear documents.
	size    int64
	content []byte
}

func openEPUB(file string) (*epubBook, error) {
	zr, err := zip.OpenReader(file)
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	entries := map[string]*zip.File{}
	for _, f := range zr.File {
		entries[f.Name] = f
	}

	var container struct {
		Rootfiles []struct {
			FullPath string `xml:"full-path,attr"`
		} `xml:"rootfiles>rootfile"`
	}
	if err := readXML(entries, "META-INF/container.xml", &container); err != nil {
		return nil, err
	}
	if len(container.Rootfiles) == 0 {
		return nil, fmt.Errorf("%s: no package document", file)
	}
	opfPath := container.Rootfiles[0].FullPath
	opf, err := readEntry(entries, opfPath)
	if err != nil {
		return nil, err
	}
	root, err := parseTree(opf)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", opfPath, err)
	}

	hrefs := map[string]string{}
	var spine *node
	var spineStep int
	for i, child := range root.children {
		switch child.name {
		case "manifest":
			for _, item := range child.children {
				hrefs[item.attr("id")] = item.attr("href")
			}
		case "spine":
			spine, spineStep = child, (i+1)*2
		}
	}
	if spine == nil {
		return nil, fmt.Errorf("%s: no spine", opfPath)
	}

	book := &epubBook{}
	for i, itemref := range spine.children {
		step := fmt.Sprintf("/%d/%d", spineStep, (i+1)*2)
		if id := itemref.attr("id"); id != "" {
			step += "[" + id + "]"
		}
		href, err := url.PathUnescape(hrefs[itemref.attr("idref")])
		if err != nil {
			return nil, err
		}
		name := path.Join(path.Dir(opfPath), href)
		content, err := readEntry(entries, name)
		if err != nil {
			return nil, err
		}
		section := epubSection{step: step, content: content}
		if itemref.attr("linear") != "no" {
			section.size = int64(entries[name].UncompressedSize64)
		}
		book.total += section.size
		book.sections = append(book.sections, section)
	}
	if book.total == 0 {
		return nil, fmt.Errorf("%s: no linear content", file)
	}
	return book, nil
}

// locate returns a CFI at about fraction of the way through the book, and
// the fraction the reader will report there.
func (b *epubBook) locate(fraction float64) (string, float64, error) {
	target := int64(fraction * float64(b.total))
	var before int64
	last := 0
	for i, s := range b.sections {
		if s.size > 0 {
			last = i
		}
	}
	for i, s := range b.sections {
		if s.size == 0 {
			continue
		}
		if before+s.size <= target && i != last {
			before += s.size
			continue
		}
		within := min(target-before, s.size)
		docPath, offset, err := elementAt(s.content, within)
		if err != nil {
			return "", 0, err
		}
		at := float64(before+offset) / float64(b.total)
		if fraction >= 1 {
			at = 1
		}
		return "epubcfi(" + s.step + "!" + docPath + ")", at, nil
	}
	return "", 0, fmt.Errorf("no section at %.2f", fraction)
}

// elementAt returns the CFI path of the last element inside <body> that
// starts at or before offset, and that element's offset.
func elementAt(doc []byte, offset int64) (string, int64, error) {
	dec := xml.NewDecoder(bytes.NewReader(doc))
	dec.Strict = false
	dec.AutoClose = xml.HTMLAutoClose
	dec.Entity = xml.HTMLEntity

	// counts[d] is how many element children have opened at depth d.
	counts := []int{0}
	var stack []int
	inBody := false
	best, bestOffset := "", int64(0)
	for {
		start := dec.InputOffset()
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", 0, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			counts[len(counts)-1]++
			stack = append(stack, counts[len(counts)-1])
			counts = append(counts, 0)
			if len(stack) == 2 && t.Name.Local == "body" {
				inBody = true
			}
			if inBody && (best == "" || start <= offset) {
				best, bestOffset = cfiPath(stack[1:]), start
			}
		case xml.EndElement:
			if len(stack) == 2 {
				inBody = false
			}
			stack = stack[:len(stack)-1]
			counts = counts[:len(counts)-1]
		}
	}
	if best == "" {
		return "", 0, fmt.Errorf("no <body>")
	}
	return best, bestOffset, nil
}

// cfiPath turns 1-based element positions below <html> into CFI steps.
func cfiPath(positions []int) string {
	var b strings.Builder
	for _, p := range positions {
		fmt.Fprintf(&b, "/%d", p*2)
	}
	return b.String()
}

type node struct {
	name     string
	attrs    []xml.Attr
	children []*node
}

func (n *node) attr(name string) string {
	for _, a := range n.attrs {
		if a.Name.Local == name {
			return a.Value
		}
	}
	return ""
}

// parseTree reads an XML document's elements, without text.
func parseTree(doc []byte) (*node, error) {
	dec := xml.NewDecoder(bytes.NewReader(doc))
	var stack []*node
	var root *node
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return root, nil
		}
		if err != nil {
			return nil, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			n := &node{name: t.Name.Local, attrs: t.Attr}
			if len(stack) == 0 {
				root = n
			} else {
				parent := stack[len(stack)-1]
				parent.children = append(parent.children, n)
			}
			stack = append(stack, n)
		case xml.EndElement:
			stack = stack[:len(stack)-1]
		}
	}
}

func readEntry(entries map[string]*zip.File, name string) ([]byte, error) {
	f, ok := entries[name]
	if !ok {
		return nil, fmt.Errorf("missing %s", name)
	}
	r, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}

func readXML(entries map[string]*zip.File, name string, v any) error {
	content, err := readEntry(entries, name)
	if err != nil {
		return err
	}
	return xml.Unmarshal(content, v)
}
