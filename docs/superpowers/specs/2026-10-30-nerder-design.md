# nerder: a Nerd Fonts browser and installer

## Intent and the why

The user lives in a terminal emulator and code editors, enjoys programming and command-line tools, and cares about code. They want to see good-looking text there. Nerd Fonts has ~72 font families, and choosing one means installing it, restarting things, and squinting. nerder lets them see every family rendered as it will look in an editor, judge it on real Go code, and install or uninstall it with one click so it is usable immediately.

The deciding criterion for a font is readability. The app itself should be calm, clean, simple and elegant, with the fonts as the main thing on screen.

## Goals

- Browse every Nerd Fonts family, each rendered in its real font (not a substitute), with ligatures and Nerd Font icon glyphs working as in an editor.
- Two switchable browsing modes:
  - **List:** one wide card per row with a large snippet.
  - **Grid:** several smaller cards across.
- Each card shows the same syntax-coloured Go snippet, so fonts compare fairly. The snippet is representative of most Go syntax: package and imports, constants/iota, structs with tags, interfaces, generics, methods and receivers, closures, goroutines and channels, `select`, `defer`, error handling, type switches, slices/maps, comments (line and block), string/rune/number literals. It includes ligature-prone operators (`!=`, `:=`, `<-`, `->`, `==`, `<=`, `>=`, `&&`, `||`, `...`) and a few Nerd Font icon glyphs.
- Install button per card. Clicking it:
  - installs the font into the user's fonts directory following Nerd Fonts' recommendation (`~/.local/share/fonts`),
  - takes every action needed so apps can use it right away (refresh the font cache with `fc-cache`),
  - turns the button into **Uninstall**.
- Uninstall removes the files nerder installed and refreshes the cache.
- A clear "Installed" indicator on each card.
- Installed state is derived from the filesystem, not only from nerder's own memory, so it stays correct after restarts and for fonts installed by hand.
- Finding fonts:
  - fuzzy search box (by family name),
  - "installed only" filter,
  - attribute filters, from the Nerd Fonts metadata: monospaced, license.
- Fonts are fetched on demand: a family is downloaded the first time its card scrolls into view and cached on disk. Install copies from the cache.

## Non-goals

- macOS and Windows support. Linux only.
- Installing system-wide (root) fonts.
- Editing or customising fonts, or generating patched fonts.
- Showing user-supplied text or snippets (the snippet is fixed, Go only).

## Anti-goals

- Cluttered or noisy UI. Anything that competes with the fonts for attention.
- Previews in a substitute font, or any rendering that drops ligatures or icons. That would defeat the purpose.
- Anything that leaves the font installed but not usable until a manual step.

## Constraints and decisions

Decided by the user:

- Go program with a nice UI, Fyne or Wails, whichever renders fonts best.
- Easy to distribute once built.
- Linux only for now.
- On-demand download with disk cache.
- Install location is the Nerd Fonts-recommended user fonts directory.
- **Wails**, confirmed by the user after comparing it with Gio, Fyne and a browser-served static binary. It renders fonts like an editor (ligatures, icon glyphs). The user accepts the cost: it depends on system WebKitGTK at build and run time, so the result is not a single self-contained binary on Linux. Packaging as an AppImage that bundles WebKitGTK is a possible later step, not part of this scope.

- The user runs KDE. Wails (WebKitGTK) will not follow their Qt theme, so the app needs its own coherent design instead of relying on native look. The user's plan: build this in Go and Wails first, and if it doesn't look good enough, try a more KDE-native version in Rust or Python (Qt/QML with Kirigami). Keep font-handling logic (download, cache, install, uninstall, installed detection) cleanly separated from the UI so a different UI could reuse the same design.

My calls (not agreed to by the user):

- Syntax highlighting done in the front end with a small Go tokenizer or a library such as Prism or highlight.js, whichever is smaller and works offline (the app should not need the network for anything except font downloads).
- Previews loaded into the web view via `@font-face` from the cached files.

More of my calls (settled after a builder review, not agreed to by the user):

- **Font list:** an embedded snapshot of `fonts.json`, with a pinned Nerd Fonts release version. The app needs the network only to download font archives. Updating the list means rebuilding.
- **What the preview uses:** the Regular file (plus Bold if present) from the standard patched variant, not Mono or Propo. Install copies every file in the family's archive, so the preview matches what gets installed.
- **Installed detection:** look in `~/.local/share/fonts` for files belonging to the family (matched by the family's archive file-name prefix, with a manifest written at install time as a cross-check). A family found only system-wide shows "Installed (system)" with no Uninstall button. Uninstall removes the family's files from the user fonts directory, including ones installed by hand.
- **"Usable immediately":** nerder runs `fc-cache` after install/uninstall. Some terminals and editors only reread fonts on restart, so the UI shows a short note after install saying so.
- **Filters:** monospaced (yes/no) and license (one value per font, from `licenseId`).

## Facts found while researching

- The Nerd Fonts repo publishes `bin/scripts/lib/fonts.json` (72 entries) with fields such as `unpatchedName`, `patchedName`, `folderName`, `licenseId`, `version`, `isMonospaced`, `description`, `imagePreviewFontSource`. It has no ligature field, so a ligatures filter is not possible from this source.
- Releases (currently v3.5.1) provide per-family archives, for example `0xProto.zip` and `0xProto.tar.xz`.
- On the user's machine at the time of writing: `go` is installed (snap), `fc-cache` is installed, `wails` is not installed, and WebKitGTK development files (`webkit2gtk-4.1`) were not found. The user will need to install build prerequisites (for example `libwebkit2gtk-4.1-dev`, `libgtk-3-dev`, a C compiler, and the Wails CLI).

## Left to the builder

- Project structure, package layout, and testing approach.
- How to resolve which release version to download (pin one, or read the latest), and how to verify downloads.
- Cache location (XDG cache dir is a sensible default).
- Exact visual design (typography, colours, light/dark), within "calm, clean, simple, elegant".
- Which files from an archive to install (Regular/Bold/Italic variants and the Mono/Propo variants), and how to track which files belong to a family so uninstall is exact.
- Fuzzy-search algorithm and the layout of the filter controls.
- Progress and error display for downloads and installs.
