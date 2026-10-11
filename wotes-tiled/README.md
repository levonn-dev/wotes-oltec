# wotes-tiled

Moves Warriors of the Eternal Sun maps between the ROM and Tiled: the outdoor regions both ways, the game's dungeon levels out, dungeon maps of your own in. Go 1.26, standard library only. `docs/tiled.md` explains the workflow; this file is the command reference.

## Commands

| Command | Does |
|---|---|
| `extract --rom PATH --out DIR [--layout FILE]` | writes every outdoor tile set and map into `DIR/rom`: four palette-line tilesets, a metatile definitions map, a metatile tileset and a map per region, plus `flags.tsj`, `solid.tsj`, `city-figures.tsj`, the VIEW MAP pictures under `DIR/rom/area`, a wall sheet per dungeon style and set of wall colours, and under `DIR/rom/dungeons` the side tileset `dungeon-sides.tsj` and every dungeon level. Each map carries the ROM's dungeon entrances, special places, castle figures, fixed fights and spawn areas as object layers; each level its entry points, items, monsters, events, stairs and exits. Byte-identical across runs. |
| `new NAME --base REGION [--dir DIR] [--layout FILE]` | writes `DIR/NAME.tmj`, `DIR/NAME-metatiles.tmj` and `DIR/gen/NAME-metatiles.tsj` for a map built on `city`, `valley`, `jungle` or `swamp`. The map gets the empty object layers `transitions`, `triggers`, `npcs` and `encounters` and the figures tileset `rom/city-figures.tsj`. Refuses to overwrite. |
| `new NAME --dungeon --size WxH --look LOOK [--dir DIR]` | writes `DIR/dungeons/NAME.tmj`, a dungeon map of W by H tiles in one of the seven looks (`cave`, `cave-06`, `azcan`, `azcan-10`, `stone`, `elven`, `elven-11`): the four side layers with every tile walled on all four sides, the empty object layers `entries`, `transitions`, `events`, `items` and `monsters`, and the look's wall sheet as a hidden `legend` layer. Refuses to overwrite. |
| `convert MAP --rom PATH --out FILE [--depfile FILE] [--tilesets-only] [--tile-budget N] [--layout FILE]` | regenerates the map's project tileset, then writes the assembler file, which holds the terrain, the places and, when the map has them, the trigger, text, figure, colour and spawn tables, and `NAME-map.txt` beside it. When the map places entrances, it also reads the dungeon maps they name from `dungeons/` beside the land map, and those their staircases name, and emits the game's entrance, level, directory and stairs tables grown by the map's entrances and the dungeon maps' caves, each cave a level, with the packed grids, object lists and texts. `--depfile` writes a Ninja depfile. `--tilesets-only` stops after the tileset. `--tile-budget` overrides the default of 943. |
| `render (MAP \| --region REGION) --rom PATH --out FILE.png [--plane ground\|front\|both] [--scale 1\|2\|4] [--solid] [--area] [--layout FILE]` | draws a land map or a ROM region; `--solid` tints blocked ground tiles red; `--area` draws a land map's VIEW MAP picture, 208 pixels square in palette line 2. |
| `layout` | prints the built-in layout of the clean ROM as JSON: every table address and count the tool reads and the region, dungeon and style names. `--layout FILE` on the other commands replaces it whole; every field is required. |
| `version` | prints the build version. |

Exit status 1 with one line on stderr on any error; a missing or unknown subcommand, or `-h`, prints the usage and exits 2.

## Layout

| Package | Holds |
|---|---|
| `main` | subcommand dispatch, one `cmd_*.go` per command |
| `internal/rom` | the layout (addresses, counts, names) with the clean ROM's as the default, the ROM's outdoor tables and object tables, the dungeon maps and the unpacker, the dungeon styles and their wall art, VDP words, the palette |
| `internal/tiled` | Tiled's JSON map and tileset types, GID flips, a row-per-line map writer |
| `internal/render` | tiles, metatiles and planes to images |
| `internal/extract` | ROM to Tiled files, the layer, class, kind and property names, the dungeon levels, the side tileset and the wall sheets |
| `internal/dungeon` | a dungeon map read into its caves: the side layers, the walk by the game's step rule, the records and texts of each cave, with the engine's limits |
| `internal/convert` | Tiled map to cell arrays, blocks, metatiles, chunk list and places, the object layers, and the land map's dungeon maps to level, directory, stairs and entrance tables, with the engine's limits |
| `internal/emit` | assembler text, the report, the depfile |
| `internal/scaffold` | `new`, land and dungeon |
| `internal/romtest` | a synthetic ROM image at the real offsets, for tests |

`go test ./...` runs without the ROM. `WOTES_CLEAN_ROM=path go test -run TestRoundTrip ./internal/convert/` extracts each ROM region and converts it back, comparing every tile word. The same run converts each region's extracted spawn areas back (`TestRoundTripOfSpawnAreas`) and compares what every cell rolls with its terrain's list in the ROM. `WOTES_CLEAN_ROM=path go test -run 'TestDungeonsReal' ./internal/rom/` decodes all 38 levels of the ROM. `WOTES_CLEAN_ROM=path go test -run TestRoundTrip ./internal/dungeon/` reads all 38 extracted levels back into the ROM's own grids, entry points, records and texts, and the six beastman levels drawn as one map back into six caves. `WOTES_CLEAN_ROM=path go test -run 'TestCityFiguresReal|TestSpeechRoundTripReal' ./internal/...` proves the figure and speech formats against the ROM's own records; `task tiled:roundtrip` runs every real-ROM test listed in `Taskfile.yml`.
