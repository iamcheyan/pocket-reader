# Pocket Reader

A PJOS reference reading application built with PocketJS-compatible web UI (React + TypeScript + Vite), Go `readerd`, and SQLite. The frontend talks only to the readerd REST API; it never reads book files or SQLite directly.

## Run

Requirements: Go 1.24+, Node.js 20+, npm. From this repository root:

```sh
make dev
```

This starts readerd on `127.0.0.1:8765` and the Vite frontend on `127.0.0.1:5173`. Open http://127.0.0.1:5173. By default the service uses a local app data directory and displays 100 generated demonstration books when the configured library contains no supported books.

Set `READERD_BOOKS` to a library directory, `READERD_DATA` to a persistent data directory, or `READERD_ADDR` to change the listen address. Run `make test` for backend, frontend type, locale parity, and production bundle checks.

See [architecture](apps/pocket-reader/docs/ARCHITECTURE.md), [database](apps/pocket-reader/docs/DATABASE.md), [API](apps/pocket-reader/docs/API.md), [dependencies](apps/pocket-reader/docs/DEPENDENCIES.md), [roadmap](apps/pocket-reader/docs/ROADMAP.md), and [screenshots](apps/pocket-reader/docs/SCREENSHOTS.md).
