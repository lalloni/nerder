#!/bin/sh
# nerder installer.
#
#   curl -fsSL https://raw.githubusercontent.com/lalloni/nerder/main/install.sh | sh
#
# Options (after `sh -s --` when piped):
#   --version vX.Y.Z   install that release instead of the latest
#   --user             Linux: install the tarball for this user only (no root)
#   --dry-run          show what would happen without changing anything
#   --help
#
# Linux:  .deb on Debian/Ubuntu (apt), .rpm on Fedora (dnf), otherwise the
#         tarball, installed for the current user.
# macOS:  Homebrew cask when brew is present, otherwise the app is copied to
#         /Applications (or ~/Applications).
#
# Downloads come from GitHub Releases and are checked against the release's
# SHA256SUMS. NERDER_VERSION and NERDER_BASE_URL override the version and the
# download location (used for testing).
set -eu

REPO=lalloni/nerder
BASE_URL=${NERDER_BASE_URL:-https://github.com/$REPO/releases/download}
VERSION=${NERDER_VERSION:-}
USER_ONLY=0
DRY_RUN=0
TMP=

say() { printf '%s\n' "$*"; }
warn() { printf 'warning: %s\n' "$*" >&2; }
die() { printf 'error: %s\n' "$*" >&2; exit 1; }
have() { command -v "$1" >/dev/null 2>&1; }

usage() {
    cat <<'EOF'
Usage: install.sh [--version vX.Y.Z] [--user] [--dry-run]

  --version vX.Y.Z   install that release instead of the latest
  --user             Linux: install the tarball for this user only (no root)
  --dry-run          show what would happen without changing anything
EOF
}

cleanup() { [ -n "$TMP" ] && rm -rf "$TMP"; }

# run prints the command in dry-run mode and runs it otherwise.
run() {
    if [ "$DRY_RUN" = 1 ]; then say "  would run: $*"; else "$@"; fi
}

as_root() {
    if [ "$(id -u)" = 0 ]; then run "$@"; else run sudo "$@"; fi
}

can_root() { [ "$(id -u)" = 0 ] || have sudo; }

latest_version() {
    have curl || die "curl is required"
    url=$(curl -fsSLI -o /dev/null -w '%{url_effective}' "https://github.com/$REPO/releases/latest") \
        || die "could not reach GitHub to find the latest release"
    case "$url" in
        */releases/tag/*) printf '%s' "${url##*/}" ;;
        *) die "no release found yet for $REPO" ;;
    esac
}

fetch() { curl -fsSL -o "$2" "$1" || die "download failed: $1"; }

sha256() {
    if have sha256sum; then sha256sum "$1" | cut -d' ' -f1; else shasum -a 256 "$1" | cut -d' ' -f1; fi
}

# download <asset>: fetch into $TMP and verify it against SHA256SUMS.
download() {
    asset=$1
    say "Downloading $asset"
    fetch "$BASE_URL/$VERSION/$asset" "$TMP/$asset"
    fetch "$BASE_URL/$VERSION/SHA256SUMS" "$TMP/SHA256SUMS"
    want=$(awk -v f="$asset" '$2 == f { print $1 }' "$TMP/SHA256SUMS")
    [ -n "$want" ] || die "$asset is not listed in SHA256SUMS"
    got=$(sha256 "$TMP/$asset")
    [ "$want" = "$got" ] || die "checksum mismatch for $asset (expected $want, got $got)"
    say "Checksum OK"
}

