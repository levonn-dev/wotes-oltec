#!/bin/sh
# Boots the patched ROM under BlastEm's GDB stub, stops in poc_init and again right
# after it returns (poc_ret in hooks/hooks.s), and checks the value left in d0.
# Usage: check-target.sh ELF ROM EXPECTED
set -u
dir=$(cd "$(dirname "$0")" && pwd)
elf=$1; rom=$2; want=$3
# shellcheck disable=SC2016  # $d0 is GDB syntax
out=$(timeout 120 gdb-multiarch -batch \
  -ex 'set architecture m68k' \
  -ex "target remote | \"$dir/blastem.sh\" \"$rom\" -D" \
  -ex 'break poc_init' -ex 'continue' \
  -ex 'break poc_ret' -ex 'continue' \
  -ex 'p/d $d0' -ex 'kill' "$elf" 2>&1)
# shellcheck disable=SC2016  # $1 is GDB's value history
echo "$out" | grep -E 'Breakpoint [12], |^\$1 = '
echo "$out" | grep -q 'Breakpoint 1, poc_init' || { echo "check-target: breakpoint on poc_init did not hit" >&2; echo "$out" >&2; exit 1; }
echo "$out" | grep -q 'Breakpoint 2, poc_ret' || { echo "check-target: breakpoint on poc_ret did not hit" >&2; echo "$out" >&2; exit 1; }
echo "$out" | grep -q "^\$1 = $want\$" || { echo "check-target: expected d0 = $want" >&2; echo "$out" >&2; exit 1; }
echo "check-target: poc_init returned $want"
