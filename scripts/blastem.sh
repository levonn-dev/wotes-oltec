#!/bin/sh
# Launches BlastEm on a ROM: the bootstrapped nightly in tools/blastem, or the binary
# named by BLASTEM. Usage: blastem.sh ROM [ARGS...]   (append -D for the GDB stub)
set -eu
dir=$(cd "$(dirname "$0")/.." && pwd)
blastem=${BLASTEM:-$dir/tools/blastem/blastem}
if [ ! -x "$blastem" ]; then
  echo "blastem.sh: BlastEm not found at $blastem; run 'task bootstrap' or set BLASTEM" >&2
  exit 1
fi
exec "$blastem" "$@"
