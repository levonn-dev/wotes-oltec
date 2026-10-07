# Oltec land patch for Warriors of the Eternal Sun

C and 68000 assembly compiled with the m68k-linux-gnu GCC, injected into the clean ROM with [mdrom](https://github.com/levonn-dev/mdrom), and shipped as a BPS patch.

## Goal

A BPS patch that gives Warriors of the Eternal Sun a fourth outdoor region: an Oltec land true to the Hollow World book, entered from the bottom of the Oltec Tradeway dungeon, so the meeting with the Oltec merchant happens in that land instead of at the dungeon's foot. The game's three regions all follow one pattern, so the patch extends the engine's own tables and loaders rather than replacing them.

New code lives in the 2 MB expansion as 68000 assembly and C, reached through six-byte `jsr` hooks; the game tables the patch extends are spliced out of the clean ROM at build time, which is why the build copies it to `build/m68k/clean.md`.

## Status

- [x] Build flow: cross-compiled with GCC, injected into a 2 MB image with mdrom, shipped as a BPS patch, with 68000-instruction and hook checks in the build.
- [x] Proof of concept: a startup hook that runs C code, and the FIGHTER class renamed to WARRIOR.
- [x] Reverse engineering: the region loader and its tables, the save format, the dungeon transitions and the world map, named as linker symbols and RAM overlays in `game/`.
- [x] A fourth region reachable in play: region 4, cloned from the jungle, entered and left through the Tradeway's bottom level, saved and loaded as OLTEC.
- [x] The region's own identity: an area id, music and an encounter list of its own.
- [x] A Tiled toolchain: every outdoor tile set and map extracted into Tiled with its entrances, places, figures and fixed fights, new maps converted into the engine's tables at build time.
- [x] Objects on a Tiled land map: entrances, shops, speeches, a restore and figures, converted into tables a region reads.
- [x] The dungeon levels in Tiled: all 38 extracted top-down with their walls, entry points, items, monsters, events and stairs, and the tool's knowledge of the ROM in a layout file another project can replace.
- [x] The Oltec land itself, drawn in Tiled from the Hollow World book: its map, its places and its people.
- [x] The caves of the land and the Oltec dungeon behind them, drawn in Tiled as one map that the build turns into the game's own levels.
- [x] The dungeon's second pass: its monsters, treasure and texts.
- [ ] The region's own map screen, drawn from the finished land.
- [ ] The merchant meeting moved into the land, with the medallion and the quest picture it grants restored.

## Prerequisites

- Ubuntu 24.04, natively or under WSL2 (the emulator window needs WSLg or an X display).
- [Task](https://taskfile.dev/installation/) 3.
- Go 1.26 or later, through [goenv](https://github.com/go-nv/goenv) or [go.dev](https://go.dev/dl/); apt's Go is too old. It builds `wotes-tiled`, which turns Tiled maps into the tables the patch links.
- Optional: [Tiled](https://www.mapeditor.org/) 1.12, fetched into `tools/` by `task tiled:install`, or your own install; see `docs/tiled.md`.
- The cross compiler, debugger, and helpers:

  ```
  sudo apt install cmake ninja-build gcc gcc-m68k-linux-gnu gdb-multiarch binutils shellcheck curl
  ```

- The clean ROM, `Dungeons & Dragons - Warriors of the Eternal Sun (USA, Europe).md`, SHA-1 `9135f7fda03ef7da92dfade9c0df75808214f693`. It is not part of the repository.

## Quick start

```
git clone https://github.com/levonn-dev/wotes-oltec.git
cd wotes-oltec
task bootstrap
cp /path/to/'Dungeons & Dragons - Warriors of the Eternal Sun (USA, Europe).md' .
task build
```

`task bootstrap` downloads the BlastEm nightly and the mdrom release into `tools/`, builds `wotes-tiled` there when Go is present, checks the apt tools, and prints the apt line above if one is missing. It writes nothing outside the repository and never uses sudo. A ROM kept elsewhere is named with `task configure -- -DCLEAN_ROM=<path>`; options after `--` reach CMake even when the tree is already configured.

## Tasks

`task` with no arguments lists them.

| Task | Does |
|---|---|
| `bootstrap` | BlastEm and mdrom into `tools/`, wotes-tiled built there; checks the apt tools and the clean ROM |
| `build` | Configures `build/m68k/` with the `m68k` preset if needed, then links, runs the 68000 and hook checks, injects, writes `patch.bps`, and runs the host helper test |
| `run` | The patched ROM in BlastEm |
| `check` | `build`, the wotes-tiled round trip against the ROM, then the scripted GDB check of `poc_init` on the target |
| `check:host` | Helper test, wotes-tiled tests, shellcheck; no cross compiler or ROM needed (what CI runs) |
| `check:target` | The on-target GDB check alone |
| `debug` | Interactive `gdb-multiarch` against BlastEm, paused at the reset vector |
| `tiled:extract` | Every outdoor tile set and map of the clean ROM into `maps/rom/` as Tiled files |
| `tiled:render -- ARGS` | A map or a ROM region to PNG |
| `tiled:install` | Tiled 1.12 into `tools/tiled/` (optional) |
| `tiled -- FILE` | Opens a map in that Tiled |
| `tiled:build`, `tiled:test`, `tiled:roundtrip` | Build, test and round-trip the tool alone |
| `configure` | The CMake configure alone; extra options after `--` |
| `clean` | Removes `build/` and the Task cache (keeps `tools/`) |

## Outputs

`build/m68k/patch.elf` (the linked patch with DWARF), `build/m68k/patched.md` (the 2MB patched image), `build/m68k/patch.bps` (the patch to distribute).

## Layout

- `src/`, `hooks/`, `game/`, `link/`: patch code, hook sections, game symbols, linker script
- `cmake/`: toolchain file and the mdrom lookup
- `wotes-tiled/`: the Go tool that moves outdoor maps between the ROM and Tiled (`docs/tiled.md`)
- `maps/`: land maps, metatile definitions, new tiles and the Tiled project file; `maps/rom/` and `maps/gen/` are generated and not committed
- `scripts/`: bootstrap, the BlastEm launcher, and the build-time checks
- `tests/`: host-side test of the runtime helpers
- `docs/`: `debugging.md` for GDB and IDEA, `tiled.md` for drawing maps
- `tools/`: created by bootstrap, not committed
- `.github/`: the CI workflow, which runs `task check:host`, and `CODEOWNERS`

## License

MIT, see `LICENSE`. The game and its ROM belong to their rights holders; this repository contains only the patch source.
