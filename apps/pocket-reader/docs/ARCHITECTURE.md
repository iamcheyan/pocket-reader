# Pocket Reader Architecture

## Runtime boundaries

The React + TypeScript + Vite UI is a client of `readerd`; it has no filesystem or database access. The Go service owns scanning, metadata, document providers, OPDS requests, persistence, and search. REST JSON is the local desktop/PJOS process boundary. The Vite development server proxies `/api` to readerd; deployment serves the static frontend and service separately.

## Services

- HTTP API: health, library/home, document navigation, reading state, annotations, collections, settings, scan control, cross-library search, OPDS.
- Library: recursive incremental scan, metadata extraction, optional cover extraction, virtual catalog when no supported books exist.
- Document engine: provider registry for EPUB, PDF, TXT, Markdown. EPUB ZIP/XML resources and PDF text extraction are delegated to Go libraries; PDF uses mutool if available and a pure-Go fallback otherwise.
- Persistence: modernc SQLite, versioned schema, FTS indexes. Per-user data stays under the application data directory.

## PJOS migration

Keep the REST contract stable. Package readerd as a PJOS service and point the frontend API base at the local runtime endpoint. Replace only process startup and static asset hosting adapters for a native PocketJS host. Use the device scheduler to coordinate suspend/resume and E-Ink partial refresh; no engine belongs in the UI layer.
