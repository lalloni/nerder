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
image:
    podman build -t {{image}} -f Containerfile .

_cache:
    mkdir -p {{cache}}/home

# Compile the app in the container; the binary lands in build/bin/nerder
build: _cache
    {{ctr}} {{image}} wails build -tags {{tags}}

# All tests, in the container
test: _cache
    {{ctr}} {{image}} go test ./...

# Pure-Go packages need no GTK, so they can run on the host
test-internal:
    go test ./internal/...

# Shell inside the build container
shell: _cache
    {{ctr}} -it {{image}} bash

# Live-reloading UI served to a host browser at http://localhost:34115
dev: _cache
    {{ctr}} --init --name nerder-dev --network host {{image}} xvfb-run -a wails dev -tags {{tags}}

# Regenerate internal/catalog/stats.json (glyphs, downloads, stars, Google Fonts rank); needs network, and a `gh` login for rate limits
snapshot: _cache
    {{ctr}} -e GITHUB_TOKEN="$(gh auth token 2>/dev/null)" {{image}} go run ./tools/snapshot

# Install the desktop entry and icon in ~/.local/share so taskbars show the app icon (run after `just build`)
install-desktop:
    install -Dm644 build/appicon.png ~/.local/share/icons/hicolor/512x512/apps/nerder.png
    mkdir -p ~/.local/share/applications
    printf '%s\n' '[Desktop Entry]' 'Type=Application' 'Name=nerder' \
        'Comment=Browse and install Nerd Fonts' 'Exec={{justfile_directory()}}/build/bin/nerder' \
        'Icon=nerder' 'Terminal=false' 'Categories=Utility;Development;' 'StartupWMClass=nerder' \
        > ~/.local/share/applications/nerder.desktop
    -update-desktop-database ~/.local/share/applications
    -gtk-update-icon-cache -f -t ~/.local/share/icons/hicolor
