# Database

SQLite uses the pure-Go `modernc.org/sqlite` driver and creates/migrates its schema at service startup. Core tables include books, authors, collections and memberships, bookmarks, highlights, notes, reading progress, history, settings, and reserved RSS subscription/cache/read-later records. Progress stores chapter, location/page, percentage, timestamp, and optional device identity.
The catalog indexes title and author for cross-library search. The current `/api/search` endpoint searches those two fields only; it does not yet search extracted book text or annotations. Extracted text is not copied into SQLite because it can be large, duplicates source data, and can expose copyrighted content in backups. Per-book search does scan provider content on demand. This keeps the database small but makes full-text search slower for large books. The schema creates FTS5 tables for future catalog/body and note indexing; they are not currently populated or queried.

Book records are refreshed incrementally using path, modification time, and file size. Annotation locations are provider-neutral strings so saved references survive frontend layout changes.
