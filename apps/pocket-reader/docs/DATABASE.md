# Database

SQLite uses the pure-Go `modernc.org/sqlite` driver and creates/migrates its schema at service startup. Core tables include books, authors, collections and memberships, bookmarks, highlights, notes, reading progress, history, settings, and reserved RSS subscription/cache/read-later records. Progress stores chapter, location/page, percentage, timestamp, and optional device identity.

The catalog indexes title and author for cross-library search. Extracted full text is not copied into SQLite: it can be large, duplicates source data, and can expose copyrighted content in backups. Per-document search asks the provider to scan content on demand. This keeps the database small but makes full-text search slower for large books. FTS5 is used for catalog/annotation text where supported by the linked SQLite build; migrations retain a fallback catalog query.

Book records are refreshed incrementally using path, modification time, and file size. Annotation locations are provider-neutral strings so saved references survive frontend layout changes.
