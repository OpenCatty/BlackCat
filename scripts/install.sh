#!/bin/sh
# BlackCat Installer — POSIX sh
# Usage: curl -fsSL https://raw.githubusercontent.com/startower-observability/BlackCat/main/scripts/install.sh | sh
#    or: curl -fsSL ... | sh -s -- --version v2026.3.1
set -eu

# --- Defaults -----------------------------------------------------------------
REPO="startower-observability/BlackCat"
INSTALL_DIR="$HOME/.blackcat/bin"
BINARY_NAME="blackcat"
VERSION=""

# --- Parse flags --------------------------------------------------------------
while [ $# -gt 0 ]; do
  case "$1" in
    --version)
      shift
      VERSION="${1:-}"
      if [ -z "$VERSION" ]; then
        echo "Error: --version requires a value" >&2
        exit 1
      fi
      ;;
    --help|-h)
      echo "Usage: install.sh [--version VERSION]"
      echo ""
      echo "Flags:"
      echo "  --version VERSION  Install a specific version (e.g. v2026.3.1)"
      echo "  --help             Show this help"
      exit 0
      ;;
    *)
      echo "Error: unknown flag: $1" >&2
      exit 1
      ;;
  esac
  shift
done

# --- Dependency checks --------------------------------------------------------
check_cmd() {
  command -v "$1" >/dev/null 2>&1
}

if ! check_cmd curl; then
  echo "Error: curl is required but not found. Install curl and retry." >&2
  exit 1
fi

if ! check_cmd tar; then
  echo "Error: tar is required but not found. Install tar and retry." >&2
  exit 1
fi

# --- Detect OS ----------------------------------------------------------------
detect_os() {
  os="$(uname -s)"
  case "$os" in
    Linux*)  echo "Linux" ;;
    Darwin*) echo "Darwin" ;;
    MINGW*|MSYS*|CYGWIN*) echo "Windows" ;;
    *)
      echo "Error: unsupported operating system: $os" >&2
      exit 1
      ;;
  esac
}

# --- Detect ARCH --------------------------------------------------------------
detect_arch() {
  arch="$(uname -m)"
  case "$arch" in
    x86_64|amd64)  echo "x86_64" ;;
    aarch64|arm64)  echo "arm64" ;;
    *)
      echo "Error: unsupported architecture: $arch" >&2
      exit 1
      ;;
  esac
}

# --- Resolve version ----------------------------------------------------------
resolve_version() {
  if [ -n "$VERSION" ]; then
    echo "$VERSION"
    return
  fi

  tag="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep '"tag_name"' \
    | sed -E 's/.*"tag_name":\s*"([^"]+)".*/\1/')"

  if [ -z "$tag" ]; then
    echo "Error: could not determine latest release version" >&2
    exit 1
  fi
  echo "$tag"
}

# --- Main ---------------------------------------------------------------------
main() {
  OS="$(detect_os)"
  ARCH="$(detect_arch)"
  VER="$(resolve_version)"

  # GoReleaser naming convention: blackcat_Linux_x86_64.tar.gz
  ARCHIVE="${BINARY_NAME}_${OS}_${ARCH}.tar.gz"
  URL="https://github.com/${REPO}/releases/download/${VER}/${ARCHIVE}"

  echo "Installing BlackCat ${VER} (${OS}/${ARCH})..."

  # Create install directory
  mkdir -p "$INSTALL_DIR"

  # Download and extract
  TMPDIR_DL="$(mktemp -d)"
  trap 'rm -rf "$TMPDIR_DL"' EXIT

  echo "  Downloading ${URL}..."
  curl -fsSL "$URL" -o "${TMPDIR_DL}/${ARCHIVE}"

  echo "  Extracting..."
  tar -xzf "${TMPDIR_DL}/${ARCHIVE}" -C "$TMPDIR_DL"

  # Install binary
  if [ -f "${TMPDIR_DL}/${BINARY_NAME}" ]; then
    mv "${TMPDIR_DL}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
    chmod +x "${INSTALL_DIR}/${BINARY_NAME}"
  else
    echo "Error: binary '${BINARY_NAME}' not found in archive" >&2
    exit 1
  fi

  # Add to PATH in shell rc files
  PATH_LINE="export PATH=\"\$HOME/.blackcat/bin:\$PATH\""
  for rc in "$HOME/.bashrc" "$HOME/.zshrc" "$HOME/.profile"; do
    if [ -f "$rc" ]; then
      if ! grep -qF '.blackcat/bin' "$rc" 2>/dev/null; then
        echo "" >> "$rc"
        echo "# BlackCat" >> "$rc"
        echo "$PATH_LINE" >> "$rc"
      fi
    fi
  done

  echo ""
  echo "BlackCat ${VER} installed to ${INSTALL_DIR}/${BINARY_NAME}"
  echo ""
  echo "To get started, restart your shell (or run: export PATH=\"\$HOME/.blackcat/bin:\$PATH\") and then:"
  echo ""
  echo "  blackcat onboard"
  echo ""
}

main
