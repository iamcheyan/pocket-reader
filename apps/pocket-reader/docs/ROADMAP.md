# Roadmap

## Implemented

- Local readerd + React/Vite architecture and REST boundary.
- EPUB, PDF text extraction, TXT, and Markdown providers; persistent library and annotations.
- Responsive library and reader UX, four themes, English/简体中文/日本語 locales, keyboard and gamepad input.
  Full-text search over book bodies is not implemented. `/api/search` covers title and author; per-book search uses document providers. FTS5 tables are created but not populated or queried.

## Known limits

- PDF extraction is text-oriented. MuPDF is optional and selected only when `mutool` exists; pure-Go extraction is the fallback. Complex layouts and image-only PDFs need OCR, which is not included.
- RSS subscription, offline cache, and read-later behavior are reserved but not implemented.
- Calibre content-server integration is designed as a future provider adapter; it has not been connected or live-tested.
- DRM-protected books and MOBI/AZW3/FB2/CBZ/CBR are not supported.

## PJOS migration steps

1. Package readerd as a managed PJOS service with its data directory on persistent storage.
2. Register the static Vite build as a PocketJS application and route `/api` to the local readerd endpoint.
3. Bind native page keys and lifecycle events through PocketJS adapters while retaining the browser keyboard mappings.
4. Integrate device-specific E-Ink refresh scheduling and test at physical panel sizes and DPI.
5. Add sync transport after a conflict model and identity policy are selected.
