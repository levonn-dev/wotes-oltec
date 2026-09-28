#!/bin/sh
# Installs the pinned Tiled AppImage into tools/tiled, extracted because WSL2 has no FUSE by default.
# Usage: scripts/install-tiled.sh (or: task tiled:install). Optional: the build never needs Tiled.
set -eu

TILED_VERSION=1.12.2
TILED_SHA256=5e0edbff61314f41af3c72c21ec006b363cf12047cc9cfb5bbd63a98bca3721c

cd "$(dirname "$0")/.."
if [ -f tools/tiled/VERSION ] && [ "$(cat tools/tiled/VERSION)" = "$TILED_VERSION" ]; then
  echo "install-tiled: Tiled $TILED_VERSION present"
  exit 0
fi
command -v curl >/dev/null 2>&1 || { echo "install-tiled: curl is required (sudo apt install curl)" >&2; exit 1; }
if [ "$(uname -sm)" != "Linux x86_64" ]; then
  echo "install-tiled: the AppImage is Linux x86_64 only; install Tiled from https://www.mapeditor.org/ and open the maps/ files with it" >&2
  exit 1
fi
name="Tiled-${TILED_VERSION}_Linux_x86_64.AppImage"
mkdir -p tools
echo "install-tiled: downloading Tiled $TILED_VERSION"
curl -fsSL --retry 3 -o "tools/$name.part" "https://github.com/mapeditor/tiled/releases/download/v$TILED_VERSION/$name" \
  || { echo "install-tiled: download failed" >&2; rm -f "tools/$name.part"; exit 1; }
echo "$TILED_SHA256  tools/$name.part" | sha256sum -c --quiet \
  || { echo "install-tiled: the download's checksum is $(sha256sum "tools/$name.part" | cut -c1-64), not the pinned one" >&2; rm -f "tools/$name.part"; exit 1; }
rm -rf tools/tiled.part
mkdir -p tools/tiled.part
mv "tools/$name.part" "tools/tiled.part/$name"
chmod +x "tools/tiled.part/$name"
(cd tools/tiled.part && "./$name" --appimage-extract >/dev/null)
rm "tools/tiled.part/$name"
rm -rf tools/tiled
mv tools/tiled.part tools/tiled
echo "$TILED_VERSION" > tools/tiled/VERSION
echo "install-tiled: Tiled installed in tools/tiled; open a map with: task tiled -- maps/rom/valley.tmj"
