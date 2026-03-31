#!/bin/bash
set -e

REPO="sunneydev/snapshot"
INSTALL_DIR="/usr/local/bin"

case "$(uname -s)" in
    Darwin) os="darwin" ;;
    Linux) os="linux" ;;
    *) echo "unsupported os: $(uname -s)" >&2; exit 1 ;;
esac

case "$(uname -m)" in
    x86_64|amd64) arch="amd64" ;;
    arm64|aarch64) arch="arm64" ;;
    *) echo "unsupported arch: $(uname -m)" >&2; exit 1 ;;
esac

if [ "$os" = "darwin" ] && [ "$arch" = "amd64" ]; then
    if [ "$(sysctl -n sysctl.proc_translated 2>/dev/null)" = "1" ]; then
        arch="arm64"
    fi
fi

ext="tar.gz"
[ "$os" = "darwin" ] && ext="zip"

name="snapshot_${os}_${arch}.${ext}"
url="https://github.com/${REPO}/releases/latest/download/${name}"

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

echo "downloading snapshot..."
curl -fsSL -o "$tmp/$name" "$url"

if [ "$ext" = "tar.gz" ]; then
    tar xzf "$tmp/$name" -C "$tmp"
else
    unzip -qo "$tmp/$name" -d "$tmp"
fi

if [ ! -w "$INSTALL_DIR" ]; then
    echo "installing to $INSTALL_DIR (requires sudo)..."
    sudo install -m 755 "$tmp/snapshot" "$INSTALL_DIR/snapshot"
else
    install -m 755 "$tmp/snapshot" "$INSTALL_DIR/snapshot"
fi

if ! command -v restic >/dev/null 2>&1; then
    echo "installing restic..."
    if command -v brew >/dev/null 2>&1; then
        brew install restic
    elif command -v apt-get >/dev/null 2>&1; then
        sudo apt-get install -y restic
    else
        echo "please install restic manually: https://restic.net"
    fi
fi

echo "snapshot installed to $INSTALL_DIR/snapshot"
snapshot --version
