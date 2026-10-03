#!/bin/sh
# Serves the snapshot release in ./dist and runs install.sh against it.
set -eu
version=$(sed -n 's/.*"version": *"\([^"]*\)".*/\1/p' dist/metadata.json | head -n 1)
python3 -m http.server 8765 --directory dist >/dev/null 2>&1 &
server=$!
trap 'kill "$server" 2>/dev/null || true' EXIT
sleep 2
TSUZURI_VERSION="v$version" TSUZURI_BASE_URL=http://127.0.0.1:8765 \
	TSUZURI_INSTALL_DIR="$PWD/bin" sh ./install.sh
./bin/tsuzuri --version
./bin/tsuzuri --list-themes | grep -q onedark
echo "install.sh OK"
