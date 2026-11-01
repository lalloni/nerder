#!/bin/sh
# Remove what install.sh added. Fonts you installed with nerder are left alone.
set -eu

data=${XDG_DATA_HOME:-$HOME/.local/share}

rm -f "$HOME/.local/bin/nerder" \
    "$data/applications/nerder.desktop" \
    "$data/icons/hicolor/512x512/apps/nerder.png"

update-desktop-database "$data/applications" 2>/dev/null || true
gtk-update-icon-cache -f -t "$data/icons/hicolor" 2>/dev/null || true

echo "Removed nerder from $HOME/.local/bin and your application menu."
