# readerd REST API

All endpoints are rooted at `/api`, use JSON unless returning a cover/export, and report failures as `{"error":{"code":"...","message":"..."}}`.

| Method | Endpoint | Purpose |
|---|---|---|
| GET | `/health` | Version, available engines, mock mode |
| GET | `/home` | Continue reading, recently added, favorites, notes, collections |
| GET | `/books?q=&collection=&page=&limit=` | Paginated library |
| GET | `/books/{id}` | Book metadata |
| GET | `/books/{id}/cover` | Cover asset |
| GET | `/books/{id}/toc` | Provider navigation |
| GET | `/books/{id}/content?loc=` | Current content chunk |
| GET | `/books/{id}/search?q=` | Search within a document |
| GET/PUT | `/books/{id}/progress` | Read and persist location and percentage |
| POST/DELETE | `/books/{id}/favorite` | Set favorite / clear favorite |
| GET/POST | `/bookmarks?book_id=` | List/create bookmark |
| DELETE | `/bookmarks/{id}` | Delete bookmark |
| GET/POST | `/highlights?book_id=` | List/create colored highlight |
| DELETE | `/highlights/{id}` | Delete highlight |
| GET/POST | `/notes?book_id=&q=` | List/search/create notes |
| PUT/DELETE | `/notes/{id}` | Edit/delete note |
| GET | `/notes/export?format=md|json` | Export notes |
| GET | `/search?q=` | Search catalog titles and authors |
| GET/POST | `/collections` | List/create collections |
| PUT/DELETE | `/collections/{id}` | Edit/delete collection |
| POST | `/scan` | Start recursive incremental scan; DELETE cancels |
| GET | `/scan/status` | Current progress and scan errors |
| GET/PUT | `/settings` | Persist/read settings |
| GET | `/opds/catalog?url=` | Fetch and parse an OPDS Atom feed |
| GET/POST | `/rss` | Reserved RSS route; returns 501 until subscriptions and offline caching are implemented |

Content chunks include `has_next`, `has_previous`, `next_loc`, and `previous_loc` when those locations exist. Locations remain provider-specific strings.

Book content stays in readerd and is returned one provider chunk at a time. Client location values are opaque strings understood by the provider. OPDS supports Atom feeds; a configured Calibre server can be used as its feed URL. Calibre's book-content APIs are not a separate connected provider yet.
