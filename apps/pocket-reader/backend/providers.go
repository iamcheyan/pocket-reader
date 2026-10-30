package readerd

import (
	"archive/zip"
	"context"
	"encoding/xml"
	"fmt"
	pdf "github.com/ledongthuc/pdf"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type TextProvider struct{ Ext string }

func (p TextProvider) Name() string              { return strings.TrimPrefix(p.Ext, ".") }
func (p TextProvider) Supports(path string) bool { return strings.EqualFold(filepath.Ext(path), p.Ext) }
func (p TextProvider) Open(ctx context.Context, path string) (Document, error) {
	b, e := os.ReadFile(path)
	if e != nil {
		return nil, e
	}
	text := string(b)
	if strings.EqualFold(filepath.Ext(path), ".md") || strings.EqualFold(filepath.Ext(path), ".markdown") {
		text = renderMarkdown(text)
	}
	return &TextDoc{title: strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)), format: p.Name(), parts: paginate(text, 5000)}, nil
}
func (p TextProvider) Metadata(_ context.Context, d Document) (Metadata, error) {
	x, ok := d.(*TextDoc)
	if !ok {
		return Metadata{}, fmt.Errorf("invalid document")
	}
	return Metadata{Title: x.title}, nil
}
func (p TextProvider) TOC(_ context.Context, d Document) ([]TOCItem, error) {
	x := d.(*TextDoc)
	o := make([]TOCItem, 0, len(x.parts))
	for i := range x.parts {
		o = append(o, TOCItem{Loc: strconv.Itoa(i), Title: fmt.Sprintf("Page %d", i+1)})
	}
	return o, nil
}
func (p TextProvider) Search(_ context.Context, d Document, q string) ([]SearchHit, error) {
	x := d.(*TextDoc)
	return searchParts(x.parts, q), nil
}
func (p TextProvider) Content(_ context.Context, d Document, l Location) (ContentChunk, error) {
	x := d.(*TextDoc)
	return chunk(x.title, x.parts, l.Loc, "text/plain")
}
func paginate(s string, n int) []string {
	r := []string{}
	for len(s) > n {
		cut := strings.LastIndex(s[:n], "\n")
		if cut < n/2 {
			cut = n
		}
		r = append(r, s[:cut])
		s = strings.TrimLeft(s[cut:], "\n")
	}
	if s != "" || len(r) == 0 {
		r = append(r, s)
	}
	return r
}
func renderMarkdown(s string) string {
	lines := strings.Split(s, "\n")
	var b strings.Builder
	for _, line := range lines {
		v := strings.TrimSpace(line)
		if strings.HasPrefix(v, "#") {
			v = strings.TrimLeft(v, "# ")
			b.WriteString("<h2>")
			b.WriteString(escapeHTML(v))
			b.WriteString("</h2>\n")
		} else if v != "" {
			b.WriteString("<p>")
			b.WriteString(escapeHTML(v))
			b.WriteString("</p>\n")
		}
	}
	return b.String()
}
func escapeHTML(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\"", "&quot;").Replace(s)
}
func searchParts(parts []string, q string) []SearchHit {
	out := []SearchHit{}
	if q == "" {
		return out
	}
	for i, p := range parts {
		idx := strings.Index(strings.ToLower(p), strings.ToLower(q))
		if idx >= 0 {
			start := idx - 70
			if start < 0 {
				start = 0
			}
			end := idx + len(q) + 120
			if end > len(p) {
				end = len(p)
			}
			out = append(out, SearchHit{Loc: strconv.Itoa(i), Text: q, Excerpt: p[start:end]})
		}
	}
	return out
}
func navigableChunk(title string, parts []string, i int, ctype string) (ContentChunk, error) {
	if i < 0 || i >= len(parts) {
		return ContentChunk{}, fmt.Errorf("location out of range")
	}
	c := ContentChunk{Loc: strconv.Itoa(i), Title: title, Content: parts[i], ContentType: ctype, HasPrevious: i > 0, HasNext: i+1 < len(parts)}
	if c.HasPrevious {
		v := strconv.Itoa(i - 1)
		c.PreviousLoc = &v
	}
	if c.HasNext {
		v := strconv.Itoa(i + 1)
		c.NextLoc = &v
	}
	return c, nil
}

func chunk(title string, parts []string, loc, ctype string) (ContentChunk, error) {
	i := 0
	if loc != "" {
		n, e := strconv.Atoi(loc)
		if e != nil {
			return ContentChunk{}, fmt.Errorf("invalid location")
		}
		i = n
	}
	return navigableChunk(title, parts, i, ctype)
}

type EPUBProvider struct{}

func (EPUBProvider) Name() string           { return "epub" }
func (EPUBProvider) Supports(p string) bool { return strings.EqualFold(filepath.Ext(p), ".epub") }

