package readerd

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

const version = "0.1.0"

type Server struct {
	db     *sql.DB
	root   string
	mu     sync.Mutex
	scan   ScanStatus
	cancel context.CancelFunc
}
type ScanStatus struct {
	Running    bool      `json:"running"`
	Scanned    int       `json:"scanned"`
	Added      int       `json:"added"`
	Updated    int       `json:"updated"`
	Errors     []string  `json:"errors"`
	StartedAt  time.Time `json:"started_at,omitempty"`
	FinishedAt time.Time `json:"finished_at,omitempty"`
}
type Book struct {
	ID       int64    `json:"id"`
	Title    string   `json:"title"`
	Author   string   `json:"author"`
	Series   string   `json:"series"`
	Language string   `json:"language"`
	Format   string   `json:"format"`
	Path     string   `json:"path,omitempty"`
	Size     int64    `json:"size"`
	Modified int64    `json:"modified"`
	Cover    string   `json:"cover,omitempty"`
	Tags     []string `json:"tags"`
	Favorite bool     `json:"favorite"`
	Mock     bool     `json:"mock"`
	Progress float64  `json:"progress"`
}
type Provider interface {
	Name() string
	Supports(string) bool
	Open(context.Context, string) (Document, error)
	Metadata(context.Context, Document) (Metadata, error)
	TOC(context.Context, Document) ([]TOCItem, error)
	Search(context.Context, Document, string) ([]SearchHit, error)
	Content(context.Context, Document, Location) (ContentChunk, error)
}
type Document interface{}
type Metadata struct {
	Title     string
	Author    string
	Language  string
	Publisher string
}
type Location struct {
	Loc string `json:"loc"`
}
type TOCItem struct {
	Loc   string `json:"loc"`
	Title string `json:"title"`
}
type SearchHit struct {
	Loc     string `json:"loc"`
	Text    string `json:"text"`
	Excerpt string `json:"excerpt"`
}
type ContentChunk struct {
	BookID      string  `json:"book_id,omitempty"`
	Loc         string  `json:"loc"`
	Title       string  `json:"title"`
	Content     string  `json:"content"`
	ContentType string  `json:"content_type"`
	HasNext     bool    `json:"has_next"`
	HasPrevious bool    `json:"has_previous"`
	NextLoc     *string `json:"next_loc,omitempty"`
	PreviousLoc *string `json:"previous_loc,omitempty"`
}
type TextDoc struct {
	title, format string
	parts         []string
}
type EPUBDoc struct {
	path     string
	parts    []string
	titles   []string
	metadata Metadata
	nav      []TOCItem
}
type PDFDoc struct {
	path  string
	parts []string
}
type MockDoc struct {
	title, format string
	parts         []string
}

func (m MockDoc) count() int { return len(m.parts) }

type mockProvider struct{ book Book }

