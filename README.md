# Pocket Reader

A PJOS reference reading application built with a PocketJS-compatible web UI (React + TypeScript + Vite), the Go `readerd` service, and SQLite. The frontend talks only to the readerd REST API; it never reads book files, parses EPUB, or touches SQLite directly.

## Run

Requirements: Go 1.24+, Node.js 20+, npm.

```sh
cd apps/pocket-reader/frontend && npm install && cd ../../..
make dev
```

`make dev` starts readerd on `127.0.0.1:8765` and Vite on `127.0.0.1:5173`. Open http://127.0.0.1:5173.

- `READERD_DATA` — persistent data directory (default `~/.config/pocket-reader`; SQLite lives there).
- `READERD_BOOKS` — library directory (default `~/Books`), scanned recursively at startup.
- `READERD_ADDR` — listen address (default `127.0.0.1:8765`).
- `READERD_URL` — Vite proxy target for `/api` (default `http://127.0.0.1:8765`).

When the library has no supported files, readerd seeds 100 generated demonstration books so every screen and flow is usable. Scanned EPUB/PDF/TXT/Markdown files replace that demo catalog; the index is incremental on mtime+size.

## Features

- EPUB (OPF spine, navigation, XHTML chapters), PDF text extraction, TXT and Markdown.
- Library, book details, reader with TOC, in-book search, bookmarks, colored highlights, notes, note export (md/json).
- Reading progress persisted with chapter/location/percentage and resumed after restart.
- Light / Dark / Sepia / E-Ink themes; E-Ink removes shadows, gradients, rounded corners and transitions and renders grayscale.
- Responsive layouts verified at 320×240, 480×320, 720×720, 1280×720 and 1920×1080, at DPR 1/1.5/2/3.
- Keyboard (arrows, Page Up/Down, B bookmark, H highlight, N note, Esc) and gamepad paging; touch swipe and tap zones.
- English, 简体中文, 日本語 with a persistent locale and parity-checked key sets.
- OPDS Atom catalog endpoint; RSS/Calibre integration points documented as reserved.

## Test

```sh
make test
```

Runs Go tests (providers, scan, resume, annotations, export round-trip, OPDS), locale parity, TypeScript checking and the production bundle build.

Docs: [architecture](apps/pocket-reader/docs/ARCHITECTURE.md), [database](apps/pocket-reader/docs/DATABASE.md), [API](apps/pocket-reader/docs/API.md), [dependencies](apps/pocket-reader/docs/DEPENDENCIES.md), [roadmap](apps/pocket-reader/docs/ROADMAP.md), [screenshots](apps/pocket-reader/docs/SCREENSHOTS.md), [build status](apps/pocket-reader/docs/BUILD_STATUS.md).
