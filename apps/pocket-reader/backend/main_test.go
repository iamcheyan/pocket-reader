package readerd

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func testServer(t *testing.T) *Server {
	t.Helper()
	db, e := sql.Open("sqlite", ":memory:")
	if e != nil {
		t.Fatal(e)
	}
	db.SetMaxOpenConns(1)
	s := &Server{db: db, root: t.TempDir()}
	if e = s.migrate(); e != nil {
		t.Fatal(e)
	}
	return s
}
func TestMigrationAndMockSeed(t *testing.T) {
	s := testServer(t)
	if e := s.seedMock(); e != nil {
		t.Fatal(e)
	}
	var n int
	if e := s.db.QueryRow("SELECT count(*) FROM books").Scan(&n); e != nil || n != 100 {
		t.Fatalf("mock count=%d err=%v", n, e)
	}
	w := httptest.NewRecorder()
	s.routes().ServeHTTP(w, httptest.NewRequest("GET", "/api/books?limit=2", nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), "\"total\":100") {
		t.Fatalf("books response %d %s", w.Code, w.Body.String())
	}
}
func TestGeneratedTextAndMarkdownProviders(t *testing.T) {
	for _, tc := range []struct {
		ext  string
		body string
		want string
	}{{".txt", "TXT generated search phrase", "TXT generated"}, {".md", "# Generated heading\n\nMarkdown search phrase", "<h2>Generated heading</h2>"}} {
		t.Run(tc.ext, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "sample"+tc.ext)
			if e := os.WriteFile(path, []byte(tc.body), 0600); e != nil {
				t.Fatal(e)
			}
			p := TextProvider{Ext: tc.ext}
			d, e := p.Open(context.Background(), path)
			if e != nil {
				t.Fatal(e)
			}
			meta, e := p.Metadata(context.Background(), d)
			if e != nil || meta.Title != "sample" {
				t.Fatalf("metadata=%+v err=%v", meta, e)
			}
			toc, e := p.TOC(context.Background(), d)
			if e != nil || len(toc) == 0 {
				t.Fatalf("toc=%v err=%v", toc, e)
			}
			hits, e := p.Search(context.Background(), d, "phrase")
			if e != nil || len(hits) == 0 {
				t.Fatalf("hits=%v err=%v", hits, e)
			}
			chunk, e := p.Content(context.Background(), d, Location{Loc: "0"})
			if e != nil || !strings.Contains(chunk.Content, tc.want) || chunk.NextLoc != nil {
				t.Fatalf("content=%+v err=%v", chunk, e)
			}
		})
	}
}
func TestEPUBGeneratedFixture(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sample.epub")
	f, e := os.Create(path)
	if e != nil {
		t.Fatal(e)
	}
	z := zip.NewWriter(f)
	members := map[string]string{"META-INF/container.xml": `<container><rootfiles><rootfile full-path="OEBPS/content.opf"/></rootfiles></container>`, "OEBPS/content.opf": `<package><metadata><title>Package Title</title><creator>Package Author</creator><language>en</language><publisher>Reader Press</publisher></metadata><manifest><item id="nav" href="nav.xhtml" media-type="application/xhtml+xml"/><item id="second" href="second.xhtml" media-type="application/xhtml+xml"/><item id="first" href="first.xhtml" media-type="application/xhtml+xml"/></manifest><spine><itemref idref="first"/><itemref idref="second"/></spine></package>`, "OEBPS/nav.xhtml": `<html><body><nav><ol><li><a href="first.xhtml">Opening Chapter</a></li><li><a href="second.xhtml">Closing Chapter</a></li></ol></nav></body></html>`, "OEBPS/second.xhtml": `<html><head><title>Second</title></head><body>Last chapter</body></html>`, "OEBPS/first.xhtml": `<html><head><title>First</title></head><body>Generated fixture phrase</body></html>`}
	for name, data := range members {
		w, e := z.Create(name)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = w.Write([]byte(data)); e != nil {
			t.Fatal(e)
		}
	}
	if e = z.Close(); e != nil {
		t.Fatal(e)
	}
	if e = f.Close(); e != nil {
		t.Fatal(e)
	}
	p := EPUBProvider{}
	d, e := p.Open(context.Background(), path)
	if e != nil {
		t.Fatal(e)
	}
	meta, _ := p.Metadata(context.Background(), d)
	toc, _ := p.TOC(context.Background(), d)
	hits, _ := p.Search(context.Background(), d, "fixture phrase")
	first, _ := p.Content(context.Background(), d, Location{Loc: "0"})
	if meta.Title != "Package Title" || meta.Author != "Package Author" || meta.Language != "en" || meta.Publisher != "Reader Press" || len(toc) != 2 || toc[0].Title != "Opening Chapter" || toc[1].Loc != "1" || len(hits) != 1 || !strings.Contains(first.Content, "Generated fixture phrase") {
		t.Fatalf("meta=%+v toc=%v hits=%v first=%+v", meta, toc, hits, first)
	}
}
func TestPDFProviderRejectsInvalidDocument(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.pdf")
	os.WriteFile(path, []byte("not a pdf"), 0600)
	_, e := (PDFProvider{}).Open(context.Background(), path)
	if e == nil {
		t.Fatal("expected generated invalid PDF to fail parsing")
	}
}
func TestMockReadingAndHealth(t *testing.T) {
	s := testServer(t)
	if e := s.seedMock(); e != nil {
		t.Fatal(e)
	}
	for _, path := range []string{"/api/books/1/toc", "/api/books/1/content?loc=0", "/api/books/1/search?q=sample"} {
		w := httptest.NewRecorder()
		s.routes().ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != 200 || w.Body.Len() == 0 {
			t.Fatalf("%s status=%d body=%s", path, w.Code, w.Body.String())
		}
	}
	w := httptest.NewRecorder()
	s.routes().ServeHTTP(w, httptest.NewRequest("GET", "/api/health", nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"mock":true`) {
		t.Fatalf("health=%s", w.Body.String())
	}
}
func TestCollectionMembershipAndProgressInList(t *testing.T) {
	s := testServer(t)
	if e := s.seedMock(); e != nil {
		t.Fatal(e)
	}
	create := httptest.NewRecorder()
	s.routes().ServeHTTP(create, httptest.NewRequest("POST", "/api/collections", bytes.NewBufferString(`{"name":"Shelf","description":"Test","book_ids":[1,2]}`)))
	if create.Code != 200 {
		t.Fatalf("create=%d %s", create.Code, create.Body.String())
	}
	list := httptest.NewRecorder()
	s.routes().ServeHTTP(list, httptest.NewRequest("GET", "/api/collections", nil))
	if !strings.Contains(list.Body.String(), `"book_count":2`) {
		t.Fatalf("collections=%s", list.Body.String())
	}
	put := httptest.NewRecorder()
	s.routes().ServeHTTP(put, httptest.NewRequest("PUT", "/api/books/1/progress", bytes.NewBufferString(`{"chapter":"Chapter 1","loc":"1","percent":0.25}`)))
	if put.Code != 200 {
		t.Fatalf("progress=%d %s", put.Code, put.Body.String())
	}
	books := httptest.NewRecorder()
	s.routes().ServeHTTP(books, httptest.NewRequest("GET", "/api/books?limit=1", nil))
	if !strings.Contains(books.Body.String(), `"progress":0.25`) {
		t.Fatalf("books=%s", books.Body.String())
	}
}
func TestProgressAndBookmarkCRUD(t *testing.T) {
	s := testServer(t)
	s.seedMock()
	put := httptest.NewRecorder()
	s.routes().ServeHTTP(put, httptest.NewRequest("PUT", "/api/books/1/progress", bytes.NewBufferString(`{"chapter":"1","loc":"2","percent":0.25}`)))
	if put.Code != 200 {
		t.Fatalf("progress status=%d: %s", put.Code, put.Body.String())
	}
	post := httptest.NewRecorder()
	s.routes().ServeHTTP(post, httptest.NewRequest("POST", "/api/bookmarks", bytes.NewBufferString(`{"book_id":1,"loc":"2","text":"mark"}`)))
	if post.Code != 200 {
		t.Fatalf("bookmark status=%d", post.Code)
	}
}
