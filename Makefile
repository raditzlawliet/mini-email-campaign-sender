.PHONY: build test test-integration dev dev-fe dev-be clean

BINARY := bin/server
FRONTEND_DIR := frontend
EMBED_DIR := cmd/server/frontend_dist

build: build-frontend copy-frontend
	go build -o $(BINARY) ./cmd/server

build-frontend:
	cd $(FRONTEND_DIR) && npm install && npm run build

copy-frontend:
	rm -rf $(EMBED_DIR)
	mkdir -p $(EMBED_DIR)
	cp -r $(FRONTEND_DIR)/dist/* $(EMBED_DIR)/

test:
	go test ./...

test-integration:
	go test -tags=integration ./...

# Production mode: single binary with embedded frontend
dev:
	go run ./cmd/server

# Backend only (API + CORS) — pair with dev-fe for hot-reload workflow
dev-be:
	DEV_MODE=true go run ./cmd/server

# Frontend dev server with hot-reload, proxies /api to backend
dev-fe:
	cd $(FRONTEND_DIR) && npm install && npm run dev

clean:
	rm -rf $(BINARY) $(FRONTEND_DIR)/dist $(EMBED_DIR) logs/