install_linux() {
    case "$(uname -m)" in
        x86_64 | amd64) arch=amd64; rpmarch=x86_64 ;;
        aarch64 | arm64) arch=arm64; rpmarch=aarch64 ;;
        *) die "unsupported architecture: $(uname -m)" ;;
    esac
    num=${VERSION#v}

    method=tarball
    if [ "$USER_ONLY" = 0 ]; then
        if have apt-get; then method=deb; elif have dnf; then method=rpm; fi
        if [ "$method" != tarball ] && ! can_root; then
            warn "root (or sudo) is needed for the $method package; installing for this user instead"
            method=tarball
        fi
    fi

    case $method in
        deb)
            asset="nerder_${num}_${arch}.deb"
            if [ "$DRY_RUN" = 1 ]; then say "Would download $asset and install it with apt-get"; else download "$asset"; fi
            as_root apt-get install -y "$TMP/$asset"
            ;;
        rpm)
            asset="nerder-${num}-1.${rpmarch}.rpm"
            if [ "$DRY_RUN" = 1 ]; then say "Would download $asset and install it with dnf"; else download "$asset"; fi
            as_root dnf install -y "$TMP/$asset"
            ;;
        tarball)
            asset="nerder-linux-${arch}.tar.gz"
            if [ "$DRY_RUN" = 1 ]; then
                say "Would download $asset and run its install.sh (installs to ~/.local, no root)"
            else
                download "$asset"
                tar -xzf "$TMP/$asset" -C "$TMP"
                sh "$TMP/nerder-linux-${arch}/install.sh"
            fi
            ;;
    esac
}

# find_brew prints the Homebrew binary: on PATH, or in its default prefixes
# (a piped shell often lacks /opt/homebrew/bin).
find_brew() {
    if have brew; then command -v brew; return 0; fi
    for b in /opt/homebrew/bin/brew /usr/local/bin/brew; do
        if [ -x "$b" ]; then printf '%s' "$b"; return 0; fi
    done
    return 1
}

install_macos() {
    # The cask always installs the latest release, so a pinned version skips it.
    if [ "$VERSION_PINNED" = 0 ] && brew=$(find_brew); then
        say "Installing with Homebrew"
        if run "$brew" install --cask lalloni/tap/nerder; then return 0; fi
        warn "Homebrew install failed; falling back to the downloaded app"
    fi

    asset=nerder-macos-universal.zip
    if [ -w /Applications ]; then dest=/Applications; else dest=$HOME/Applications; fi
    if [ "$DRY_RUN" = 1 ]; then say "Would download $asset and copy nerder.app to $dest"; return 0; fi
    download "$asset"
    have ditto || die "ditto is required to unpack the app"
    mkdir -p "$dest"
    ditto -x -k "$TMP/$asset" "$TMP/app"
    rm -rf "$dest/nerder.app"
    mv "$TMP/app/nerder.app" "$dest/nerder.app"
    # The app is not signed or notarized yet; clear the quarantine flag so it opens.
    xattr -dr com.apple.quarantine "$dest/nerder.app" 2>/dev/null || true
    say "Installed $dest/nerder.app (open it from Launchpad or Finder)"
}

main() {
    VERSION_PINNED=0
    [ -n "$VERSION" ] && VERSION_PINNED=1
    while [ $# -gt 0 ]; do
        case $1 in
            --version) [ $# -ge 2 ] || die "--version needs a value"; VERSION=$2; VERSION_PINNED=1; shift ;;
            --user) USER_ONLY=1 ;;
            --dry-run) DRY_RUN=1 ;;
            -h | --help) usage; exit 0 ;;
            *) die "unknown option: $1 (try --help)" ;;
        esac
        shift
    done

    have curl || die "curl is required"
    [ -n "$VERSION" ] || VERSION=$(latest_version)
    case $VERSION in v*) ;; *) VERSION=v$VERSION ;; esac
    TMP=$(mktemp -d)
    chmod 755 "$TMP" # apt's download user must be able to read the package
    trap cleanup EXIT INT TERM

    say "nerder $VERSION"
    case "$(uname -s)" in
        Linux) install_linux ;;
        Darwin) install_macos ;;
        *) die "unsupported system: $(uname -s) (nerder supports Linux and macOS)" ;;
    esac
    [ "$DRY_RUN" = 1 ] || say "Done."
}

main "$@"
