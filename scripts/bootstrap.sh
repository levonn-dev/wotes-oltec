#!/bin/sh
# Installs the pinned BlastEm nightly and mdrom release into tools/, fetches planetchili's two patches,
# builds wotes-tiled when Go is present, then checks the apt tools and the clean ROM.
# Usage: scripts/bootstrap.sh (or: task bootstrap). Never sudo.
set -eu

BLASTEM_VERSION=0.6.3-pre-8013468ed981
BLASTEM_SHA256=26539e1efe89aea1d79663372856e40fe874fa141517276bedc47c0c476604c8
MDROM_REPO=levonn-dev/mdrom
MDROM_VERSION=v0.1.0
# planetchili's patches, one a release: tag, asset, the asset's SHA-256.
CHILI_REPO=planetchili/wotes_patch
CHILI_PATCHES='V1.0:wotes_chili_patch_1_0.zip:a008cb008d93299667956aa3a45c1662b1a0de717619248f3d31036e457033f9
V2.0:wotes_chili_patch_2_0.zip:edcbfb8bc9e17bd82353f0f16bf51bf8a12f20c1e2f5047e19b4d3138b16d54c'

ROM_NAME='Dungeons & Dragons - Warriors of the Eternal Sun (USA, Europe).md'
ROM_SHA1=9135f7fda03ef7da92dfade9c0df75808214f693

cd "$(dirname "$0")/.."
command -v curl >/dev/null 2>&1 || { echo "bootstrap: curl is required (sudo apt install curl)" >&2; exit 1; }
command -v unzip >/dev/null 2>&1 || { echo "bootstrap: unzip is required (sudo apt install unzip)" >&2; exit 1; }
mkdir -p tools

fetch() { # URL DEST
  curl -fsSL --retry 3 -o "$2" "$1" || { echo "bootstrap: download failed: $1" >&2; exit 1; }
}

install_blastem() {
  if [ -f tools/blastem/VERSION ] && [ "$(cat tools/blastem/VERSION)" = "$BLASTEM_VERSION" ]; then
    echo "bootstrap: BlastEm $BLASTEM_VERSION present"
    return
  fi
  if [ "$(uname -sm)" != "Linux x86_64" ]; then
    echo "bootstrap: BlastEm nightlies exist only for Linux x86_64; build BlastEm yourself and set BLASTEM to the binary"
    return
  fi
  name=blastem64-$BLASTEM_VERSION
  echo "bootstrap: downloading BlastEm $BLASTEM_VERSION"
  fetch "https://www.retrodev.com/blastem/nightlies/$name.tar.gz" "tools/$name.tar.gz.part"
  echo "$BLASTEM_SHA256  tools/$name.tar.gz.part" | sha256sum -c --quiet \
    || { rm -f "tools/$name.tar.gz.part"; echo "bootstrap: BlastEm download failed its checksum" >&2; exit 1; }
  rm -rf "tools/$name" tools/blastem
  tar -xzf "tools/$name.tar.gz.part" -C tools
  mv "tools/$name" tools/blastem
  rm -f "tools/$name.tar.gz.part"
  # With no user config BlastEm reads default.cfg beside the binary; GL cannot set vsync under WSLg.
  sed -i 's/^\([[:space:]]*\)gl on$/\1gl off/' tools/blastem/default.cfg
  grep -q '^[[:space:]]*gl off$' tools/blastem/default.cfg \
    || { echo "bootstrap: could not set gl off in tools/blastem/default.cfg" >&2; exit 1; }
  echo "$BLASTEM_VERSION" > tools/blastem/VERSION
  echo "bootstrap: BlastEm installed in tools/blastem"
}

