# wotes-tiled

Moves Warriors of the Eternal Sun outdoor maps between the clean ROM and Tiled. Go 1.26, standard library only. `docs/tiled.md` in the repository root explains the workflow; this file is the command reference.

## Commands

| Command | Does |
|---|---|
| `extract --rom PATH --out DIR` | writes every outdoor tile set and map into `DIR/rom`: four palette-line tilesets, a metatile definitions map, a metatile tileset and a map per region, plus `flags.tsj` and `city-figures.tsj`. Each map carries the ROM's dungeon entrances, special places, castle figures and fixed fights as object layers. Byte-identical across runs. |
| `new NAME --base REGION [--dir DIR]` | writes `DIR/NAME.tmj`, `DIR/NAME-metatiles.tmj` and `DIR/gen/NAME-metatiles.tsj` for a map built on `city`, `valley`, `jungle` or `swamp`. Refuses to overwrite. |
| `convert MAP --rom PATH --out FILE [--depfile FILE] [--tilesets-only] [--tile-budget N]` | regenerates the map's project tileset, then writes the assembler file and `NAME-map.txt` beside it. `--depfile` writes a Ninja depfile. `--tilesets-only` stops after the tileset. `--tile-budget` overrides the default of 943. |
| `render (MAP \| --region REGION) --rom PATH --out FILE.png [--plane ground\|front\|both] [--scale 1\|2\|4] [--solid]` | draws a land map or a ROM region; `--solid` tints blocked ground tiles red. |
| `version` | prints the build version. |

Exit status 1 with one line on stderr on any error; a missing or unknown subcommand, or `-h`, prints the usage and exits 2.

## Layout

| Package | Holds |
|---|---|
| `main` | subcommand dispatch, one `cmd_*.go` per command |
| `internal/rom` | the ROM's outdoor tables and object tables (entrances, special places, figures, fixed fights), VDP words, the palette |
| `internal/tiled` | Tiled's JSON map and tileset types, GID flips, a row-per-line map writer |
| `internal/render` | tiles, metatiles and planes to images |
| `internal/extract` | ROM to Tiled files, the layer, class, kind and property names, the dungeon names |
| `internal/convert` | Tiled map to cell arrays, blocks, metatiles, chunk list and places, with the engine's limits |
| `internal/emit` | assembler text, the report, the depfile |
| `internal/scaffold` | `new` |
| `internal/romtest` | a synthetic ROM image at the real offsets, for tests |

`go test ./...` runs without the ROM. `WOTES_CLEAN_ROM=path go test -run TestRoundTrip ./internal/convert/` extracts each ROM region and converts it back, comparing every tile word.
