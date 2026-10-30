package readerd

import (
	"context"
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func (s *Server) book(w http.ResponseWriter, r *http.Request) {
	p := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/books/"), "/"), "/")
	if len(p) == 0 || p[0] == "" {
		fail(w, 404, "not_found", "Book not found")
		return
	}
	id := p[0]
	if len(p) == 2 && p[1] == "cover" {
		b, e := s.bookPath(id)
		if e != nil {
			fail(w, 404, "not_found", "Book not found")
			return
		}
		if b.Mock {
			w.Header().Set("Content-Type", "image/svg+xml")
			_, _ = io.WriteString(w, fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="300" height="420" viewBox="0 0 300 420"><rect width="300" height="420" fill="#e8e0d3"/><rect x="14" y="14" width="272" height="392" rx="9" fill="#839b83"/><text x="32" y="74" fill="white" font-family="sans-serif" font-size="14">%s</text><text x="32" y="190" fill="white" font-family="serif" font-size="23">%s</text><text x="32" y="375" fill="white" font-family="sans-serif" font-size="14">%s</text></svg>`, html.EscapeString(b.Format), html.EscapeString(b.Title), html.EscapeString(b.Author)))
			return
		}
		fail(w, 404, "cover_not_found", "Book has no cover")
		return
	}
	if len(p) == 1 {
		b, e := s.bookPath(id)
		if e != nil {
			fail(w, 404, "not_found", "Book not found")
			return
		}
		jsonOut(w, b)
		return
	}
	if p[1] == "favorite" {
		switch r.Method {
		case "POST":
			_, e := s.db.Exec("UPDATE books SET favorite=1 WHERE id=?", id)
			if e != nil {
				fail(w, 500, "database", e.Error())
				return
			}
			jsonOut(w, map[string]bool{"favorite": true})
		case "DELETE":
			if _, err := s.db.Exec("UPDATE books SET favorite=0 WHERE id=?", id); err != nil {
				fail(w, 500, "database", err.Error())
				return
			}
			w.WriteHeader(204)
		default:
			fail(w, 405, "method_not_allowed", "Method not allowed")
		}
		return
	}
	if len(p) == 2 && p[1] == "progress" {
		b, e := s.bookPath(id)
		if e != nil {
			fail(w, 404, "not_found", "Book not found")
			return
		}
		switch r.Method {
		case "GET":
			var ch, loc, updated, device string
			var page int
			var percent float64
			e := s.db.QueryRow("SELECT chapter,loc,page,percent,updated_at,device FROM reading_progress WHERE book_id=?", id).Scan(&ch, &loc, &page, &percent, &updated, &device)
			if e == sql.ErrNoRows {
				jsonOut(w, map[string]any{"book_id": b.ID, "chapter": "", "loc": "0", "page": 0, "percent": 0})
				return
			}
			if e != nil {
				fail(w, 500, "database", e.Error())
				return
			}
			jsonOut(w, map[string]any{"book_id": b.ID, "chapter": ch, "loc": loc, "page": page, "percent": percent, "updated_at": updated, "device": device})
		case "PUT":
			var v struct {
				Chapter string          `json:"chapter"`
				Loc     json.RawMessage `json:"loc"`
				Percent float64         `json:"percent"`
				Page    int             `json:"page"`
				Device  string          `json:"device"`
			}
			if json.NewDecoder(r.Body).Decode(&v) != nil {
				fail(w, 400, "invalid_json", "Invalid JSON")
				return
			}
			var location any
			if len(v.Loc) > 0 {
				if err := json.Unmarshal(v.Loc, &location); err != nil {
					fail(w, 400, "invalid_location", "loc must be a string or number")
					return
				}
				switch location.(type) {
				case string, float64:
				default:
					fail(w, 400, "invalid_location", "loc must be a string or number")
					return
				}
			}
			loc := ""
			if location != nil {
				loc = fmt.Sprint(location)
			}
			now := time.Now().UTC().Format(time.RFC3339)
			_, e := s.db.Exec("INSERT INTO reading_progress(book_id,chapter,loc,page,percent,updated_at,device) VALUES(?,?,?,?,?,?,?) ON CONFLICT(book_id) DO UPDATE SET chapter=excluded.chapter,loc=excluded.loc,page=excluded.page,percent=excluded.percent,updated_at=excluded.updated_at,device=excluded.device", id, v.Chapter, loc, v.Page, v.Percent, now, v.Device)
			if e != nil {
				fail(w, 500, "database", e.Error())
				return
			}
			_, _ = s.db.Exec("INSERT INTO history(book_id,action,created_at) VALUES(?,?,?)", id, "progress", now)
			jsonOut(w, map[string]any{"book_id": b.ID, "chapter": v.Chapter, "loc": location, "page": v.Page, "percent": v.Percent, "updated_at": now, "device": v.Device})
		default:
			fail(w, 405, "method_not_allowed", "Method not allowed")
		}
		return
	}
	if len(p) == 2 && (p[1] == "toc" || p[1] == "content" || p[1] == "search") {
		b, e := s.bookPath(id)
		if e != nil {
			fail(w, 404, "not_found", "Book not found")
			return
		}
		var pr Provider
		var doc Document
		if b.Mock {
			mp := mockProvider{book: b}
			pr = mp
			doc = mp.doc()
		} else {
			pr = providerFor(b.Path)
			if pr == nil {
				fail(w, 415, "unsupported_format", "Unsupported book format")
				return
			}
			doc, e = pr.Open(r.Context(), b.Path)
			if e != nil {
				fail(w, 422, "document_error", e.Error())
				return
			}
		}
		switch p[1] {
		case "toc":
			x, e := pr.TOC(r.Context(), doc)
			if e != nil {
				fail(w, 500, "document_error", e.Error())
				return
			}
			jsonOut(w, map[string]any{"items": x})
		case "search":
			hits, e := pr.Search(r.Context(), doc, r.URL.Query().Get("q"))
			if e != nil {
				fail(w, 500, "document_error", e.Error())
				return
			}
			jsonOut(w, map[string]any{"items": hits})
		case "content":
			v, e := pr.Content(r.Context(), doc, Location{Loc: r.URL.Query().Get("loc")})
			if e != nil {
				fail(w, 400, "invalid_location", e.Error())
				return
			}
			v.BookID = id
			jsonOut(w, v)
		}
		return
	}
	fail(w, 404, "not_found", "Endpoint not found")
}
func (s *Server) bookPath(id string) (Book, error) {
	var b Book
	var tags sql.NullString
	var p sql.NullString
	var f, m int
	e := s.db.QueryRow("SELECT id,title,author,series,language,format,path,size,modified,cover,tags,favorite,mock FROM books WHERE id=?", id).Scan(&b.ID, &b.Title, &b.Author, &b.Series, &b.Language, &b.Format, &p, &b.Size, &b.Modified, &b.Cover, &tags, &f, &m)
	b.Mock = m != 0
	if e == nil {
		b.Path = p.String
		_ = json.Unmarshal([]byte(tags.String), &b.Tags)
		b.Favorite = f != 0
		_ = s.db.QueryRow("SELECT percent FROM reading_progress WHERE book_id=?", b.ID).Scan(&b.Progress)
	}
	return b, e
}
func providerFor(p string) Provider {
	for _, v := range []Provider{EPUBProvider{}, PDFProvider{}, TextProvider{Ext: ".txt"}, TextProvider{Ext: ".md"}, TextProvider{Ext: ".markdown"}} {
		if v.Supports(p) {
			return v
		}
	}
	return nil
}
func (s *Server) home(w http.ResponseWriter, r *http.Request) {
	recent := s.queryBooks("SELECT id,title,author,series,language,format,path,size,modified,cover,tags,favorite,mock FROM books ORDER BY id DESC LIMIT 12")
	cont := s.queryBooks("SELECT b.id,b.title,b.author,b.series,b.language,b.format,b.path,b.size,b.modified,b.cover,b.tags,b.favorite,b.mock FROM books b JOIN reading_progress p ON b.id=p.book_id ORDER BY p.updated_at DESC LIMIT 10")
	jsonOut(w, map[string]any{"continue_reading": cont, "recently_added": recent, "favorites": s.bookQuery("favorite=1"), "recent_notes": s.notesQuery("", 10), "collections": s.collectionQuery()})
}
func (s *Server) queryBooks(query string, args ...any) []Book {
	rows, e := s.db.Query(query, args...)
	if e != nil {
		return []Book{}
	}
	defer rows.Close()
	out := []Book{}
	for rows.Next() {
		b, err := scanBook(rows)
		if err == nil {
			out = append(out, b)
		}
	}
	return out
}
func (s *Server) bookQuery(where string) []Book {
	rows, e := s.db.Query("SELECT id,title,author,series,language,format,path,size,modified,cover,tags,favorite,mock FROM books WHERE " + where + " ORDER BY id DESC LIMIT 10")
	if e != nil {
		return []Book{}
	}
	defer rows.Close()
	a := []Book{}
	for rows.Next() {
		b, e := scanBook(rows)
		if e == nil {
			a = append(a, b)
		}
	}
	return a
}
func (s *Server) notesQuery(book string, limit int) []map[string]any {
	q := "SELECT n.id,n.book_id,n.loc,n.text,n.created_at,n.updated_at,COALESCE(b.title,'') FROM notes n LEFT JOIN books b ON b.id=n.book_id"
	args := []any{}
	if book != "" {
		q += " WHERE n.book_id=?"
		args = append(args, book)
	}
	q += " ORDER BY n.updated_at DESC LIMIT ?"
	args = append(args, limit)
	rows, e := s.db.Query(q, args...)
	if e != nil {
		return []map[string]any{}
	}
	defer rows.Close()
	return noteRows(rows)
}
func (s *Server) bookmarks(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" && r.Method != "POST" {
		w.Header().Set("Allow", "GET, POST")
		fail(w, 405, "method_not_allowed", "Method not allowed")
		return
	}
	if r.Method == "POST" {
		var v struct {
			BookID int64  `json:"book_id"`
			Loc    string `json:"loc"`
			Text   string `json:"text"`
		}
		if json.NewDecoder(r.Body).Decode(&v) != nil {
			fail(w, 400, "invalid_json", "Invalid JSON")
			return
		}
		res, e := s.db.Exec("INSERT INTO bookmarks(book_id,loc,text,created_at) VALUES(?,?,?,?)", v.BookID, v.Loc, v.Text, time.Now().UTC().Format(time.RFC3339))
		if e != nil {
			fail(w, 400, "invalid_bookmark", e.Error())
			return
		}
		id, _ := res.LastInsertId()
		jsonOut(w, map[string]any{"id": id, "book_id": v.BookID, "loc": v.Loc, "text": v.Text})
		return
	}
	q := "SELECT id,book_id,loc,text,created_at FROM bookmarks"
	args := []any{}
	if x := r.URL.Query().Get("book_id"); x != "" {
		q += " WHERE book_id=?"
		args = append(args, x)
	}
	rows, e := s.db.Query(q+" ORDER BY created_at DESC", args...)
	if e != nil {
		fail(w, 500, "database", e.Error())
		return
	}
	defer rows.Close()
	a := []map[string]any{}
	for rows.Next() {
		var id, b int64
		var loc, text, at string
		rows.Scan(&id, &b, &loc, &text, &at)
		a = append(a, map[string]any{"id": id, "book_id": b, "loc": loc, "text": text, "created_at": at})
	}
	jsonOut(w, map[string]any{"items": a})
}
func (s *Server) bookmark(w http.ResponseWriter, r *http.Request) {
	if !method(w, r, "DELETE") {
		return
	}
	_, e := s.db.Exec("DELETE FROM bookmarks WHERE id=?", strings.TrimPrefix(r.URL.Path, "/api/bookmarks/"))
	if e != nil {
		fail(w, 500, "database", e.Error())
		return
	}
	w.WriteHeader(204)
}
func (s *Server) highlights(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		var v struct {
			BookID int64  `json:"book_id"`
			Loc    string `json:"loc"`
			Text   string `json:"text"`
			Color  string `json:"color"`
		}
		if json.NewDecoder(r.Body).Decode(&v) != nil {
			fail(w, 400, "invalid_json", "Invalid JSON")
			return
		}
		if v.Color == "" {
			v.Color = "yellow"
		}
		res, e := s.db.Exec("INSERT INTO highlights(book_id,loc,text,color,created_at) VALUES(?,?,?,?,?)", v.BookID, v.Loc, v.Text, v.Color, time.Now().UTC().Format(time.RFC3339))
		if e != nil {
			fail(w, 400, "invalid_highlight", e.Error())
			return
		}
		id, _ := res.LastInsertId()
		jsonOut(w, map[string]any{"id": id, "book_id": v.BookID, "loc": v.Loc, "text": v.Text, "color": v.Color})
		return
	}
	if !method(w, r, "GET") {
		return
	}
	bookID := r.URL.Query().Get("book_id")
	q := "SELECT id,book_id,loc,text,color,created_at FROM highlights"
	args := []any{}
	if bookID != "" {
		q += " WHERE book_id=?"
		args = append(args, bookID)
	}
	rows, e := s.db.Query(q+" ORDER BY created_at DESC", args...)
	if e != nil {
		fail(w, 500, "database", e.Error())
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, book int64
		var loc, text, color, created string
		if rows.Scan(&id, &book, &loc, &text, &color, &created) == nil {
			items = append(items, map[string]any{"id": id, "book_id": book, "loc": loc, "text": text, "color": color, "created_at": created})
		}
	}
	jsonOut(w, map[string]any{"items": items})
}
func (s *Server) highlight(w http.ResponseWriter, r *http.Request) {
	if !method(w, r, "DELETE") {
		return
	}
	_, e := s.db.Exec("DELETE FROM highlights WHERE id=?", strings.TrimPrefix(r.URL.Path, "/api/highlights/"))
	if e != nil {
		fail(w, 500, "database", e.Error())
		return
	}
	w.WriteHeader(204)
}
func (s *Server) notes(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		var v struct {
			BookID    int64 `json:"book_id"`
			Loc, Text string
		}
		if json.NewDecoder(r.Body).Decode(&v) != nil {
			fail(w, 400, "invalid_json", "Invalid JSON")
			return
		}
		now := time.Now().UTC().Format(time.RFC3339)
		res, e := s.db.Exec("INSERT INTO notes(book_id,loc,text,created_at,updated_at) VALUES(?,?,?,?,?)", v.BookID, v.Loc, v.Text, now, now)
		if e != nil {
			fail(w, 400, "invalid_note", e.Error())
			return
		}
		id, _ := res.LastInsertId()
		jsonOut(w, map[string]any{"id": id, "book_id": v.BookID, "loc": v.Loc, "text": v.Text, "created_at": now, "updated_at": now})
		return
	}
	if !method(w, r, "GET") {
		return
	}
	book := r.URL.Query().Get("book_id")
	all := s.notesQuery(book, 500)
	q := strings.ToLower(r.URL.Query().Get("q"))
	if q != "" {
		out := all[:0]
		for _, n := range all {
			if strings.Contains(strings.ToLower(fmt.Sprint(n["text"])), q) {
				out = append(out, n)
			}
		}
		all = out
	}
	jsonOut(w, map[string]any{"items": all})
}
func noteRows(rows *sql.Rows) []map[string]any {
	a := []map[string]any{}
	for rows.Next() {
		var id, b int64
		var loc, text, created, updated, title string
		if rows.Scan(&id, &b, &loc, &text, &created, &updated, &title) == nil {
			a = append(a, map[string]any{"id": id, "book_id": b, "loc": loc, "text": text, "created_at": created, "updated_at": updated, "book_title": title})
		}
	}
	return a
}
func (s *Server) note(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/notes/")
	switch r.Method {
	case "PUT":
		var v struct {
			Text string `json:"text"`
			Loc  string `json:"loc"`
		}
		if json.NewDecoder(r.Body).Decode(&v) != nil {
			fail(w, 400, "invalid_json", "Invalid JSON")
			return
		}
		_, e := s.db.Exec("UPDATE notes SET text=?,loc=?,updated_at=? WHERE id=?", v.Text, v.Loc, time.Now().UTC().Format(time.RFC3339), id)
		if e != nil {
			fail(w, 500, "database", e.Error())
			return
		}
		jsonOut(w, map[string]any{"id": id, "text": v.Text, "loc": v.Loc})
	case "DELETE":
		_, e := s.db.Exec("DELETE FROM notes WHERE id=?", id)
		if e != nil {
			fail(w, 500, "database", e.Error())
			return
		}
		w.WriteHeader(204)
	default:
		fail(w, 405, "method_not_allowed", "Method not allowed")
	}
}
func (s *Server) exportNotes(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	notes := s.notesQuery("", 10000)
	if format == "json" {
		jsonOut(w, map[string]any{"items": notes})
		return
	}
	if format != "" && format != "md" {
		fail(w, 400, "invalid_format", "format must be md or json")
		return
	}
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	for _, n := range notes {
		fmt.Fprintf(w, "## Book %v\n\n%s\n\n", n["book_id"], n["text"])
	}
}
func (s *Server) search(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		jsonOut(w, map[string]any{"items": []any{}})
		return
	}
	like := "%" + q + "%"
	rows, e := s.db.Query("SELECT id,title,author,series,language,format,path,size,modified,cover,tags,favorite,mock FROM books WHERE title LIKE ? OR author LIKE ? ORDER BY title LIMIT 100", like, like)
	if e != nil {
		fail(w, 500, "database", e.Error())
		return
	}
	defer rows.Close()
	a := []Book{}
	for rows.Next() {
		b, e := scanBook(rows)
		if e == nil {
			a = append(a, b)
		}
	}
	jsonOut(w, map[string]any{"items": a})
}
func (s *Server) collections(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		var v struct {
			Name        string  `json:"name"`
			Description string  `json:"description"`
			BookIDs     []int64 `json:"book_ids"`
		}
		if json.NewDecoder(r.Body).Decode(&v) != nil {
			fail(w, 400, "invalid_json", "Invalid JSON")
			return
		}
		tx, e := s.db.Begin()
		if e != nil {
			fail(w, 500, "database", e.Error())
			return
		}
		res, e := tx.Exec("INSERT INTO collections(name,description) VALUES(?,?)", v.Name, v.Description)
		if e != nil {
			tx.Rollback()
			fail(w, 400, "collection_error", e.Error())
			return
		}
		id, _ := res.LastInsertId()
		for _, book := range v.BookIDs {
			if _, e = tx.Exec("INSERT INTO collection_books(collection_id,book_id) VALUES(?,?)", id, book); e != nil {
				tx.Rollback()
				fail(w, 400, "collection_error", e.Error())
				return
			}
		}
		if e = tx.Commit(); e != nil {
			fail(w, 500, "database", e.Error())
			return
		}
		jsonOut(w, map[string]any{"id": id, "name": v.Name, "description": v.Description, "book_ids": v.BookIDs})
		return
	}
	if !method(w, r, "GET") {
		return
	}
	jsonOut(w, map[string]any{"items": s.collectionQuery()})
}
func (s *Server) collectionQuery() []map[string]any {
	rows, e := s.db.Query("SELECT c.id,c.name,c.description,COALESCE(group_concat(cb.book_id), '') FROM collections c LEFT JOIN collection_books cb ON cb.collection_id=c.id GROUP BY c.id ORDER BY c.name")
	if e != nil {
		return []map[string]any{}
	}
	defer rows.Close()
	a := []map[string]any{}
	for rows.Next() {
		var id int64
		var name, desc, ids string
		rows.Scan(&id, &name, &desc, &ids)
		bookIDs := []int64{}
		for _, raw := range strings.Split(ids, ",") {
			if n, e := strconv.ParseInt(raw, 10, 64); e == nil {
				bookIDs = append(bookIDs, n)
			}
		}
		a = append(a, map[string]any{"id": id, "name": name, "description": desc, "book_ids": bookIDs, "book_count": len(bookIDs)})
	}
	return a
}
func (s *Server) collection(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/collections/")
	switch r.Method {
	case "PUT":
		var v struct {
			Name        string  `json:"name"`
			Description string  `json:"description"`
			BookIDs     []int64 `json:"book_ids"`
		}
		if json.NewDecoder(r.Body).Decode(&v) != nil {
			fail(w, 400, "invalid_json", "Invalid JSON")
			return
		}
		tx, e := s.db.Begin()
		if e != nil {
			fail(w, 500, "database", e.Error())
			return
		}
		res, e := tx.Exec("UPDATE collections SET name=?,description=? WHERE id=?", v.Name, v.Description, id)
		if e != nil {
			tx.Rollback()
			fail(w, 400, "collection_error", e.Error())
			return
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			tx.Rollback()
			fail(w, 404, "not_found", "Collection not found")
			return
		}
		if _, e = tx.Exec("DELETE FROM collection_books WHERE collection_id=?", id); e == nil {
			for _, book := range v.BookIDs {
				if _, e = tx.Exec("INSERT INTO collection_books(collection_id,book_id) VALUES(?,?)", id, book); e != nil {
					break
				}
			}
		}
		if e != nil {
			tx.Rollback()
			fail(w, 400, "collection_error", e.Error())
			return
		}
		if e = tx.Commit(); e != nil {
			fail(w, 500, "database", e.Error())
			return
		}
		jsonOut(w, map[string]any{"id": id, "name": v.Name, "description": v.Description, "book_ids": v.BookIDs})
	case "DELETE":
		res, e := s.db.Exec("DELETE FROM collections WHERE id=?", id)
		if e != nil {
			fail(w, 500, "database", e.Error())
			return
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			fail(w, 404, "not_found", "Collection not found")
			return
		}
		w.WriteHeader(204)
	default:
		fail(w, 405, "method_not_allowed", "Method not allowed")
	}
}
func (s *Server) settings(w http.ResponseWriter, r *http.Request) {
	if r.Method == "PUT" {
		var v map[string]any
		if json.NewDecoder(r.Body).Decode(&v) != nil {
			fail(w, 400, "invalid_json", "Invalid JSON")
			return
		}
		tx, e := s.db.Begin()
		if e != nil {
			fail(w, 500, "database", e.Error())
			return
		}
		for k, val := range v {
			b, _ := json.Marshal(val)
			_, e = tx.Exec("INSERT INTO settings(key,value) VALUES(?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value", k, string(b))
			if e != nil {
				tx.Rollback()
				fail(w, 500, "database", e.Error())
				return
			}
		}
		tx.Commit()
	}
	if r.Method != "GET" && r.Method != "PUT" {
		fail(w, 405, "method_not_allowed", "Method not allowed")
		return
	}
	rows, e := s.db.Query("SELECT key,value FROM settings")
	if e != nil {
		fail(w, 500, "database", e.Error())
		return
	}
	defer rows.Close()
	v := map[string]any{}
	for rows.Next() {
		var k, b string
		rows.Scan(&k, &b)
		var x any
		if json.Unmarshal([]byte(b), &x) != nil {
			x = b
		}
		v[k] = x
	}
	jsonOut(w, v)
}

type feed struct {
	XMLName xml.Name `xml:"feed"`
	Entries []struct {
		Title string `xml:"title"`
		ID    string `xml:"id"`
		Links []struct {
			Href string `xml:"href,attr"`
			Rel  string `xml:"rel,attr"`
			Type string `xml:"type,attr"`
		} `xml:"link"`
	} `xml:"entry"`
}

func (s *Server) opds(w http.ResponseWriter, r *http.Request) {
	raw := r.URL.Query().Get("url")
	u, e := url.Parse(raw)
	if e != nil || (u.Scheme != "http" && u.Scheme != "https") {
		fail(w, 400, "invalid_url", "OPDS URL must be http or https")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", raw, nil)
	resp, e := http.DefaultClient.Do(req)
	if e != nil {
		fail(w, 502, "opds_fetch", e.Error())
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		fail(w, 502, "opds_fetch", resp.Status)
		return
	}
	var f feed
	if e = xml.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&f); e != nil {
		fail(w, 502, "opds_parse", e.Error())
		return
	}
	items := []map[string]any{}
	for _, x := range f.Entries {
		href := ""
		for _, l := range x.Links {
			if l.Rel == "http://opds-spec.org/acquisition" || strings.Contains(l.Type, "epub") {
				href = l.Href
				break
			}
		}
		items = append(items, map[string]any{"title": x.Title, "id": x.ID, "url": href})
	}
	jsonOut(w, map[string]any{"items": items})
}
func (s *Server) scanStatus(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	jsonOut(w, s.scan)
}
func (s *Server) scanStart(w http.ResponseWriter, r *http.Request) {
	if r.Method == "DELETE" {
		s.mu.Lock()
		if s.cancel != nil {
			s.cancel()
		}
		s.mu.Unlock()
		jsonOut(w, map[string]bool{"cancelled": true})
		return
	}
	if !method(w, r, "POST") {
		return
	}
	s.mu.Lock()
	if s.scan.Running {
		s.mu.Unlock()
		fail(w, 409, "scan_running", "Scan already running")
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.scan = ScanStatus{Running: true, StartedAt: time.Now().UTC()}
	s.mu.Unlock()
	go s.doScan(ctx)
	jsonOut(w, map[string]bool{"started": true})
}
func (s *Server) doScan(ctx context.Context) {
	defer func() {
		s.mu.Lock()
		s.scan.Running = false
		s.scan.FinishedAt = time.Now().UTC()
		s.cancel = nil
		s.mu.Unlock()
	}()
	_ = filepath.WalkDir(s.root, func(path string, d os.DirEntry, e error) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if e != nil {
			s.mu.Lock()
			s.scan.Errors = append(s.scan.Errors, e.Error())
			s.mu.Unlock()
			return nil
		}
		if d.IsDir() {
			return nil
		}
		pr := providerFor(path)
		if pr == nil {
			return nil
		}
		info, e := d.Info()
		if e != nil {
			return nil
		}
		var id int64
		var oldSize, oldMod int64
		er := s.db.QueryRow("SELECT id,size,modified FROM books WHERE path=?", path).Scan(&id, &oldSize, &oldMod)
		if er == nil && oldSize == info.Size() && oldMod == info.ModTime().UnixNano() {
			return nil
		}
		doc, e := pr.Open(ctx, path)
		if e != nil {
			s.mu.Lock()
			s.scan.Errors = append(s.scan.Errors, path+": "+e.Error())
			s.mu.Unlock()
			return nil
		}
		meta, e := pr.Metadata(ctx, doc)
		if e != nil {
			s.mu.Lock()
			s.scan.Errors = append(s.scan.Errors, path+": "+e.Error())
			s.mu.Unlock()
			return nil
		}
		if meta.Title == "" {
			meta.Title = strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))
		}
		format := strings.TrimPrefix(strings.ToUpper(filepath.Ext(path)), ".")
		tags, _ := json.Marshal([]string{})
		_, e = s.db.Exec("INSERT INTO books(title,author,series,language,format,path,size,modified,tags,mock) VALUES(?,?,?,?,?,?,?,?,?,0) ON CONFLICT(path) DO UPDATE SET title=excluded.title,author=excluded.author,series=excluded.series,language=excluded.language,format=excluded.format,size=excluded.size,modified=excluded.modified", meta.Title, meta.Author, "", meta.Language, format, path, info.Size(), info.ModTime().UnixNano(), string(tags))
		if e == nil {
			if meta.Author != "" {
				_, _ = s.db.Exec("INSERT INTO authors(name) VALUES(?) ON CONFLICT(name) DO NOTHING", meta.Author)
			}
			s.mu.Lock()
			if er == sql.ErrNoRows {
				s.scan.Added++
			} else {
				s.scan.Updated++
			}
			s.scan.Scanned++
			s.mu.Unlock()
		}
		return nil
	})
	if ctx.Err() != nil {
		s.mu.Lock()
		s.scan.Errors = append(s.scan.Errors, "scan cancelled")
		s.mu.Unlock()
	}
}
