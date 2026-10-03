#!/bin/sh
# Tsuzuri installer for macOS, Linux (any distro) and FreeBSD.
#
#   curl -fsSL https://raw.githubusercontent.com/jaisuriya-11/tsuzuri/main/install.sh | sh
#
# Environment overrides:
#   TSUZURI_VERSION      release tag to install (default: latest), e.g. v0.1.0
#   TSUZURI_INSTALL_DIR  where to put the binary (default: /usr/local/bin if
#                        writable or sudo is available, else ~/.local/bin)
#   TSUZURI_BASE_URL     download location (default: GitHub releases)
set -eu

REPO="jaisuriya-11/tsuzuri"

say() { printf '%s\n' "$*"; }
die() { printf 'tsuzuri install: %s\n' "$*" >&2; exit 1; }
have() { command -v "$1" >/dev/null 2>&1; }

fetch() { # fetch URL OUTFILE
	if have curl; then
		curl -fsSL "$1" -o "$2"
	elif have wget; then
		wget -qO "$2" "$1"
	else
		die "need curl or wget"
	fi
}

os=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$os" in
	linux | darwin | freebsd) ;;
	*) die "unsupported OS: $os (on Windows use install.ps1)" ;;
esac

arch=$(uname -m)
case "$arch" in
	x86_64 | amd64) arch=amd64 ;;
	aarch64 | arm64) arch=arm64 ;;
	armv7* | armv8l) arch=armv7 ;;
	i386 | i686) arch=386 ;;
	*) die "unsupported CPU: $arch" ;;
esac

version=${TSUZURI_VERSION:-}
if [ -z "$version" ]; then
	tmpjson=$(mktemp)
	fetch "https://api.github.com/repos/$REPO/releases/latest" "$tmpjson" || die "could not reach GitHub"
	version=$(sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' "$tmpjson" | head -n 1)
	rm -f "$tmpjson"
	[ -n "$version" ] || die "could not find the latest release"
fi
plain=${version#v}

base=${TSUZURI_BASE_URL:-https://github.com/$REPO/releases/download/$version}
archive="tsuzuri_${plain}_${os}_${arch}.tar.gz"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT INT TERM

say "Downloading Tsuzuri ${version} for ${os}/${arch}..."
fetch "$base/$archive" "$tmp/$archive" || die "download failed: $base/$archive"

if fetch "$base/checksums.txt" "$tmp/checksums.txt" 2>/dev/null; then
	want=$(grep " $archive\$" "$tmp/checksums.txt" | cut -d ' ' -f 1)
	if have sha256sum; then
		got=$(sha256sum "$tmp/$archive" | cut -d ' ' -f 1)
	elif have shasum; then
		got=$(shasum -a 256 "$tmp/$archive" | cut -d ' ' -f 1)
	else
		got=$want
	fi
	[ -z "$want" ] || [ "$want" = "$got" ] || die "checksum mismatch for $archive"
fi

tar -xzf "$tmp/$archive" -C "$tmp" tsuzuri || die "could not unpack $archive"
chmod +x "$tmp/tsuzuri"

dir=${TSUZURI_INSTALL_DIR:-}
sudo=""
if [ -z "$dir" ]; then
	if [ -w /usr/local/bin ]; then
		dir=/usr/local/bin
	elif have sudo && [ -d /usr/local/bin ]; then
		dir=/usr/local/bin
		sudo=sudo
	else
		dir="$HOME/.local/bin"
	fi
fi
mkdir -p "$dir" 2>/dev/null || $sudo mkdir -p "$dir"
if [ -w "$dir" ]; then sudo=""; fi
$sudo mv "$tmp/tsuzuri" "$dir/tsuzuri"

say "Installed $("$dir/tsuzuri" --version) to $dir/tsuzuri"
case ":$PATH:" in
	*":$dir:"*) ;;
	*) say "Add $dir to your PATH, e.g.:  echo 'export PATH=\"$dir:\$PATH\"' >> ~/.profile" ;;
esac
say "Run 'tsuzuri' in a folder of notes (a Nerd Font is recommended for icons)."
