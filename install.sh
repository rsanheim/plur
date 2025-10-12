#!/bin/sh
# Universal installation script for plur
# Usage:
#   curl -sSL https://raw.githubusercontent.com/rsanheim/plur/main/install.sh | sh
#   curl -sSL https://raw.githubusercontent.com/rsanheim/plur/main/install.sh | sh -s -- v1.0.0
#
# Environment variables:
#   PLUR_INSTALL_PATH - Installation directory (default: /usr/local/bin)
#   PLUR_VERSION - Version to install (default: latest)

set -e

# Configuration
REPO_OWNER="rsanheim"
REPO_NAME="plur"
INSTALL_PATH="${PLUR_INSTALL_PATH:-/usr/local/bin}"
VERSION="${PLUR_VERSION:-}"

# Colors for output (if terminal supports it)
if [ -t 1 ]; then
  RED='\033[0;31m'
  GREEN='\033[0;32m'
  YELLOW='\033[1;33m'
  NC='\033[0m' # No Color
else
  RED=''
  GREEN=''
  YELLOW=''
  NC=''
fi

# Helper functions
info() {
  printf "${GREEN}>>>>${NC} %s\n" "$1"
}

warn() {
  printf "${YELLOW}Warning:${NC} %s\n" "$1"
}

error() {
  printf "${RED}Error:${NC} %s\n" "$1"
  exit 1
}

# Parse command line arguments
if [ $# -gt 0 ]; then
  VERSION="$1"
fi

# Detect architecture
detect_arch() {
  ARCH=$(uname -m)
  case $ARCH in
    x86_64)
      echo "x86_64"
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
      echo "Linux"
      ;;
    darwin)
      echo "Darwin"
      ;;
    *)
      error "Unsupported OS: $OS"
      ;;
  esac
}

# Get latest version from GitHub
get_latest_version() {
  if command -v curl >/dev/null 2>&1; then
    LATEST=$(curl -sSL "https://api.github.com/repos/$REPO_OWNER/$REPO_NAME/releases/latest" | \
             grep '"tag_name":' | \
             sed -E 's/.*"([^"]+)".*/\1/')
  elif command -v wget >/dev/null 2>&1; then
    LATEST=$(wget -qO- "https://api.github.com/repos/$REPO_OWNER/$REPO_NAME/releases/latest" | \
             grep '"tag_name":' | \
             sed -E 's/.*"([^"]+)".*/\1/')
  else
    error "Neither curl nor wget found. Please install one of them."
  fi

  if [ -z "$LATEST" ]; then
    error "Failed to fetch latest version from GitHub"
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
  URL="https://github.com/$REPO_OWNER/$REPO_NAME/releases/download/$VERSION/$FILENAME"

  info "Downloading from: $URL"

  # Create temp directory
  TMP_DIR=$(mktemp -d)
  trap "rm -rf $TMP_DIR" EXIT

  # Download the archive
  if command -v curl >/dev/null 2>&1; then
    curl -sSL "$URL" -o "$TMP_DIR/plur.tar.gz" || error "Failed to download plur"
  elif command -v wget >/dev/null 2>&1; then
    wget -q "$URL" -O "$TMP_DIR/plur.tar.gz" || error "Failed to download plur"
  fi

  # Extract the archive
  info "Extracting archive..."
  tar -xzf "$TMP_DIR/plur.tar.gz" -C "$TMP_DIR" || error "Failed to extract archive"

  # Find the plur binary (it might be in a subdirectory)
  BINARY_PATH=$(find "$TMP_DIR" -name "plur" -type f | head -1)

  if [ -z "$BINARY_PATH" ]; then
    error "plur binary not found in archive"
  fi

  # Check if we need sudo for installation
  if [ -w "$INSTALL_PATH" ]; then
    SUDO=""
  else
    if command -v sudo >/dev/null 2>&1; then
      info "Requesting sudo access for installation to $INSTALL_PATH"
      SUDO="sudo"
    else
      error "Cannot write to $INSTALL_PATH and sudo is not available"
    fi
  fi

  # Create install directory if it doesn't exist
  if [ ! -d "$INSTALL_PATH" ]; then
    info "Creating installation directory: $INSTALL_PATH"
    $SUDO mkdir -p "$INSTALL_PATH"
  fi

  # Install the binary
  info "Installing plur to $INSTALL_PATH"
  $SUDO mv "$BINARY_PATH" "$INSTALL_PATH/plur"
  $SUDO chmod +x "$INSTALL_PATH/plur"

  # Verify installation
  if [ -x "$INSTALL_PATH/plur" ]; then
    INSTALLED_VERSION=$("$INSTALL_PATH/plur" --version 2>/dev/null || echo "unknown")
    printf "${GREEN}✓${NC} Successfully installed plur\n"
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