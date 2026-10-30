.PHONY: dev backend frontend test screenshots

dev:
	@set -eu; export READERD_ADDR=127.0.0.1:8765 READERD_BOOKS="$${READERD_BOOKS:-$${HOME}/Books}" READERD_DATA="$${READERD_DATA:-$${HOME}/.config/pocket-reader}"; cd apps/pocket-reader/backend; go run ./cmd/readerd & backend=$$!; trap 'kill $$backend 2>/dev/null || true' EXIT INT TERM; cd ../frontend && npm run dev -- --host 127.0.0.1

backend:
	READERD_BOOKS="$${READERD_BOOKS:-$${HOME}/Books}" READERD_DATA="$${READERD_DATA:-$${HOME}/.local/share/pocket-reader}" go run ./cmd/readerd


frontend:
	cd apps/pocket-reader/frontend && npm run dev -- --host 127.0.0.1

screenshots:
	CHROME_BIN=$${CHROME_BIN:-$$(command -v google-chrome || command -v chromium)} \
	PUPPETEER_CORE=$${PUPPETEER_CORE:-$$HOME/.bun/install/global/node_modules/puppeteer-core/lib/puppeteer/puppeteer-core.js} \
	node apps/pocket-reader/tests/capture-screenshots.mjs

test:
	cd apps/pocket-reader/backend && go test ./...
	cd apps/pocket-reader/frontend && npm run test:locales && npm run typecheck && npm run build
