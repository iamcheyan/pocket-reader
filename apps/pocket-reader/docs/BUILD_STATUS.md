# Build and verification status

Verified on this host (HX90, NixOS):

- `make test` passes: Go tests in `apps/pocket-reader/backend` (providers, scan, progress/resume, annotations, export round-trip, OPDS) plus frontend locale parity (92 keys), `tsc --noEmit` and `vite build`.
- Backend `go vet ./...` passes.
- Live runtime check: readerd served 100 mock books with generated chapter content, TOC, navigation, progress, bookmarks, highlights and notes.
- Live file scan: a recursive temporary library containing generated EPUB, PDF, TXT and Markdown files indexed all four formats with correct size/format and mock mode disabled.
- OPDS: readerd parsed a locally served Atom feed and rejected a non-HTTP URL with 400.
- Screenshots were captured from the running app; dimensions were checked from PNG IHDR headers.

Environment notes:

- `mutool` is not on `PATH`, so PDF uses the pure-Go fallback. The preferred MuPDF path is implemented but was not exercised here.
- Network access from this host required an IPv6 route; dependency installation used the local SOCKS proxy at `127.0.0.1:1080`. The pinned PDF module version originally recorded by the scaffold did not exist upstream and was corrected to a published revision.
