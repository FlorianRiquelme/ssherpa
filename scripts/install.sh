#!/bin/sh
set -e

# ssherpa installer
# Usage: curl -fsSL https://raw.githubusercontent.com/florianriquelme/ssherpa/main/scripts/install.sh | sh

REPO="florianriquelme/ssherpa"

# Detect OS
detect_os() {
    os=$(uname -s)
    case "$os" in
        Linux*)  echo "linux" ;;
        Darwin*) echo "darwin" ;;
        *)       echo "Unsupported OS: $os" >&2; exit 1 ;;
    esac
}

# Detect architecture
detect_arch() {
    arch=$(uname -m)
    case "$arch" in
        x86_64)  echo "amd64" ;;
        aarch64) echo "arm64" ;;
        arm64)   echo "arm64" ;;
        *)       echo "Unsupported architecture: $arch" >&2; exit 1 ;;
    esac
}

# Get latest version from GitHub API
get_latest_version() {
    version=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"v([^"]+)".*/\1/')
    if [ -z "$version" ]; then
        echo "Failed to determine latest version" >&2
        exit 1
    fi
    echo "$version"
}

main() {
    os=$(detect_os)
    arch=$(detect_arch)
    version="${VERSION:-$(get_latest_version)}"

    echo "Installing ssherpa v${version} for ${os}/${arch}..."

    url="https://github.com/${REPO}/releases/download/v${version}/ssherpa_${version}_${os}_${arch}.tar.gz"
    checksum_url="https://github.com/${REPO}/releases/download/v${version}/checksums.txt"

    tmpdir=$(mktemp -d)
    trap 'rm -rf "$tmpdir"' EXIT

    # Download archive and checksums
    curl -fsSL "$url" -o "${tmpdir}/ssherpa.tar.gz"
    curl -fsSL "$checksum_url" -o "${tmpdir}/checksums.txt"

    # Verify checksum
    cd "$tmpdir"
    expected=$(grep "ssherpa_${version}_${os}_${arch}.tar.gz" checksums.txt | awk '{print $1}')
    if [ -n "$expected" ]; then
        if command -v sha256sum >/dev/null 2>&1; then
            actual=$(sha256sum ssherpa.tar.gz | awk '{print $1}')
        elif command -v shasum >/dev/null 2>&1; then
            actual=$(shasum -a 256 ssherpa.tar.gz | awk '{print $1}')
        else
            echo "Warning: sha256sum/shasum not found, skipping checksum verification"
            actual="$expected"
        fi

        if [ "$actual" != "$expected" ]; then
            echo "Checksum verification failed!" >&2
            echo "Expected: $expected" >&2
            echo "Got:      $actual" >&2
            exit 1
        fi
        echo "Checksum verified."
    else
        echo "Warning: Could not find checksum for this platform, skipping verification"
    fi

    # Extract
    tar -xzf ssherpa.tar.gz

    # Install
    install_dir="${INSTALL_DIR:-/usr/local/bin}"
    if [ -w "$install_dir" ]; then
        mv ssherpa "$install_dir/ssherpa"
    else
        echo "Installing to ${install_dir} (requires sudo)..."
        sudo mv ssherpa "$install_dir/ssherpa"
    fi

    echo ""
    echo "ssherpa v${version} installed to ${install_dir}/ssherpa"
    echo "Run 'ssherpa' to get started!"
}

main
