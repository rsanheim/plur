#!/bin/sh
# Universal installation script for plur
# Usage:
#   curl -sSL https://raw.githubusercontent.com/rsanheim/plur/main/install.sh | sh
#   curl -sSL https://raw.githubusercontent.com/rsanheim/plur/main/install.sh | sh -s -- --version v1.0.0
#   curl -sSL https://raw.githubusercontent.com/rsanheim/plur/main/install.sh | sh -s -- --install-path ~/.local/bin --version v1.0.0
#
# Environment variables:
#   PLUR_INSTALL_PATH - Installation directory (default: ~/.local/bin)

set -eu

# Configuration
REPO_OWNER="rsanheim"
REPO_NAME="plur"
INSTALL_PATH="${PLUR_INSTALL_PATH:-$HOME/.local/bin}"
VERSION=""
DOWNLOAD_RETRIES=3
DOWNLOAD_CONNECT_TIMEOUT=10

# Helper functions
info() {
  printf ">>> %s\n" "$1"
}

warn() {
  printf "Warning: %s\n" "$1"
}

error() {
  printf "Error: %s\n" "$1"
  exit 1
}

show_help() {
  cat << EOF
plur installer

Usage:
  sh install.sh [options]
  curl -sSL https://raw.githubusercontent.com/rsanheim/plur/main/install.sh | sh
  curl -sSL https://raw.githubusercontent.com/rsanheim/plur/main/install.sh | sh -s -- [options]

Options:
  -h, --help             Show this help text
  --install-path PATH    Installation directory (default: \$PLUR_INSTALL_PATH or ~/.local/bin)
  --version VERSION      Version tag to install (for example: v0.5.0)
EOF
}

parse_args() {
  while [ $# -gt 0 ]; do
    case "$1" in
      -h|--help)
        show_help
        exit 0
        ;;
      --install-path)
        [ $# -ge 2 ] || error "--install-path requires a value"
        INSTALL_PATH="$2"
        shift 2
        ;;
      --install-path=*)
        INSTALL_PATH="${1#*=}"
        shift
        ;;
      --version)
        [ $# -ge 2 ] || error "--version requires a value"
        VERSION="$2"
        shift 2
        ;;
      --version=*)
        VERSION="${1#*=}"
        shift
        ;;
      --)
        error "Unexpected '--' (run with --help)"
        ;;
      -*)
        error "Unknown option: $1 (run with --help)"
        ;;
      *)
        error "Unexpected argument: $1 (use --version)"
        ;;
    esac
  done
}

require_command() {
  command -v "$1" >/dev/null 2>&1 || error "Required command not found: $1"
}

has_download_tool() {
  command -v curl >/dev/null 2>&1 || command -v wget >/dev/null 2>&1
}

check_prerequisites() {
  require_command uname
  require_command tr
  require_command sed
  require_command tar
  require_command mktemp
  require_command find
  require_command chmod
  require_command awk

  has_download_tool || error "Neither curl nor wget found. Please install one of them."
}

download_to_file() {
  URL="$1"
  OUTPUT_PATH="$2"

  if command -v curl >/dev/null 2>&1; then
    curl -fL \
      --retry "$DOWNLOAD_RETRIES" \
      --retry-delay 1 \
      --connect-timeout "$DOWNLOAD_CONNECT_TIMEOUT" \
      -sS \
      "$URL" \
      -o "$OUTPUT_PATH" || error "Failed to download: $URL"
  elif command -v wget >/dev/null 2>&1; then
    # Keep wget flags minimal for BusyBox compatibility (Alpine/minimal images).
    wget -q "$URL" -O "$OUTPUT_PATH" || error "Failed to download: $URL"
  else
    error "Neither curl nor wget found. Please install one of them."
  fi
}

download_to_stdout() {
  URL="$1"

  if command -v curl >/dev/null 2>&1; then
    curl -fL \
      --retry "$DOWNLOAD_RETRIES" \
      --retry-delay 1 \
      --connect-timeout "$DOWNLOAD_CONNECT_TIMEOUT" \
      -sS \
      "$URL" || error "Failed to fetch: $URL"
  elif command -v wget >/dev/null 2>&1; then
    wget -q -O- "$URL" || error "Failed to fetch: $URL"
  else
    error "Neither curl nor wget found. Please install one of them."
  fi
}

sha256_file() {
  FILE_PATH="$1"

  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$FILE_PATH" | awk '{ print $1 }'
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$FILE_PATH" | awk '{ print $1 }'
  elif command -v openssl >/dev/null 2>&1; then
    openssl dgst -sha256 "$FILE_PATH" | awk '{ print $2 }'
  else
    error "No checksum tool found (need sha256sum, shasum, or openssl)"
  fi
}

