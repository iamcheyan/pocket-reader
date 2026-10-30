package readerd

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScanIndexesAllFourProviders(t *testing.T) {
	root := t.TempDir()
	writeText(t, filepath.Join(root, "sub", "plain.txt"), strings.Repeat("Generated text chapter. ", 400))
	writeText(t, filepath.Join(root, "notes.md"), "# Generated heading\n\nMarkdown body phrase")
	writeEPUB(t, filepath.Join(root, "nested", "sample.epub"))
	writeText(t, filepath.Join(root, "doc.pdf"), minimalPDFText())

	s := testServer(t)
	s.root = root
	s.doScan(context.Background())

	rows, err := s.db.Query("SELECT format,size,path FROM books ORDER BY format")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	found := map[string]bool{}
	for rows.Next() {
		var format, path string
		var size int64
		if err := rows.Scan(&format, &size, &path); err != nil {
			t.Fatal(err)
		}
		found[format] = true
		if size <= 0 || path == "" {
			t.Fatalf("bad metadata for %s: size=%d path=%q", format, size, path)
		}
	}
	for _, format := range []string{"EPUB", "PDF", "TXT", "MD"} {
		if !found[format] {
			t.Fatalf("missing scanned format %s (found %v)", format, found)
		}
	}
	if s.scan.Added != 4 || s.scan.Running {
		t.Fatalf("scan status=%+v", s.scan)
	}
}

func TestEPUBProviderEndpoints(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "book.epub")
	writeEPUB(t, path)
	s := testServer(t)
	s.root = root
	s.doScan(context.Background())
	var id int64
	if err := s.db.QueryRow("SELECT id FROM books WHERE format='EPUB'").Scan(&id); err != nil {
		t.Fatal(err)
	}
	toc := callJSON(t, s, fmt.Sprintf("/api/books/%d/toc", id))
	if !strings.Contains(toc, "Opening Chapter") {
		t.Fatalf("toc=%s", toc)
	}
	content := callJSON(t, s, fmt.Sprintf("/api/books/%d/content?loc=0", id))
	if !strings.Contains(content, "Generated fixture phrase") || !strings.Contains(content, `"next_loc":"1"`) {
		t.Fatalf("content=%s", content)
	}
	hits := callJSON(t, s, fmt.Sprintf("/api/books/%d/search?q=fixture", id))
	if !strings.Contains(hits, "Generated fixture phrase") {
		t.Fatalf("hits=%s", hits)
	}
}

func TestPDFProviderExtractsGeneratedFixture(t *testing.T) {
	root := t.TempDir()
	writeText(t, filepath.Join(root, "doc.pdf"), minimalPDFText())
	s := testServer(t)
	s.root = root
	s.doScan(context.Background())
	var id int64
	if err := s.db.QueryRow("SELECT id FROM books WHERE format='PDF'").Scan(&id); err != nil {
		t.Fatal(err)
	}
	content := callJSON(t, s, fmt.Sprintf("/api/books/%d/content?loc=0", id))
	if !strings.Contains(content, "Generated PDF phrase") {
		t.Fatalf("pdf content=%s", content)
	}
	toc := callJSON(t, s, fmt.Sprintf("/api/books/%d/toc", id))
	if !strings.Contains(toc, "Page 1") {
		t.Fatalf("pdf toc=%s", toc)
	}
}

func TestProgressResumesAcrossServerRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "resume.sqlite")
	put := func(server *Server) {
		w := httptest.NewRecorder()
		server.routes().ServeHTTP(w, httptest.NewRequest("PUT", "/api/books/1/progress", bytes.NewBufferString(`{"chapter":"Chapter 2","loc":"3","percent":37.5}`)))
		if w.Code != 200 {
			t.Fatalf("put=%d %s", w.Code, w.Body.String())
		}
	}
	first := openServer(t, path)
	if err := first.seedMock(); err != nil {
		t.Fatal(err)
	}
	put(first)
	first.db.Close()

	second := openServer(t, path)
	body := callJSON(t, second, "/api/books/1/progress")
	if !strings.Contains(body, `"loc":"3"`) || !strings.Contains(body, `"percent":37.5`) {
		t.Fatalf("resume progress=%s", body)
	}
	listed := callJSON(t, second, "/api/books?limit=1")
	if !strings.Contains(listed, `"progress":37.5`) {
		t.Fatalf("listed progress=%s", listed)
	}
}

func TestNotesExportRoundTrip(t *testing.T) {
	s := testServer(t)
	if err := s.seedMock(); err != nil {
		t.Fatal(err)
	}
	post := httptest.NewRecorder()
	s.routes().ServeHTTP(post, httptest.NewRequest("POST", "/api/notes", bytes.NewBufferString(`{"book_id":1,"loc":"2","text":"exported note"}`)))
	if post.Code != 200 {
		t.Fatalf("post=%d", post.Code)
	}
	md := httptest.NewRecorder()
	s.routes().ServeHTTP(md, httptest.NewRequest("GET", "/api/notes/export?format=md", nil))
	if !strings.Contains(md.Body.String(), "exported note") || !strings.HasPrefix(md.Header().Get("Content-Type"), "text/markdown") {
		t.Fatalf("md=%s", md.Body.String())
	}
	js := httptest.NewRecorder()
	s.routes().ServeHTTP(js, httptest.NewRequest("GET", "/api/notes/export?format=json", nil))
	var payload struct {
		Items []struct {
			Text string `json:"text"`
		} `json:"items"`
	}
	if err := json.Unmarshal(js.Body.Bytes(), &payload); err != nil || len(payload.Items) != 1 || payload.Items[0].Text != "exported note" {
		t.Fatalf("json=%s err=%v", js.Body.String(), err)
	}
}

