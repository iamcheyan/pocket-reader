# readerd
Run from this directory with `go run .`; from repository root use `make backend`.
- `READERD_ADDR` (default `127.0.0.1:8765`)
- `READERD_DATA` (default `~/.config/pocket-reader`)
- `READERD_DB` (optional database path override; default `readerd.sqlite` in the data directory)
- `READERD_BOOKS` (default `~/Books`)

The service creates the data and library directories at startup.
