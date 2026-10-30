# Build and verification status

Verified on HX90 (NixOS) on 2026-10-30.

## Automated

- `make test` passes: Go tests, locale parity (92 keys), `tsc --noEmit`, `vite build`.
- `go vet ./...` passes.

## Live runtime (readerd :8791 + Vite :5199)

- `/api/health` → `{"engines":{"epub":true,"pdf":"pure-go","text":true},"mock":true,"version":"0.1.0"}` with the demo catalog; `"mock":false` after scanning a real library.
- Demo catalog: `total: 100`, generated chapters, TOC of 8 chapters, `next_loc` navigation.
- Real recursive scan indexed 4 generated files: `TXT plain` (60 B), `EPUB` "Real EPUB Title"/"Real EPUB Author" (1542 B), `MD notes`, `PDF real` (600 B). Mock rows are removed when real books exist.
- EPUB: TOC `["Real Opening","Real Closing"]`, XHTML content, chapter search hit for `beta`.
- PDF: 1 page, extracted text `Generated PDF real phrase`.
- TXT/Markdown: content returned with Markdown rendered to `<h2>/<p>`.
- Progress: PUT then GET persisted `loc`, `percent`; a fresh page load resumed at `Page 1`.
- Annotations: bookmark via `B`, highlight via `H`, note via `N` created through the real UI; notes list returns `book_title`.
- Search: `/api/search?q=epub` returned the EPUB book; OPDS parsed a locally served Atom feed; `file://` URL rejected with 400; `/api/rss` returned 501.
- Keyboard page turn confirmed via CDP: content and location advanced from `Page 0` to `Page 1`.

## Screenshots

22 PNGs in `docs/screenshots`, captured from the running app and dimension-verified from PNG headers; see `SCREENSHOTS.md`.

## Limits

- `mutool` is not on `PATH`; the pure-Go PDF fallback was exercised instead. The MuPDF command path is implemented but untested here.
- Image-only/scanned PDFs need OCR, which is not included.
- Cross-library search covers titles and authors only; book-body full-text search is not implemented.
- RSS behavior and Calibre content-server integration remain reserved/undocumented-in-practice (interface and schema only).
- Dependency installation required a SOCKS proxy on this host because the direct route lacked DNS.
