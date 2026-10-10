#!/bin/sh
# Boots the patched ROM under BlastEm's GDB stub, stops in fade_in_hook on the first fade-in, the
# intro's first picture before any button, and checks the palette address left in a0.
# Usage: check-target.sh ELF ROM EXPECTED
set -u
dir=$(cd "$(dirname "$0")" && pwd)
elf=$1; rom=$2; want=$3
# BlastEm writes the ROM's saves into its home on exit, so it boots in a throwaway one, never the player's.
home=$(mktemp -d); trap 'rm -rf "$home"' EXIT
# shellcheck disable=SC2016  # $a0 is GDB syntax
out=$(timeout 120 gdb-multiarch -batch \
  -ex 'set architecture m68k' \
  -ex "target remote | env HOME=\"$home\" \"$dir/blastem.sh\" \"$rom\" -D" \
  -ex 'break fade_in_hook' -ex 'continue' \
  -ex 'p/x $a0' -ex 'kill' "$elf" 2>&1)
# shellcheck disable=SC2016  # $1 is GDB's value history
echo "$out" | grep -E 'Breakpoint 1, |^\$1 = '
echo "$out" | grep -q 'Breakpoint 1, .*fade_in_hook' || { echo "check-target: breakpoint on fade_in_hook did not hit" >&2; echo "$out" >&2; exit 1; }
echo "$out" | grep -q "^\$1 = $want\$" || { echo "check-target: expected a0 = $want" >&2; echo "$out" >&2; exit 1; }
echo "check-target: fade_in_hook saw a0 = $want"
