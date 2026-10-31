# nerder

Browse every [Nerd Fonts](https://www.nerdfonts.com) family rendered in its real
font on a syntax-coloured Go snippet, then install or uninstall it with one
click. Fonts go to `~/.local/share/fonts/NerdFonts/<Family>/` (or
`$XDG_DATA_HOME/fonts`), and `fc-cache` is run so apps can use them right away.
Some terminals and editors still need a restart to pick up a new font.

Linux only. Built with Go and [Wails](https://wails.io) v2 (WebKitGTK).

## Using it

```
just build        # compile in the container -> build/bin/nerder
./build/bin/nerder
```

Running the binary needs only the WebKitGTK runtime library
(`libwebkit2gtk-4.1-0` on Debian/Ubuntu). Families are downloaded the first
time their card scrolls into view and cached under `~/.cache/nerder/`.

## Developing

Compiling happens in a container, so the host needs no GTK or WebKitGTK
development packages. You need [Podman](https://podman.io) and
[just](https://github.com/casey/just).

```
just image          # build the container image (once)
just build          # wails build in the container
just test           # go test ./... in the container
just test-internal  # pure-Go packages only, on the host
just dev            # live-reloading UI at http://localhost:34115
just shell          # shell inside the build container
```

`just dev` runs `wails dev` under Xvfb in the container; open
http://localhost:34115 in a browser. Installs made there go to the container's
throwaway home (`~/.cache/nerder-build/home`), not your real fonts folder.

Archive availability can be checked against GitHub with
`go test -tags live ./internal/fetch/ -run Live`.

## Layout

- `internal/catalog` embedded font list, fuzzy search, file-to-family matching
- `internal/snippet` the Go sample and its highlighter
- `internal/fetch` download, extract and cache font archives
- `internal/installer` install, uninstall, installed-state detection
- `internal/app` the methods bound to the front end, plus the font file handler
- `frontend/` plain JS/CSS UI

## Updating the Nerd Fonts version

Replace `internal/catalog/fonts.json` with the file from the new release tag,
change `ReleaseTag` in `internal/catalog/catalog.go`, and update the expected
family count in the catalog tests.
