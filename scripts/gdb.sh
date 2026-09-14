#!/bin/sh
# GDB for the shared IDEA debug profile: a repo path IDEA stores as $PROJECT_DIR$, unlike /usr/bin.
exec gdb-multiarch "$@"
