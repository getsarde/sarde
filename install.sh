#!/bin/sh
set -e

REPO="getsarde/sarde"
BINARY="sarde"

# Must match the archive name_template in .goreleaser.yaml, which is
# {{ .ProjectName }}_{{ .Os }}_{{ .Arch }} using Go's lowercase GOOS/GOARCH
# names. Any drift here produces a 404 at download time.
detect_platform() {
  OS=$(uname -s)
  ARCH=$(uname -m)

  case "$OS" in
    Linux)  OS="linux" ;;
    Darwin) OS="darwin" ;;
    *)
      echo "Error: unsupported OS: $OS" >&2
      exit 1
      ;;
  esac

  case "$ARCH" in
    x86_64|amd64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *)
      echo "Error: unsupported architecture: $ARCH" >&2
      exit 1
      ;;
  esac

  # windows/arm64 is the only combination goreleaser excludes, and this script
  # never runs there (the OS check above already rejected it).
  echo "${OS}_${ARCH}"
}

get_latest_version() {
  curl -sSf "https://api.github.com/repos/${REPO}/releases/latest" |
    grep '"tag_name"' |
    sed -E 's/.*"tag_name": *"([^"]+)".*/\1/'
}

main() {
  PLATFORM=$(detect_platform)
  VERSION=$(get_latest_version)

  if [ -z "$VERSION" ]; then
    echo "Error: could not determine latest version" >&2
    exit 1
  fi

  VERSION_NUM="${VERSION#v}"
  ARCHIVE="${BINARY}_${PLATFORM}.tar.gz"
  URL="https://github.com/${REPO}/releases/download/${VERSION}/${ARCHIVE}"
  CHECKSUMS_URL="https://github.com/${REPO}/releases/download/${VERSION}/checksums.txt"

  TMPDIR=$(mktemp -d)
  trap 'rm -rf "$TMPDIR"' EXIT

  echo "Downloading ${BINARY} ${VERSION} for ${PLATFORM}..."
  curl -sSfL -o "${TMPDIR}/${ARCHIVE}" "$URL"
  curl -sSfL -o "${TMPDIR}/checksums.txt" "$CHECKSUMS_URL"

  echo "Verifying checksum..."
  cd "$TMPDIR"
  if command -v sha256sum > /dev/null 2>&1; then
    grep "$ARCHIVE" checksums.txt | sha256sum -c --quiet
  elif command -v shasum > /dev/null 2>&1; then
    grep "$ARCHIVE" checksums.txt | shasum -a 256 -c --quiet
  else
    echo "Warning: no sha256 tool found, skipping checksum verification" >&2
  fi

  tar xzf "$ARCHIVE"

  INSTALL_DIR="/usr/local/bin"
  if [ ! -w "$INSTALL_DIR" ]; then
    INSTALL_DIR="${HOME}/.local/bin"
    mkdir -p "$INSTALL_DIR"
  fi

  mv "${BINARY}" "${INSTALL_DIR}/${BINARY}"
  chmod +x "${INSTALL_DIR}/${BINARY}"

  echo "Installed ${BINARY} ${VERSION} to ${INSTALL_DIR}/${BINARY}"

  if ! echo "$PATH" | tr ':' '\n' | grep -qx "$INSTALL_DIR"; then
    echo ""
    echo "Add ${INSTALL_DIR} to your PATH:"
    echo "  export PATH=\"${INSTALL_DIR}:\$PATH\""
  fi
}

main
