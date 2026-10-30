# Verification — pocket-reader @ 2026-10-31

- Verdict: PASS
- Commit: `__COMMIT__` (fixes + screenshots; this document is the following commit)
- Remote: `__REMOTE__`
- Runtime: `readerd` on `127.0.0.1:8791` (fresh temp data, generated demo catalogue) + Vite `127.0.0.1:5199`; production bundle from `make test`
- Toolchain: go 1.26.0, node v22.23.3, Chrome 155.0.8059.84

## Test gate
| command | result | notes |
|---|---|---|
| `go test ./...` | PASS | 14 Go tests (providers EPUB/PDF/TXT/Markdown, scan, resume, annotations, export round-trip, OPDS), 0 failures |
| `npm run test:locales` | PASS | locale key parity |
| `npm run typecheck` | PASS | `tsc --noEmit` |
| `npm run build` | PASS | production Vite bundle |

`make test` runs all four gates; green.

## Runtime
`GET /api/health` → `{"version":"0.1.0","mock":true,"engines":{"epub":true,"pdf":"pure-go","text":true}}`.
The library serves 20 generated demo books (5 each EPUB/PDF/TXT/Markdown); EPUB id 1 and PDF
id 2 were opened through the real reader view, with chapter content, TOC, bookmarks and notes
driven through the app's deep-link state.

## Screenshot matrix
All 21 files were regenerated this run by the new repo script
`apps/pocket-reader/tests/capture-screenshots.mjs` (puppeteer-core + system Chrome) against the
live stack, using the app's deterministic `?screen=…&book=…&theme=…&lang=…&q=…` parameters.
`docs/screenshots/manifest.json` records the run timestamp and every SHA-256.

| screen | viewport | file | note |
|---|---|---|---|
| library-1280x720.png | 1280×720 | `apps/pocket-reader/docs/screenshots/library-1280x720.png` | screen=library&theme=light&lang=en |
| library-1920x1080.png | 1920×1080 | `apps/pocket-reader/docs/screenshots/library-1920x1080.png` | screen=library&theme=light&lang=en |
| library-720x720.png | 720×720 | `apps/pocket-reader/docs/screenshots/library-720x720.png` | screen=library&theme=light&lang=en |
| library-480x320.png | 480×320 | `apps/pocket-reader/docs/screenshots/library-480x320.png` | screen=library&theme=light&lang=en |
| library-320x240.png | 320×240 | `apps/pocket-reader/docs/screenshots/library-320x240.png` | screen=library&theme=light&lang=en |
| details-1280x720.png | 1280×720 | `apps/pocket-reader/docs/screenshots/details-1280x720.png` | book=1&theme=light&lang=en |
| reader-epub-1280x720.png | 1280×720 | `apps/pocket-reader/docs/screenshots/reader-epub-1280x720.png` | screen=reader&book=1&theme=light&lang=en |
| reader-pdf-1280x720.png | 1280×720 | `apps/pocket-reader/docs/screenshots/reader-pdf-1280x720.png` | screen=reader&book=2&theme=light&lang=en |
| reader-320x240.png | 320×240 | `apps/pocket-reader/docs/screenshots/reader-320x240.png` | screen=reader&book=1&theme=light&lang=en |
| bookmarks-1280x720.png | 1280×720 | `apps/pocket-reader/docs/screenshots/bookmarks-1280x720.png` | screen=bookmarks&theme=light&lang=en |
| notes-1280x720.png | 1280×720 | `apps/pocket-reader/docs/screenshots/notes-1280x720.png` | screen=notes&theme=light&lang=en |
| search-1280x720.png | 1280×720 | `apps/pocket-reader/docs/screenshots/search-1280x720.png` | screen=search&q=Volume&theme=light&lang=en |
| settings-1280x720.png | 1280×720 | `apps/pocket-reader/docs/screenshots/settings-1280x720.png` | screen=settings&theme=light&lang=en |
| settings-eink-1280x720.png | 1280×720 | `apps/pocket-reader/docs/screenshots/settings-eink-1280x720.png` | screen=settings&theme=eink&lang=en |
| dark-1280x720.png | 1280×720 | `apps/pocket-reader/docs/screenshots/dark-1280x720.png` | screen=library&theme=dark&lang=en |
| eink-1280x720.png | 1280×720 | `apps/pocket-reader/docs/screenshots/eink-1280x720.png` | screen=library&theme=eink&lang=en |
| library-zh-1280x720.png | 1280×720 | `apps/pocket-reader/docs/screenshots/library-zh-1280x720.png` | screen=library&theme=light&lang=zh |
| library-ja-1280x720.png | 1280×720 | `apps/pocket-reader/docs/screenshots/library-ja-1280x720.png` | screen=library&theme=light&lang=ja |
| dpi-1.5x-720x720.png | 1080×1080 | `apps/pocket-reader/docs/screenshots/dpi-1.5x-720x720.png` | screen=library&theme=light&lang=en, dsf 1.5 |
| dpi-2x-720x720.png | 1440×1440 | `apps/pocket-reader/docs/screenshots/dpi-2x-720x720.png` | screen=library&theme=light&lang=en, dsf 2 |
| dpi-3x-720x720.png | 2160×2160 | `apps/pocket-reader/docs/screenshots/dpi-3x-720x720.png` | screen=library&theme=light&lang=en, dsf 3 |

Count: 21 screenshots, 21 unique hashes (0 duplicates). `tools/check_shots.py`: all PNGs valid,
dimensions match (including 1080×1080 / 1440×1440 / 2160×2160 for DPR 1.5/2/3), minimum
per-image RGB stddev above the blank threshold, no blank frames. The script fails on horizontal
overflow, short documents, duplicate hashes, console errors and HTTP ≥400; this run passed.

## Fixed
- The 21 committed images had no capture script (only an ad-hoc `chromium --screenshot` recipe
  that cannot drive the reader/annotation flows). → added
  `apps/pocket-reader/tests/capture-screenshots.mjs` + `make screenshots`, with deep-link driven
  states, hash/size/overflow/console gates, and a manifest → `apps/pocket-reader/tests/`,
  `Makefile`.
- `dpi-1x-720x720.png` was a byte-for-byte duplicate of `library-720x720.png` (same URL, DPR 1),
  which violates the no-duplicate-image rule. → dropped the redundant 1× file (the docs now
  state that 1× is identical to `library-720x720.png`), keeping DPR 1.5/2/3 as distinct
  captures → `docs/SCREENSHOTS.md` and the screenshot set → this run reports 21/21 unique.
- Every page logged a console 404 for `/favicon.ico`. → added
  `apps/pocket-reader/frontend/public/favicon.svg` and `<link rel="icon">` in `index.html`; the
  capture now finishes with "no console errors".

## Blocked / open questions
- OPDS/RSS/Calibre integrations remain reserved (documented); not exercised.
- The demo catalogue is generated, not scanned from real EPUB/PDF files on a device.

## Not verified here
- 掌机真机（RG34XX / PJOS-LINUX）未验证；本轮只在本机 headless Chrome 里跑真实进程。
- Real e-ink ghosting/refresh behaviour; only the CSS e-ink theme was rendered.