type epubPackage struct {
	Metadata struct {
		Title     string `xml:"title"`
		Creator   string `xml:"creator"`
		Language  string `xml:"language"`
		Publisher string `xml:"publisher"`
	} `xml:"metadata"`
	Manifest struct {
		Items []struct {
			ID    string `xml:"id,attr"`
			Href  string `xml:"href,attr"`
			Media string `xml:"media-type,attr"`
		} `xml:"item"`
	} `xml:"manifest"`
	Spine struct {
		Refs []struct {
			ID string `xml:"idref,attr"`
		} `xml:"itemref"`
	} `xml:"spine"`
}
type epubContainer struct {
	RootFiles struct {
		Root struct {
			OPF string `xml:"full-path,attr"`
		} `xml:"rootfile"`
	} `xml:"rootfiles"`
}
type epubNav struct {
	Body struct {
		Nav struct {
			Ol struct {
				Items []struct {
					Anchors []struct {
						Href  string `xml:"href,attr"`
						Label string `xml:",chardata"`
					} `xml:"a"`
				} `xml:"li"`
			} `xml:"ol"`
		} `xml:"nav"`
	} `xml:"body"`
}

func (EPUBProvider) Open(ctx context.Context, filename string) (Document, error) {
	z, e := zip.OpenReader(filename)
	if e != nil {
		return nil, e
	}
	defer z.Close()
	files := map[string]*zip.File{}
	for _, f := range z.File {
		files[f.Name] = f
	}
	read := func(name string) ([]byte, error) {
		f := files[path.Clean(name)]
		if f == nil {
			return nil, fmt.Errorf("EPUB member %q missing", name)
		}
		r, e := f.Open()
		if e != nil {
			return nil, e
		}
		defer r.Close()
		return io.ReadAll(io.LimitReader(r, 8<<20))
	}
	container, e := read("META-INF/container.xml")
	if e != nil {
		return nil, e
	}
	var c epubContainer
	if e = xml.Unmarshal(container, &c); e != nil || c.RootFiles.Root.OPF == "" {
		return nil, fmt.Errorf("invalid EPUB container")
	}
	opfBytes, e := read(c.RootFiles.Root.OPF)
	if e != nil {
		return nil, e
	}
	var pkg epubPackage
	if e = xml.Unmarshal(opfBytes, &pkg); e != nil {
		return nil, e
	}
	base := path.Dir(c.RootFiles.Root.OPF)
	manifest := map[string]string{}
	for _, item := range pkg.Manifest.Items {
		manifest[item.ID] = path.Clean(path.Join(base, item.Href))
	}
	var parts, titles []string
	for _, ref := range pkg.Spine.Refs {
		name := manifest[ref.ID]
		if name == "" {
			return nil, fmt.Errorf("EPUB spine references missing item %q", ref.ID)
		}
		b, e := read(name)
		if e != nil {
			return nil, e
		}
		title := strings.TrimSuffix(filepath.Base(name), filepath.Ext(name))
		heading := regexp.MustCompile(`(?is)<h[1-6][^>]*>(.*?)</h[1-6]>`)
		titleTag := regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
		if m := heading.FindStringSubmatch(string(b)); len(m) > 1 && strings.TrimSpace(stripMarkup(m[1])) != "" {
			title = stripMarkup(m[1])
		} else if m := titleTag.FindStringSubmatch(string(b)); len(m) > 1 {
			title = stripMarkup(m[1])
		}
		parts = append(parts, string(b))
		titles = append(titles, title)
	}
	if len(parts) == 0 {
		return nil, fmt.Errorf("EPUB spine contains no chapters")
	}
	meta := Metadata{Title: pkg.Metadata.Title, Author: pkg.Metadata.Creator, Language: pkg.Metadata.Language, Publisher: pkg.Metadata.Publisher}
	if meta.Title == "" {
		meta.Title = strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
	}
	nav := []TOCItem{}
	for _, item := range pkg.Manifest.Items {
		if item.Media != "application/xhtml+xml" || !strings.Contains(strings.ToLower(item.ID), "nav") {
			continue
		}
		b, e := read(path.Join(base, item.Href))
		if e != nil {
			continue
		}
		var n epubNav
		if xml.Unmarshal(b, &n) != nil {
			continue
		}
		for _, x := range n.Body.Nav.Ol.Items {
			if len(x.Anchors) == 0 {
				continue
			}
			a := x.Anchors[0]
			href := strings.Split(a.Href, "#")[0]
			index := -1
			for i, ref := range pkg.Spine.Refs {
				if manifest[ref.ID] == path.Clean(path.Join(base, href)) {
					index = i
					break
				}
			}
			if index >= 0 {
				nav = append(nav, TOCItem{Loc: strconv.Itoa(index), Title: strings.TrimSpace(a.Label)})
			}
		}
		break
	}
	return &EPUBDoc{path: filename, parts: parts, titles: titles, metadata: meta, nav: nav}, nil
}
func (EPUBProvider) Metadata(_ context.Context, d Document) (Metadata, error) {
	return d.(*EPUBDoc).metadata, nil
}
func (EPUBProvider) TOC(_ context.Context, d Document) ([]TOCItem, error) {
	x := d.(*EPUBDoc)
	if len(x.nav) > 0 {
		return x.nav, nil
	}
	out := make([]TOCItem, len(x.titles))
	for i, t := range x.titles {
		out[i] = TOCItem{Loc: strconv.Itoa(i), Title: t}
	}
	return out, nil
}
func (EPUBProvider) Search(_ context.Context, d Document, q string) ([]SearchHit, error) {
	x := d.(*EPUBDoc)
	parts := make([]string, len(x.parts))
	for i, p := range x.parts {
		parts[i] = stripMarkup(p)
	}
	return searchParts(parts, q), nil
}
func (EPUBProvider) Content(_ context.Context, d Document, l Location) (ContentChunk, error) {
	x := d.(*EPUBDoc)
	i := 0
	if l.Loc != "" {
		n, e := strconv.Atoi(l.Loc)
		if e != nil {
			return ContentChunk{}, e
		}
		i = n
	}
	if i < 0 || i >= len(x.parts) {
		return ContentChunk{}, fmt.Errorf("location out of range")
	}
	c := ContentChunk{Loc: strconv.Itoa(i), Title: x.titles[i], Content: x.parts[i], ContentType: "application/xhtml+xml", HasPrevious: i > 0, HasNext: i+1 < len(x.parts)}
	if c.HasPrevious {
		v := strconv.Itoa(i - 1)
		c.PreviousLoc = &v
	}
	if c.HasNext {
		v := strconv.Itoa(i + 1)
		c.NextLoc = &v
	}
	return c, nil
}
func stripMarkup(s string) string {
	re := regexp.MustCompile(`(?s)<[^>]*>`)
	return strings.TrimSpace(re.ReplaceAllString(s, " "))
}

