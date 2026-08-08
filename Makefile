.PHONY: build build-windows build-linux build-all dev test test-integration clean

VERSION ?= dev

# Current OS build (dev/test)
build:
	wails build -ldflags "-X main.version=$(VERSION)"

# Windows build (amd64) + NSIS installer
build-windows:
	wails build -platform windows/amd64 -nsis -ldflags "-X main.version=$(VERSION)"

# Linux build (amd64) - uses webkit2_41 tag (webkit2gtk-4.1, 4.0 dropped on modern Ubuntu)
build-linux:
	wails build -platform linux/amd64 -tags webkit2_41 -ldflags "-X main.version=$(VERSION)"

# Build both platforms
build-all: build-windows build-linux

# Development mode with hot-reload (Go + Vite)
dev:
	wails dev

test:
	go test ./...

test-integration:
	go test -tags=integration ./...

clean:
	rm -rf build frontend/dist frontend/src/lib/wailsjs
