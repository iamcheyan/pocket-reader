# Dependencies and document engines

## Build and run

- Go 1.24 or newer for readerd.
- Frontend npm dependencies are declared in `apps/pocket-reader/frontend/package.json`; install with `npm install` in that directory.
- SQLite is supplied by `modernc.org/sqlite` (pure Go, no cgo).
- `github.com/ledongthuc/pdf` is the pure-Go PDF text extraction fallback.
- `mutool` is optional. If present on `PATH`, readerd prefers `mutool draw -F txt`; it is not bundled or required to build.

## Document formats

- EPUB opens the container, OPF manifest and spine, exposes navigation/XHTML chapters, and searches chapter text. DRM-protected EPUB is unsupported.
- PDF extraction is text-oriented; it does not render PDFs. The pure-Go extractor is the fallback when MuPDF is absent. Complex layouts, encrypted documents, and image-only/scanned pages may produce incomplete or no text. Install MuPDF and put `mutool` on `PATH` to use its text extractor.
- TXT and Markdown use UTF-8 text and provider pagination. Markdown is rendered as text/markup by the frontend, not by a browser document engine.
- MOBI, AZW3, FB2, CBZ and CBR are reserved for future provider adapters.

Fonts use host system sans-serif and serif families with language fallbacks. E-Ink styling removes shadows and gradients and requests reduced motion; physical display refresh policy is device-specific.
