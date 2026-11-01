image := "nerder-build"
cache := env("HOME") / ".cache/nerder-build"
tags  := "webkit2_41"

# Run a command in the build container as the current user.
ctr := "podman run --rm --userns=keep-id" \
    + " -v " + justfile_directory() + ":/src:Z -v " + cache + ":/cache:Z -w /src" \
    + " -e HOME=/cache/home -e GOPATH=/cache/go -e GOCACHE=/cache/gobuild" \
    + " -e npm_config_cache=/cache/npm"

default:
    @just --list

# Build the container image with the GTK/WebKitGTK dev packages, Go, Node and Wails
[linux]
image:
    podman build -t {{image}} -f Containerfile .

_cache:
    mkdir -p {{cache}}/home

# Compile the app in the container; the binary lands in build/bin/nerder
[linux]
build: _cache
    {{ctr}} {{image}} wails build -tags {{tags}}

# All tests, in the container
[linux]
test: _cache
    {{ctr}} {{image}} go test ./...

# macOS builds natively (no container); a universal app for Intel and Apple Silicon lands in build/bin/nerder.app
[macos]
build:
    wails build -platform darwin/universal

# Pure-Go packages, natively
[macos]
test:
    go test ./internal/... ./tools/...

# Pure-Go packages need no GTK, so they can run on the host
test-internal:
    go test ./internal/...

# Shell inside the build container
[linux]
shell: _cache
    {{ctr}} -it {{image}} bash

# Live-reloading UI served to a host browser at http://localhost:34115
[linux]
dev: _cache
    {{ctr}} --init --name nerder-dev --network host {{image}} xvfb-run -a wails dev -tags {{tags}}

# Regenerate internal/catalog/stats.json (glyphs, downloads, stars, Google Fonts rank); needs network, and a `gh` login for rate limits
[linux]
snapshot: _cache
    {{ctr}} -e GITHUB_TOKEN="$(gh auth token 2>/dev/null)" {{image}} go run ./tools/snapshot

# Same, natively on macOS
[macos]
snapshot:
    GITHUB_TOKEN="$(gh auth token 2>/dev/null)" go run ./tools/snapshot

# Build, then install the app for the current user (binary in ~/.local/bin, icon and desktop entry in ~/.local/share)
[linux]
desktop-install: build
    sh packaging/linux/install.sh build/bin/nerder build/appicon.png

# Remove what desktop-install added
[linux]
desktop-uninstall:
    sh packaging/linux/uninstall.sh