func (p mockProvider) Name() string                                   { return strings.ToLower(p.book.Format) }
func (p mockProvider) Supports(string) bool                           { return true }
func (p mockProvider) Open(context.Context, string) (Document, error) { return p.doc(), nil }
func (p mockProvider) doc() MockDoc {
	parts := make([]string, 8)
	for i := range parts {
		parts[i] = fmt.Sprintf("%s — Chapter %d\n\n%s is a sample reading chapter for %s. This generated demo content lets you try page navigation, search, and progress tracking.\n\n", p.book.Title, i+1, p.book.Author, p.book.Title)
	}
	return MockDoc{title: p.book.Title, format: p.book.Format, parts: parts}
}
func (p mockProvider) Metadata(_ context.Context, d Document) (Metadata, error) {
	return Metadata{Title: p.book.Title, Author: p.book.Author, Language: p.book.Language}, nil
}
func (p mockProvider) TOC(_ context.Context, d Document) ([]TOCItem, error) {
	x := d.(MockDoc)
	out := make([]TOCItem, len(x.parts))
	for i := range x.parts {
		out[i] = TOCItem{Loc: strconv.Itoa(i), Title: fmt.Sprintf("Chapter %d", i+1)}
	}
	return out, nil
}
func (p mockProvider) Search(_ context.Context, d Document, q string) ([]SearchHit, error) {
	x := d.(MockDoc)
	return searchParts(x.parts, q), nil
}
func (p mockProvider) Content(_ context.Context, d Document, l Location) (ContentChunk, error) {
	x := d.(MockDoc)
	return chunk(x.title, x.parts, l.Loc, "text/plain")
}
func Main() {
	dataDir := os.Getenv("READERD_DATA")
	if dataDir == "" {
		var err error
		dataDir, err = os.UserConfigDir()
		if err != nil {
			log.Fatal(err)
		}
		dataDir = filepath.Join(dataDir, "pocket-reader")
	}
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatal(err)
	}
	dbPath := os.Getenv("READERD_DB")
	if dbPath == "" {
		dbPath = filepath.Join(dataDir, "readerd.sqlite")
	}
	root := os.Getenv("READERD_BOOKS")
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatal(err)
		}
		root = filepath.Join(home, "Books")
	}
	addr := os.Getenv("READERD_ADDR")
	if addr == "" {
		addr = "127.0.0.1:8765"
	}
	if !filepath.IsAbs(dbPath) {
		dbPath = filepath.Join(dataDir, dbPath)
	}
	if !filepath.IsAbs(root) {
		root = filepath.Join(dataDir, root)
	}
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		log.Fatal(err)
	}
	if err := os.MkdirAll(root, 0755); err != nil {
		log.Fatal(err)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	s := &Server{db: db, root: root}
	if err = s.migrate(); err != nil {
		log.Fatal(err)
	}
	hasBooks, err := hasSupportedFile(root)
	if err != nil {
		log.Fatal(err)
	}
	if hasBooks {
		if _, err := db.Exec("DELETE FROM books WHERE mock=1"); err != nil {
			log.Fatal(err)
		}
		s.doScan(context.Background())
	} else if err = s.seedMock(); err != nil {
		log.Fatal(err)
	}
	log.Printf("readerd %s listening on %s", version, addr)
	log.Fatal(http.ListenAndServe(addr, s.routes()))
}
func (s *Server) migrate() error {
	_, e := s.db.Exec(`PRAGMA foreign_keys=ON; CREATE TABLE IF NOT EXISTS schema_version(version INTEGER NOT NULL); INSERT INTO schema_version SELECT 1 WHERE NOT EXISTS(SELECT 1 FROM schema_version); CREATE TABLE IF NOT EXISTS books(id INTEGER PRIMARY KEY,title TEXT NOT NULL,author TEXT NOT NULL DEFAULT '',series TEXT NOT NULL DEFAULT '',language TEXT NOT NULL DEFAULT '',format TEXT NOT NULL DEFAULT '',path TEXT UNIQUE,size INTEGER NOT NULL DEFAULT 0,modified INTEGER NOT NULL DEFAULT 0,cover TEXT NOT NULL DEFAULT '',tags TEXT NOT NULL DEFAULT '[]',favorite INTEGER NOT NULL DEFAULT 0,mock INTEGER NOT NULL DEFAULT 0); CREATE TABLE IF NOT EXISTS authors(id INTEGER PRIMARY KEY,name TEXT UNIQUE NOT NULL); CREATE TABLE IF NOT EXISTS collections(id INTEGER PRIMARY KEY,name TEXT UNIQUE NOT NULL,description TEXT NOT NULL DEFAULT ''); CREATE TABLE IF NOT EXISTS collection_books(collection_id INTEGER REFERENCES collections(id) ON DELETE CASCADE,book_id INTEGER REFERENCES books(id) ON DELETE CASCADE,PRIMARY KEY(collection_id,book_id)); CREATE TABLE IF NOT EXISTS bookmarks(id INTEGER PRIMARY KEY,book_id INTEGER NOT NULL REFERENCES books(id) ON DELETE CASCADE,loc TEXT NOT NULL,text TEXT NOT NULL DEFAULT '',created_at TEXT NOT NULL); CREATE TABLE IF NOT EXISTS highlights(id INTEGER PRIMARY KEY,book_id INTEGER NOT NULL REFERENCES books(id) ON DELETE CASCADE,loc TEXT NOT NULL,text TEXT NOT NULL,color TEXT NOT NULL DEFAULT 'yellow',created_at TEXT NOT NULL); CREATE TABLE IF NOT EXISTS notes(id INTEGER PRIMARY KEY,book_id INTEGER NOT NULL REFERENCES books(id) ON DELETE CASCADE,loc TEXT NOT NULL DEFAULT '',text TEXT NOT NULL,created_at TEXT NOT NULL,updated_at TEXT NOT NULL); CREATE TABLE IF NOT EXISTS reading_progress(book_id INTEGER PRIMARY KEY REFERENCES books(id) ON DELETE CASCADE,chapter TEXT NOT NULL DEFAULT '',loc TEXT NOT NULL DEFAULT '',page INTEGER NOT NULL DEFAULT 0,percent REAL NOT NULL DEFAULT 0,updated_at TEXT NOT NULL,device TEXT NOT NULL DEFAULT ''); CREATE TABLE IF NOT EXISTS history(id INTEGER PRIMARY KEY,book_id INTEGER REFERENCES books(id) ON DELETE CASCADE,action TEXT NOT NULL,created_at TEXT NOT NULL); CREATE TABLE IF NOT EXISTS settings(key TEXT PRIMARY KEY,value TEXT NOT NULL); CREATE TABLE IF NOT EXISTS rss_subscriptions(id INTEGER PRIMARY KEY,title TEXT NOT NULL,url TEXT UNIQUE NOT NULL,enabled INTEGER NOT NULL DEFAULT 1,created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP); CREATE TABLE IF NOT EXISTS rss_items(id INTEGER PRIMARY KEY,subscription_id INTEGER REFERENCES rss_subscriptions(id) ON DELETE CASCADE,title TEXT NOT NULL,url TEXT NOT NULL,content TEXT NOT NULL DEFAULT '',published_at TEXT NOT NULL DEFAULT '',cached_at TEXT,read_later INTEGER NOT NULL DEFAULT 0,UNIQUE(subscription_id,url)); CREATE VIRTUAL TABLE IF NOT EXISTS book_fts USING fts5(title,author,body,book_id UNINDEXED); CREATE VIRTUAL TABLE IF NOT EXISTS note_fts USING fts5(text,book_id UNINDEXED);`)
	return e
}
func (s *Server) seedMock() error {
	var n int
	if e := s.db.QueryRow("SELECT count(*) FROM books").Scan(&n); e != nil {
		return e
	}
	if n > 0 {
		return nil
	}
	tx, e := s.db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	genres := []string{"Literary Fiction", "Science", "History", "Fantasy", "Technology", "Poetry", "Comics", "Reference", "Mystery", "Travel"}
	for i := 1; i <= 100; i++ {
		title := fmt.Sprintf("%s: Volume %02d", genres[(i-1)%len(genres)], i)
		tags, _ := json.Marshal([]string{genres[(i-1)%len(genres)], "Demo"})
		author := fmt.Sprintf("Author %02d", (i-1)%25+1)
		format := []string{"EPUB", "PDF", "TXT", "Markdown"}[(i-1)%4]
		_, e = tx.Exec("INSERT INTO books(title,author,series,language,format,size,cover,tags,mock) VALUES(?,?,?,?,?,?,?,?,1)", title, author, fmt.Sprintf("Collection %02d", (i-1)/5+1), []string{"en", "zh", "ja"}[(i-1)%3], format, int64(50000+i*733), "/api/books/mock/cover", string(tags))
		if e != nil {
			return e
		}
	}
	return tx.Commit()
}
func (s *Server) routes() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("/api/health", s.health)
	m.HandleFunc("/api/home", s.home)
	m.HandleFunc("/api/books", s.books)
	m.HandleFunc("/api/books/", s.book)
	m.HandleFunc("/api/bookmarks", s.bookmarks)
	m.HandleFunc("/api/bookmarks/", s.bookmark)
	m.HandleFunc("/api/highlights", s.highlights)
	m.HandleFunc("/api/highlights/", s.highlight)
	m.HandleFunc("/api/notes/export", s.exportNotes)
	m.HandleFunc("/api/notes", s.notes)
	m.HandleFunc("/api/notes/", s.note)
	m.HandleFunc("/api/search", s.search)
	m.HandleFunc("/api/collections", s.collections)
	m.HandleFunc("/api/collections/", s.collection)
	m.HandleFunc("/api/scan", s.scanStart)
	m.HandleFunc("/api/scan/status", s.scanStatus)
	m.HandleFunc("/api/settings", s.settings)
	m.HandleFunc("/api/opds/catalog", s.opds)
	m.HandleFunc("/api/rss", s.rssReservation)
	return m
}
func (s *Server) rssReservation(w http.ResponseWriter, r *http.Request) {
	fail(w, 501, "not_implemented", "RSS subscriptions and offline caching are reserved for a future release")
}
func jsonOut(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(v)
}
func fail(w http.ResponseWriter, status int, code, msg string) {
	w.WriteHeader(status)
	jsonOut(w, map[string]any{"error": map[string]string{"code": code, "message": msg}})
}
func method(w http.ResponseWriter, r *http.Request, want string) bool {
	if r.Method != want {
		w.Header().Set("Allow", want)
		fail(w, 405, "method_not_allowed", "Method not allowed")
		return false
	}
	return true
}
func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	mutool, _ := exec.LookPath("mutool")
	engine := "pure-go"
	if mutool != "" {
		engine = "mutool"
	}
	var mocks, total int
	_ = s.db.QueryRow("SELECT count(*) FROM books WHERE mock=1").Scan(&mocks)
	_ = s.db.QueryRow("SELECT count(*) FROM books").Scan(&total)
	jsonOut(w, map[string]any{"version": version, "engines": map[string]any{"pdf": engine, "epub": true, "text": true}, "mock": mocks > 0 && mocks == total})
}
func (s *Server) books(w http.ResponseWriter, r *http.Request) {
	if !method(w, r, "GET") {
		return
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	page := num(r.URL.Query().Get("page"), 1)
	limit := num(r.URL.Query().Get("limit"), 20)
	if limit > 100 {
		limit = 100
	}
	where := ""
	args := []any{}
	conditions := []string{}
	if q != "" {
		conditions = append(conditions, "(title LIKE ? OR author LIKE ?)")
		args = append(args, "%"+q+"%", "%"+q+"%")
	}
	if collection := r.URL.Query().Get("collection"); collection != "" {
		conditions = append(conditions, "id IN (SELECT book_id FROM collection_books WHERE collection_id=?)")
		args = append(args, collection)
	}
	if len(conditions) > 0 {
		where = " WHERE " + strings.Join(conditions, " AND ")
	}
	var total int
	_ = s.db.QueryRow("SELECT count(*) FROM books"+where, args...).Scan(&total)
	rows, e := s.db.Query("SELECT id,title,author,series,language,format,path,size,modified,cover,tags,favorite,mock FROM books"+where+" ORDER BY id LIMIT ? OFFSET ?", append(args, limit, (page-1)*limit)...)
	if e != nil {
		fail(w, 500, "database", e.Error())
		return
	}
	defer rows.Close()
	items := []Book{}
	for rows.Next() {
		b, e := scanBook(rows)
		if e == nil {
			_ = s.db.QueryRow("SELECT percent FROM reading_progress WHERE book_id=?", b.ID).Scan(&b.Progress)
			items = append(items, b)
		}
	}
	jsonOut(w, map[string]any{"items": items, "total": total, "page": page, "limit": limit})
}
func scanBook(r interface{ Scan(...any) error }) (Book, error) {
	var b Book
	var tags string
	var fav, mock int
	e := r.Scan(&b.ID, &b.Title, &b.Author, &b.Series, &b.Language, &b.Format, &b.Path, &b.Size, &b.Modified, &b.Cover, &tags, &fav, &mock)
	_ = json.Unmarshal([]byte(tags), &b.Tags)
	b.Favorite = fav != 0
	b.Mock = mock != 0
	return b, e
}
func num(s string, d int) int {
	n, e := strconv.Atoi(s)
	if e != nil || n < 1 {
		return d
	}
	return n
}
