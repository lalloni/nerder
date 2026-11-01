#!/bin/sh
# Install nerder for the current user: the binary in ~/.local/bin, the icon and a
# desktop entry under $XDG_DATA_HOME (default ~/.local/share). No root needed.
#
# Usage: install.sh [binary [icon]]    (default: nerder and nerder.png beside this script)
set -eu

here=$(cd "$(dirname "$0")" && pwd)
binary=${1:-$here/nerder}
icon=${2:-$here/nerder.png}
bindir=$HOME/.local/bin
data=${XDG_DATA_HOME:-$HOME/.local/share}

for f in "$binary" "$icon"; do
    [ -f "$f" ] || { echo "install.sh: $f not found" >&2; exit 1; }
done

install -Dm755 "$binary" "$bindir/nerder"
install -Dm644 "$icon" "$data/icons/hicolor/512x512/apps/nerder.png"
# Simplified faces for small sizes (panels, menus), when shipped beside the script.
for n in 16 22 32; do
    [ -f "$here/icons/$n.png" ] &&
        install -Dm644 "$here/icons/$n.png" "$data/icons/hicolor/${n}x${n}/apps/nerder.png"
done
mkdir -p "$data/applications"
cat > "$data/applications/nerder.desktop" <<EOF
[Desktop Entry]
Type=Application
Name=nerder
Comment=Browse and install Nerd Fonts
Exec=$bindir/nerder
Icon=nerder
Terminal=false
Categories=Utility;Development;
StartupWMClass=nerder
EOF

update-desktop-database "$data/applications" 2>/dev/null || true
gtk-update-icon-cache -f -t "$data/icons/hicolor" 2>/dev/null || true
# KDE keeps its own icon cache and ignores the GTK one.
command -v kbuildsycoca6 >/dev/null 2>&1 && kbuildsycoca6 >/dev/null 2>&1 || true

echo "Installed nerder to $bindir/nerder and added it to your application menu."
case ":$PATH:" in
    *":$bindir:"*) ;;
    *) echo "Note: $bindir is not on your PATH; start it from the menu or run $bindir/nerder." ;;
esac

ldconfig=$(command -v ldconfig 2>/dev/null || echo /sbin/ldconfig)
if [ -x "$ldconfig" ] && ! "$ldconfig" -p 2>/dev/null | grep -q 'libwebkit2gtk-4.1'; then
    echo "Warning: the WebKitGTK 4.1 library was not found. nerder needs it to run:" >&2
    echo "  Debian/Ubuntu: sudo apt install libwebkit2gtk-4.1-0" >&2
    echo "  Fedora:        sudo dnf install webkit2gtk4.1" >&2
    echo "  Arch:          sudo pacman -S webkit2gtk-4.1" >&2
fi
