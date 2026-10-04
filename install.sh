#!/bin/sh
# Tsuzuri installer for macOS and Linux (any distribution, x86-64).
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

os=$(uname -s)
arch=$(uname -m)
case "$os" in
	Darwin) archive=tsuzuri-macos.tar.gz ;; # universal: Intel and Apple Silicon
	Linux)
		case "$arch" in
			x86_64 | amd64) archive=tsuzuri-linux.tar.gz ;;
			*) die "prebuilt Linux binaries are x86-64 only (this is $arch); install with: go install github.com/$REPO/cmd/tsuzuri@latest" ;;
		esac
		;;
	*) die "unsupported OS: $os (on Windows use install.ps1)" ;;
esac

version=${TSUZURI_VERSION:-latest}
if [ -n "${TSUZURI_BASE_URL:-}" ]; then
	base=$TSUZURI_BASE_URL
elif [ "$version" = latest ]; then
	base="https://github.com/$REPO/releases/latest/download"
else
	base="https://github.com/$REPO/releases/download/$version"
fi

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT INT TERM

say "Downloading Tsuzuri ($version) for $os..."
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