type PDFProvider struct{}

func (PDFProvider) Name() string           { return "pdf" }
func (PDFProvider) Supports(p string) bool { return strings.EqualFold(filepath.Ext(p), ".pdf") }
func (PDFProvider) Open(ctx context.Context, path string) (Document, error) {
	parts, e := extractPDF(path)
	if e != nil {
		return nil, e
	}
	return &PDFDoc{path: path, parts: parts}, nil
}
func extractPDF(filename string) ([]string, error) {
	if exe, e := exec.LookPath("mutool"); e == nil {
		cmd := exec.Command(exe, "draw", "-F", "txt", "-o", "-", filename)
		out, e := cmd.Output()
		if e == nil {
			return splitPDFPages(string(out)), nil
		}
	}
	f, r, e := pdf.Open(filename)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	parts := make([]string, r.NumPage())
	for i := 1; i <= r.NumPage(); i++ {
		p := r.Page(i)
		txt, e := p.GetPlainText(nil)
		if e != nil {
			return nil, e
		}
		parts[i-1] = txt
	}
	return parts, nil
}
func splitPDFPages(text string) []string {
	pages := strings.Split(text, "\f")
	for len(pages) > 0 && pages[len(pages)-1] == "" {
		pages = pages[:len(pages)-1]
	}
	if len(pages) == 0 {
		return []string{""}
	}
	return pages
}
func (PDFProvider) Metadata(_ context.Context, d Document) (Metadata, error) {
	x := d.(*PDFDoc)
	return Metadata{Title: strings.TrimSuffix(filepath.Base(x.path), filepath.Ext(x.path))}, nil
}
func (PDFProvider) TOC(_ context.Context, d Document) ([]TOCItem, error) {
	x := d.(*PDFDoc)
	out := []TOCItem{}
	for i := range x.parts {
		out = append(out, TOCItem{Loc: strconv.Itoa(i), Title: fmt.Sprintf("Page %d", i+1)})
	}
	return out, nil
}
func (PDFProvider) Search(_ context.Context, d Document, q string) ([]SearchHit, error) {
	return searchParts(d.(*PDFDoc).parts, q), nil
}
func (PDFProvider) Content(_ context.Context, d Document, l Location) (ContentChunk, error) {
	i := 0
	if l.Loc != "" {
		n, e := strconv.Atoi(l.Loc)
		if e != nil {
			return ContentChunk{}, e
		}
		i = n
	}
	return navigableChunk(filepath.Base(d.(*PDFDoc).path), d.(*PDFDoc).parts, i, "text/plain")
}