verify_checksum() {
  ARCHIVE_PATH="$1"
  CHECKSUM_FILE_PATH="$2"
  ARCHIVE_NAME="$3"

  EXPECTED_CHECKSUM=$(awk -v filename="$ARCHIVE_NAME" '
    $2 == filename { print $1; exit }
    $2 == "*" filename { print $1; exit }
  ' "$CHECKSUM_FILE_PATH")

  [ -n "$EXPECTED_CHECKSUM" ] || error "Checksum entry not found for $ARCHIVE_NAME"

  ACTUAL_CHECKSUM=$(sha256_file "$ARCHIVE_PATH")

  if [ "$ACTUAL_CHECKSUM" != "$EXPECTED_CHECKSUM" ]; then
    error "Checksum mismatch for $ARCHIVE_NAME"
  fi

  info "Checksum verified for $ARCHIVE_NAME"
}

# Detect architecture
detect_arch() {
  ARCH=$(uname -m)
  case $ARCH in
    x86_64)
      echo "amd64"
      ;;
    aarch64|arm64)
      echo "arm64"
      ;;
    *)
      error "Unsupported architecture: $ARCH"
      ;;
  esac
}

# Detect OS
detect_os() {
  OS=$(uname -s | tr '[:upper:]' '[:lower:]')
  case $OS in
    linux)
      echo "linux"
      ;;
    darwin)
      echo "darwin"
      ;;
    *)
      error "Unsupported OS: $OS"
      ;;
  esac
}

# Get latest version from GitHub
get_latest_version() {
  RELEASES_JSON=$(download_to_stdout "https://api.github.com/repos/$REPO_OWNER/$REPO_NAME/releases/latest")
  LATEST=$(printf "%s\n" "$RELEASES_JSON" | awk -F '"' '/"tag_name":[[:space:]]*/ { print $4; exit }')

  if [ -z "$LATEST" ]; then
    error "Failed to fetch latest version from GitHub releases API"
  fi

  echo "$LATEST"
}

# Download and install plur
install_plur() {
  # Detect system info
  OS=$(detect_os)
  ARCH=$(detect_arch)

  info "Detected: $OS/$ARCH"

  # Determine version
  if [ -z "$VERSION" ]; then
    info "Fetching latest version..."
    VERSION=$(get_latest_version)
  fi

  # Remove 'v' prefix if present for the filename
  VERSION_NUM=$(echo "$VERSION" | sed 's/^v//')

  info "Installing plur $VERSION"

  # Construct download URL
  # Format: plur_VERSION_OS_ARCH.tar.gz
  FILENAME="plur_${VERSION_NUM}_${OS}_${ARCH}.tar.gz"
  BASE_URL="https://github.com/$REPO_OWNER/$REPO_NAME/releases/download/$VERSION"
  URL="$BASE_URL/$FILENAME"
  CHECKSUM_FILENAME="plur_${VERSION_NUM}_checksums.txt"
  CHECKSUM_URL="$BASE_URL/$CHECKSUM_FILENAME"

  info "Downloading from: $URL"

  # Create temp directory
  TMP_DIR=$(mktemp -d)
  trap 'rm -rf "$TMP_DIR"' EXIT

  # Download the archive
  download_to_file "$URL" "$TMP_DIR/plur.tar.gz"
  info "Downloading checksums from: $CHECKSUM_URL"
  download_to_file "$CHECKSUM_URL" "$TMP_DIR/checksums.txt"
  verify_checksum "$TMP_DIR/plur.tar.gz" "$TMP_DIR/checksums.txt" "$FILENAME"

  # Extract the archive
  info "Extracting archive..."
  tar -xzf "$TMP_DIR/plur.tar.gz" -C "$TMP_DIR" || error "Failed to extract archive"

  # Find the plur binary (it might be in a subdirectory)
  BINARY_PATH=$(find "$TMP_DIR" -name "plur" -type f | head -1)

  if [ -z "$BINARY_PATH" ]; then
    error "plur binary not found in archive"
  fi

  # Create install directory if it doesn't exist
  mkdir -p "$INSTALL_PATH"

  # Install the binary
  info "Installing plur to $INSTALL_PATH"
  mv "$BINARY_PATH" "$INSTALL_PATH/plur"
  chmod +x "$INSTALL_PATH/plur"

  # Verify installation
  if [ -x "$INSTALL_PATH/plur" ]; then
    INSTALLED_VERSION=$("$INSTALL_PATH/plur" --version 2>/dev/null || echo "unknown")
    printf "✓ Successfully installed plur\n"
    printf "  Version: %s\n" "$INSTALLED_VERSION"
    printf "  Location: %s/plur\n" "$INSTALL_PATH"

    # Check if plur is in PATH
    if ! command -v plur >/dev/null 2>&1; then
      warn "$INSTALL_PATH is not in your PATH"
      printf "  Add it with: export PATH=\"%s:\$PATH\"\n" "$INSTALL_PATH"
    fi
  else
    error "Installation failed"
  fi
}

# Main
main() {
  parse_args "$@"
  check_prerequisites

  printf "\n"
  info "plur installer"
  printf "\n"

  install_plur

  printf "\n"
  info "Installation complete!"
  printf "\n"

  # Show next steps
  printf "Next steps:\n"
  printf "  1. Run: plur --version\n"
  printf "  2. Run: plur doctor\n"
  printf "  3. Visit: https://github.com/%s/%s\n" "$REPO_OWNER" "$REPO_NAME"
  printf "\n"
}

main "$@"