install_mdrom() {
  if [ -f tools/mdrom/VERSION ] && [ "$(cat tools/mdrom/VERSION)" = "$MDROM_VERSION" ]; then
    echo "bootstrap: mdrom $MDROM_VERSION present"
    return
  fi
  if [ -x tools/mdrom/mdrom ] && [ ! -f tools/mdrom/VERSION ]; then
    echo "bootstrap: using the local mdrom build in tools/mdrom (delete it to fetch $MDROM_VERSION)"
    return
  fi
  case $(uname -m) in
    x86_64) arch=amd64 ;;
    aarch64) arch=arm64 ;;
    *) echo "bootstrap: no mdrom release for $(uname -m); build it from https://github.com/$MDROM_REPO and copy the binary to tools/mdrom/mdrom" >&2; exit 1 ;;
  esac
  archive=mdrom_${MDROM_VERSION#v}_linux_$arch.tar.gz
  base=https://github.com/$MDROM_REPO/releases/download/$MDROM_VERSION
  echo "bootstrap: downloading mdrom $MDROM_VERSION"
  rm -rf tools/mdrom.part
  mkdir -p tools/mdrom.part
  fetch "$base/$archive" "tools/mdrom.part/$archive"
  fetch "$base/checksums.txt" tools/mdrom.part/checksums.txt
  (cd tools/mdrom.part && sha256sum -c --quiet --ignore-missing checksums.txt)
  tar -xzf "tools/mdrom.part/$archive" -C tools/mdrom.part mdrom
  rm -rf tools/mdrom
  mkdir -p tools/mdrom
  mv tools/mdrom.part/mdrom tools/mdrom/mdrom
  echo "$MDROM_VERSION" > tools/mdrom/VERSION
  rm -rf tools/mdrom.part
  echo "bootstrap: mdrom installed in tools/mdrom"
}

install_chili() {
  present=yes
  for entry in $CHILI_PATCHES; do [ -f "tools/chili/${entry%%:*}.bps" ] || present=; done
  if [ -n "$present" ] && [ -f tools/chili/VERSION ] && [ "$(cat tools/chili/VERSION)" = "$CHILI_PATCHES" ]; then
    echo "bootstrap: planetchili's patches present"
    return
  fi
  rm -rf tools/chili.part tools/chili
  mkdir -p tools/chili.part
  for entry in $CHILI_PATCHES; do
    tag=${entry%%:*}; rest=${entry#*:}; asset=${rest%%:*}; sum=${rest#*:}
    echo "bootstrap: downloading planetchili's patch $tag"
    fetch "https://github.com/$CHILI_REPO/releases/download/$tag/$asset" "tools/chili.part/$asset"
    echo "$sum  tools/chili.part/$asset" | sha256sum -c --quiet \
      || { rm -rf tools/chili.part; echo "bootstrap: planetchili's $tag download failed its checksum" >&2; exit 1; }
    # Each zip holds one BPS file, named .md.bst.
    unzip -p "tools/chili.part/$asset" '*.bst' > "tools/chili.part/$tag.bps"
    rm "tools/chili.part/$asset"
  done
  printf '%s\n' "$CHILI_PATCHES" > tools/chili.part/VERSION
  mv tools/chili.part tools/chili
  echo "bootstrap: planetchili's patches installed in tools/chili"
}

build_wotes_tiled() {
  if command -v go >/dev/null 2>&1 && go version | grep -Eq 'go(1\.(2[6-9]|[3-9][0-9])|[2-9])'; then
    mkdir -p tools/wotes-tiled
    (cd wotes-tiled && go build -o ../tools/wotes-tiled/wotes-tiled .) \
      || { echo "bootstrap: building wotes-tiled failed" >&2; exit 1; }
    echo "bootstrap: wotes-tiled built into tools/wotes-tiled"
  else
    echo "bootstrap: go 1.26 or later is required for wotes-tiled: install it with goenv or from https://go.dev/dl/" >&2
  fi
}

check_tools() {
  missing=
  for pair in cmake:cmake ninja:ninja-build gcc:gcc m68k-linux-gnu-gcc:gcc-m68k-linux-gnu \
              gdb-multiarch:gdb-multiarch readelf:binutils shellcheck:shellcheck; do
    tool=${pair%%:*}
    command -v "$tool" >/dev/null 2>&1 || missing="$missing ${pair#*:}"
  done
  if [ -n "$missing" ]; then
    echo "bootstrap: missing tools; install them with:" >&2
    echo "  sudo apt install$missing" >&2
    exit 1
  fi
  echo "bootstrap: cmake ninja gcc m68k-linux-gnu-gcc gdb-multiarch readelf shellcheck present"
}

check_rom() {
  if [ -f "$ROM_NAME" ] && [ "$(sha1sum "$ROM_NAME" | cut -c1-40)" = "$ROM_SHA1" ]; then
    echo "bootstrap: clean ROM present"
  else
    echo "bootstrap: clean ROM missing or not the expected image; copy '$ROM_NAME' (SHA-1 $ROM_SHA1) here, or run: task configure -- -DCLEAN_ROM=<path>"
  fi
}

install_blastem
install_mdrom
install_chili
build_wotes_tiled
check_tools
check_rom
echo "bootstrap OK"
