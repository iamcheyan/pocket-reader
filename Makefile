.PHONY: dev backend frontend test

dev:
	@set -eu; export READERD_ADDR=127.0.0.1:8765 READERD_BOOKS="$${READERD_BOOKS:-$${HOME}/Books}" READERD_DATA="$${READERD_DATA:-$${HOME}/.local/share/pocket-reader}"; cd apps/pocket-reader/backend; go run . & backend=$$!; trap 'kill $$backend 2>/dev/null || true' EXIT INT TERM; cd ../frontend && npm run dev -- --host 127.0.0.1

backend:
	READERD_BOOKS="$${READERD_BOOKS:-$${HOME}/Books}" READERD_DATA="$${READERD_DATA:-$${HOME}/.local/share/pocket-reader}" go run ./cmd/readerd


frontend:
	cd apps/pocket-reader/frontend && npm run dev -- --host 127.0.0.1

test:
	cd apps/pocket-reader/backend && go test ./...
	cd apps/pocket-reader/frontend && npm run test:locales && npm run typecheck && npm run build
