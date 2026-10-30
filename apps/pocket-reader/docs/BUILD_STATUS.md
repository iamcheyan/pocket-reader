# Build and verification status

## Verified

- Frontend: `npm run test:locales`, `npm run typecheck` and `npm run build` pass.
- Backend: code compiles against the module graph except for the missing `github.com/ledongthuc/pdf` module.

## Blocked

- `proxy.golang.org` and `registry.npmjs.org` are unreachable from this host; the required Go module is not cached locally, so `go test` and `go build` cannot complete here.
- `mutool` is not installed on `PATH`, so the preferred MuPDF text extraction path is unavailable; the pure-Go PDF extractor is the configured fallback.

## Workarounds

- Frontend dependencies were satisfied by reusing an existing local `node_modules` tree.
- To finish the backend, install the PDF module from a network-enabled host or add it to the Go module cache, then run `go mod tidy` and `make test`.