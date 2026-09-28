#!/bin/sh
set -eu

# Install the clover2 CLI from GitHub releases.
#
#   curl -fsSL https://github.com/klever-coex/clover2-cli/releases/latest/download/install.sh | sh
#
# Environment variables:
#   VERSION       release to install: "latest" (default), "1.2.3" or "v1.2.3"
#   REPO          github repository           (default: klever-coex/clover2-cli)
#   INSTALL_DIR   target directory            (default: /usr/local/bin)
#   BIN_NAME      installed binary name       (default: clover2)
#   GITHUB_TOKEN  access token, needed for private repositories
#   GITHUB_URL    github base url             (default: https://github.com)

REPO=${REPO:-klever-coex/clover2-cli}
VERSION=${VERSION:-latest}
INSTALL_DIR=${INSTALL_DIR:-/usr/local/bin}
BIN_NAME=${BIN_NAME:-clover2}
GITHUB_URL=${GITHUB_URL:-https://github.com}

info() {
    echo '[INFO] ' "$@"
}

fatal() {
    echo '[ERROR]' "$@" >&2
    exit 1
}

TMP_DIR=$(mktemp -d -t clover2-install.XXXXXXXXXX)
trap 'rm -rf "$TMP_DIR"' EXIT INT TERM

# --- platform --------------------------------------------------------------

case "$(uname -s)" in
    Linux) OS=linux ;;
    *) fatal "unsupported OS: $(uname -s), releases are linux-only for now" ;;
esac
case "$(uname -m)" in
    x86_64 | amd64) ARCH=amd64 ;;
    aarch64 | arm64) ARCH=arm64 ;;
    *) fatal "unsupported architecture: $(uname -m)" ;;
esac
ARTIFACT="clover2-cli-${OS}-${ARCH}"

case "$VERSION" in
    latest | "")
        BASE="$GITHUB_URL/$REPO/releases/latest/download"
        info "Installing latest release for $OS/$ARCH"
        ;;
    v*)
        BASE="$GITHUB_URL/$REPO/releases/download/$VERSION"
        info "Installing $VERSION for $OS/$ARCH"
        ;;
    *)
        BASE="$GITHUB_URL/$REPO/releases/download/v$VERSION"
        info "Installing v$VERSION for $OS/$ARCH"
        ;;
esac

# --- downloader ------------------------------------------------------------

DOWNLOADER=""
if command -v curl >/dev/null 2>&1; then
    DOWNLOADER=curl
elif command -v wget >/dev/null 2>&1; then
    DOWNLOADER=wget
else
    fatal "curl or wget is required"
fi

fetch() {
    # fetch <destination> <url>
    _dst=$1
    _url=$2
    if [ "$DOWNLOADER" = "curl" ]; then
        if [ -n "${GITHUB_TOKEN:-}" ]; then
            curl -fsSL --retry 3 -H "Authorization: Bearer $GITHUB_TOKEN" -o "$_dst" "$_url"
        else
            curl -fsSL --retry 3 -o "$_dst" "$_url"
        fi
    else
        if [ -n "${GITHUB_TOKEN:-}" ]; then
            wget -q --header "Authorization: Bearer $GITHUB_TOKEN" -O "$_dst" "$_url"
        else
            wget -q -O "$_dst" "$_url"
        fi
    fi
}

# --- download and verify ---------------------------------------------------

fetch "$TMP_DIR/$BIN_NAME" "$BASE/$ARTIFACT" ||
    fatal "failed to download $ARTIFACT from $BASE"
fetch "$TMP_DIR/checksums.txt" "$BASE/checksums.txt" ||
    fatal "failed to download checksums.txt from $BASE"

EXPECTED=$(awk -v n="$ARTIFACT" '$2 == n { print $1; exit }' "$TMP_DIR/checksums.txt")
[ -n "$EXPECTED" ] || fatal "$ARTIFACT not found in checksums.txt"

if command -v sha256sum >/dev/null 2>&1; then
    ACTUAL=$(sha256sum "$TMP_DIR/$BIN_NAME" | awk '{ print $1 }')
elif command -v shasum >/dev/null 2>&1; then
    ACTUAL=$(shasum -a 256 "$TMP_DIR/$BIN_NAME" | awk '{ print $1 }')
else
    fatal "sha256sum or shasum is required to verify the checksum"
fi

[ "$ACTUAL" = "$EXPECTED" ] ||
    fatal "checksum mismatch for $ARTIFACT: expected $EXPECTED, got $ACTUAL"
info "Checksum OK: $ACTUAL"

chmod 0755 "$TMP_DIR/$BIN_NAME"
"$TMP_DIR/$BIN_NAME" --version >/dev/null 2>&1 ||
    fatal "downloaded binary failed to run on this platform"

# --- install ---------------------------------------------------------------

if [ "$(id -u)" = "0" ]; then
    SUDO=""
elif [ -w "$INSTALL_DIR" ] || [ -w "$(dirname "$INSTALL_DIR")" ]; then
    SUDO=""
elif command -v sudo >/dev/null 2>&1; then
    SUDO="sudo"
else
    fatal "cannot write to $INSTALL_DIR: run as root or install sudo"
fi

$SUDO mkdir -p "$INSTALL_DIR"
$SUDO install -m 0755 "$TMP_DIR/$BIN_NAME" "$INSTALL_DIR/$BIN_NAME"
info "Installed $INSTALL_DIR/$BIN_NAME"

"$INSTALL_DIR/$BIN_NAME" --version
info "Done"