func TestOPDSAtomParsing(t *testing.T) {
	feed := `<?xml version="1.0"?><feed xmlns="http://www.w3.org/2005/Atom"><entry><title>Generated Catalog Book</title><id>urn:book:1</id><link rel="http://opds-spec.org/acquisition" type="application/epub+zip" href="http://127.0.0.1:9/book.epub"/></entry></feed>`
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/atom+xml")
		_, _ = w.Write([]byte(feed))
	}))
	defer upstream.Close()
	s := testServer(t)
	body := callJSON(t, s, "/api/opds/catalog?url="+upstream.URL)
	if !strings.Contains(body, "Generated Catalog Book") || !strings.Contains(body, "book.epub") {
		t.Fatalf("opds=%s", body)
	}
	bad := httptest.NewRecorder()
	s.routes().ServeHTTP(bad, httptest.NewRequest("GET", "/api/opds/catalog?url=file:///etc/passwd", nil))
	if bad.Code != 400 {
		t.Fatalf("bad url status=%d", bad.Code)
	}
}

func TestHighlightColorsAndDeletion(t *testing.T) {
	s := testServer(t)
	if err := s.seedMock(); err != nil {
		t.Fatal(err)
	}
	post := httptest.NewRecorder()
	s.routes().ServeHTTP(post, httptest.NewRequest("POST", "/api/highlights", bytes.NewBufferString(`{"book_id":1,"loc":"2","text":"colored","color":"blue"}`)))
	if post.Code != 200 || !strings.Contains(post.Body.String(), `"color":"blue"`) {
		t.Fatalf("post=%d %s", post.Code, post.Body.String())
	}
	list := callJSON(t, s, "/api/highlights?book_id=1")
	if !strings.Contains(list, "colored") {
		t.Fatalf("list=%s", list)
	}
	var id int64
	_ = s.db.QueryRow("SELECT id FROM highlights WHERE text='colored'").Scan(&id)
	del := httptest.NewRecorder()
	s.routes().ServeHTTP(del, httptest.NewRequest("DELETE", fmt.Sprintf("/api/highlights/%d", id), nil))
	if del.Code != 204 {
		t.Fatalf("delete=%d", del.Code)
	}
	if remaining := callJSON(t, s, "/api/highlights?book_id=1"); strings.Contains(remaining, "colored") {
		t.Fatalf("highlight not deleted: %s", remaining)
	}
}

func openServer(t *testing.T, path string) *Server {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	s := &Server{db: db, root: t.TempDir()}
	if err := s.migrate(); err != nil {
		t.Fatal(err)
	}
	return s
}

func callJSON(t *testing.T, s *Server, path string) string {
	t.Helper()
	w := httptest.NewRecorder()
	s.routes().ServeHTTP(w, httptest.NewRequest("GET", path, nil))
	if w.Code != 200 {
		t.Fatalf("%s status=%d body=%s", path, w.Code, w.Body.String())
	}
	return w.Body.String()
}

func writeText(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func writeEPUB(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	z := zip.NewWriter(f)
	members := map[string]string{
		"META-INF/container.xml": `<container><rootfiles><rootfile full-path="OEBPS/content.opf"/></rootfiles></container>`,
		"OEBPS/content.opf":      `<package><metadata><title>Package Title</title><creator>Package Author</creator><language>en</language><publisher>Reader Press</publisher></metadata><manifest><item id="nav" href="nav.xhtml" media-type="application/xhtml+xml"/><item id="first" href="first.xhtml" media-type="application/xhtml+xml"/><item id="second" href="second.xhtml" media-type="application/xhtml+xml"/></manifest><spine><itemref idref="first"/><itemref idref="second"/></spine></package>`,
		"OEBPS/nav.xhtml":        `<html><body><nav><ol><li><a href="first.xhtml">Opening Chapter</a></li><li><a href="second.xhtml">Closing Chapter</a></li></ol></nav></body></html>`,
		"OEBPS/first.xhtml":      `<html><head><title>First</title></head><body><h1>Opening</h1><p>Generated fixture phrase</p></body></html>`,
		"OEBPS/second.xhtml":     `<html><head><title>Second</title></head><body><p>Closing text</p></body></html>`,
	}
	for name, data := range members {
		w, err := z.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = w.Write([]byte(data)); err != nil {
			t.Fatal(err)
		}
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

func minimalPDFText() string {
	text := "Generated PDF phrase"
	stream := fmt.Sprintf("BT /F1 18 Tf 72 720 Td (%s) Tj ET", text)
	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 4 0 R >> >> /Contents 5 0 R >>",
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(stream), stream),
	}
	var b strings.Builder
	b.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objects)+1)
	for i, object := range objects {
		offsets[i+1] = b.Len()
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", i+1, object)
	}
	xref := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)
	for _, offset := range offsets[1:] {
		fmt.Fprintf(&b, "%010d 00000 n \n", offset)
	}
	fmt.Fprintf(&b, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xref)
	return b.String()
}
