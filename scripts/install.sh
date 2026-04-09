#!/bin/sh
# DroidLink installer — https://github.com/Rohit-554/droidLink
# Usage: curl -fsSL https://raw.githubusercontent.com/Rohit-554/droidLink/main/scripts/install.sh | sh

set -e

REPO="Rohit-554/droidLink"
BINARY="droidlink"
INSTALL_DIR="${DROIDLINK_INSTALL_DIR:-/usr/local/bin}"

resolve_latest_version() {
    curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
        | grep '"tag_name"' \
        | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/'
}

resolve_platform() {
    os="$(uname -s | tr '[:upper:]' '[:lower:]')"
    arch="$(uname -m)"
    case "$arch" in
        x86_64)  arch="amd64" ;;
        aarch64) arch="arm64" ;;
        arm64)   arch="arm64" ;;
        *)
            echo "Unsupported architecture: $arch" >&2
            exit 1
        ;;
    esac
    echo "${os}_${arch}"
}

download_and_install() {
    version="$1"
    platform="$2"

    archive_name="${BINARY}_${platform}"
    download_url="https://github.com/${REPO}/releases/download/${version}/${archive_name}.tar.gz"
    tmp_dir="$(mktemp -d)"

    echo "Downloading droidlink ${version} for ${platform}..."
    curl -fsSL "$download_url" | tar -xz -C "$tmp_dir"

    install_binary "$tmp_dir"
    rm -rf "$tmp_dir"
}

install_binary() {
    extracted_dir="$1"
    binary_path="${extracted_dir}/${BINARY}"

    if [ ! -f "$binary_path" ]; then
        echo "Binary not found in archive." >&2
        exit 1
    fi

    chmod +x "$binary_path"

    if [ -w "$INSTALL_DIR" ]; then
        mv "$binary_path" "${INSTALL_DIR}/${BINARY}"
    else
        echo "Installing to ${INSTALL_DIR} (requires sudo)..."
        sudo mv "$binary_path" "${INSTALL_DIR}/${BINARY}"
    fi
}

verify_installation() {
    if command -v "$BINARY" >/dev/null 2>&1; then
        echo "droidlink installed successfully at $(command -v $BINARY)"
        "$BINARY" --help | head -1
    else
        echo "Installed to ${INSTALL_DIR}/${BINARY}"
        echo "Make sure ${INSTALL_DIR} is in your PATH."
    fi
}

main() {
    version="${1:-$(resolve_latest_version)}"
    if [ -z "$version" ]; then
        echo "Could not resolve latest version. Pass a version explicitly: install.sh v1.0.0" >&2
        exit 1
    fi

    platform="$(resolve_platform)"
    download_and_install "$version" "$platform"
    verify_installation
}

main "$@"
