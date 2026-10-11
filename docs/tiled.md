# Drawing maps in Tiled

`wotes-tiled` moves maps between the clean ROM and [Tiled](https://www.mapeditor.org/): the outdoor regions both ways, the game's dungeon levels out, dungeon maps of your own in. It extracts every tile set, region map, dungeon level and VIEW MAP picture the game has into Tiled files you can open, study and copy from, and it converts a land map you draw, with its objects, the dungeon maps its entrances name and its area picture, into the tables the engine reads, at build time. This guide is for someone who knows Tiled and not this engine.

## What the engine stores

An outdoor region is two planes of 32 by 32 cells, each cell 192 pixels square, so a map is 6144 pixels square. A cell names a **block**, a block is 8 by 8 **metatiles**, a metatile is 3 by 3 **tiles** of 8 pixels, and a tile is a VDP name-table word: a tile index, horizontal and vertical flips, a palette line and a priority bit. On the **ground** plane the priority bit means "blocked": the party cannot walk onto that 8-pixel tile. On the **front** plane it means "draw over the party": tree canopies and cliff tops. Every outdoor screen uses one palette of four lines of sixteen colours, and colour 0 of a line is transparent.

Tiled edits one level at a time, so the tool uses two map sizes: a **land map** at 24-pixel tiles, where a Tiled tile is one metatile, and a **metatile definitions map** at 8-pixel tiles, where 3 by 3 groups define metatiles. The converter derives the blocks.

## Setup

1. `task bootstrap` builds the tool into `tools/wotes-tiled/` when Go 1.26 or later is present (goenv or https://go.dev/dl/, not apt).
2. Put the clean ROM at the repository root, or point at it with `task configure -- -DCLEAN_ROM=<path>` as the README's Quick start describes, then `task tiled:extract` writes `maps/rom/` from it. `task build` runs the same extraction whenever `maps/rom/` is missing or the ROM or the tool changed, because the land map paints with the extracted tilesets. Nothing in it is committed; delete it and run again whenever you like.
3. `task tiled:install` fetches Tiled 1.12.2 into `tools/tiled/`. `task tiled` opens the Tiled project `maps/wotes.tiled-project`, which lists the `maps/` folder in Tiled's project view and adds the command Regenerate metatiles to Tiled's Commands menu: it saves the current map and runs `convert --tilesets-only` on it, so new metatiles appear in Tiled without leaving it. `task tiled -- maps/rom/valley.tmj` opens one map. Tiled on Windows works as well: the files under `maps/` are plain JSON.
4. `task build` needs Go, because `maps/oltec.tmj` is wired into the build: `configure` fails with the goenv or go.dev hint if Go is missing.

## The files

| Path | What | Committed |
|---|---|---|
| `maps/rom/REGION-tiles-p0.tsj` … `p3.tsj` | a region's tiles on palette line 0 to 3, 32 per row; a tile's ROM index is its position | no |
| `maps/rom/REGION-metatiles.tmj` | the region's metatiles as 3 by 3 tile groups, 64 per row: layers `tiles` and `priority` | no |
| `maps/rom/REGION-metatiles.tsj` | the same metatiles rendered, for painting land maps; each has the property `wotes.solid` | no |
| `maps/rom/REGION.tmj` | the region's map, expanded to metatiles; its `solid` layer shows which 8-pixel tiles of each ground metatile block, as patterns from `solid.tsj` | no |
| `maps/rom/flags.tsj` | the marker tileset: tile 1 is the red cross | no |
| `maps/rom/solid.tsj` | the `solid` layer's patterns, 24 pixels square: pattern `k` crosses the 8-pixel tiles whose bits are set in `k`, bit 0 the top-left tile, bit 2 the top-right, bit 8 the bottom-right | no |
| `maps/rom/city-figures.tsj` | the 13 figures of the castle grounds, 32 by 24 each, from the four tile blocks the city loads over graphics set 0, on palette line 1; the city map's `npcs` layer places them | no |
| `maps/rom/area/REGION.png` | the region's VIEW MAP picture, 208 by 208 pixels in palette line 2's colours, as the camp menu draws it inside its frame; the region map's `wotes.area` names it, `wotes.area_width` and `wotes.area_height` say the land pixels it covers, and on the jungle's and the swamp's `wotes.area_flag` the event flag VIEW MAP waits for | no |
| `maps/rom/dungeons/dungeon-sides.tsj` | the side tileset the dungeon levels draw with: one row per side (north, east, south, west), one column per side value 0 to 15, a bar along the side's edge; each tile carries `wotes.side`, `wotes.value` and, where the value's meaning is known, `wotes.name` | no |
| `maps/rom/dungeons/NN-name.tmj` | dungeon level `NN` (its `ram_map` number in hex) as a top-down map: four side layers over `dungeon-sides.tsj` and its objects, and a hidden `legend` image layer showing the style's wall sheet | no |
| `maps/rom/dungeon-walls-NAME.png` | one sheet per dungeon style and set of wall colours (`cave`, `azcan`, `stone`, `elven`; a style whose levels use more than one set gets `NAME-NN` sheets for the others, `NN` the first such level's number): a row per side value from 1, the value's colour from `dungeon-sides.tsj` as a swatch, then the twelve pieces of the wall as the game stores them with that value's decoration, tile for tile | no |
| `maps/NAME.tmj` | your land map | yes |
| `maps/NAME-metatiles.tmj` | your metatile definitions | yes |
| `maps/dungeons/NAME.tmj` | your dungeon maps, one grid each, that the land map's entrances name ("Dungeon maps of your own" below): the four side layers over `rom/dungeons/dungeon-sides.tsj`, the object layers `entries`, `transitions`, `events`, `items` and `monsters`, and the look's sheet as a hidden `legend` image layer | yes |
| `maps/gen/NAME-metatiles.tsj` | your metatiles rendered, regenerated by `convert` | no |
| `maps/tiles/SHEET.png` and `.tsj` | your new tiles | yes |
| `maps/area/NAME.png` | your land's VIEW MAP picture, 208 by 208, rendered from the map and yours to retouch ("The area map" below) | yes |
| `maps/wotes.tiled-project` | the Tiled project: the `maps/` folder, the object classes the maps use (`Place`, `Transition`, `Trigger`, `Npc`, `Encounter`, `Spawn`, `Entry`, `Event`, `Item`, `Monster`) with their colours, and the Regenerate metatiles command; Tiled writes its session file beside it, which is not committed | yes |

`REGION` is `city`, `valley`, `jungle` or `swamp`. The city and the valley are one map drawn with two tile sets, and the game swaps them by the party's position: it loads the city set when the party enters the rectangle x 0x938 to 0xDF8, y 0xCF8 to 0x1100 (map pixels, around the castle) and the valley set again when the party leaves x 0x8B8 to 0xE78, y 0xC78 to 0x1180; the band between the two is why the switch happens about 120 pixels before the wall. The sets share only tiles 0x00 to 0x31, so `valley.tmj` shows the castle as the valley set draws it, a brown block, and `city.tmj` shows the wilderness as the city set draws it, with brown rivers; each is what the engine displays while that set is loaded. A land map has one tile set, so a town in the valley's style is drawn from borrowed and new tiles, not from a second set.

Every `.tsj` under `maps/rom/` and `maps/gen/` has its image beside it as a `.png` of the same name. Everything under `maps/rom/` is a reference, never edited or copied: `extract` overwrites it on every run, and only the generated assembler refers to it by ROM address, not by copying it. Only your own files are committed: the land map `maps/NAME.tmj`, the definitions map `maps/NAME-metatiles.tmj`, your dungeon maps under `maps/dungeons/`, your area picture under `maps/area/` and your sheets.

## Objects on the extracted maps

Each `maps/rom/REGION.tmj` carries four object layers after `places`, filled from the ROM's own tables. Positions are party pixels, which are the map's pixels: an object at x 0x938 stands where the party's x reads 0x938. Every object has a class, a `wotes.kind` and the ROM's numbers as `wotes.*` properties; a property that does not apply is left out.

| Layer | Objects |
|---|---|
| `transitions` | `entrance` points where a dungeon opens, named for the dungeon (`wotes.map`, `wotes.entry`, `wotes.level` for the least game level that may enter, `wotes.flag` when entering sets an event flag; a record shadowed by an earlier one at the same position is left out); `zone` rectangles that walk the party into an entrance (the jungle's Azcan Caverns and the swamp's Gateway Passage; their `wotes.level` is the place's own level cap); on the city and the valley the two `tileset` rectangles of the castle-wall swap |
| `triggers` | the special places: `trigger` rectangles named for the shop, the person or the effect, or by the first sentence of a speech, with the speech at each game level in `wotes.speech.0` to `wotes.speech.4`; the graveyard's `gravestone` points with their epitaphs |
| `npcs` | the castle grounds' figures as `npc` tile objects from `rom/city-figures.tsj`, named `Figure N` and `at` the trigger they stand by; `wotes.shows` says when the game draws one (`always`, a flag bit, `random`), `wotes.moves` which way a pacing one walks; ten `townsfolk` points: the game draws its four wandering townsfolk from the first eight, the last two are spare |
| `encounters` | the fixed fights: an `encounter` rectangle reaching 96 pixels either side of where the fight starts, named with the count and the monster, and a `monster` point where each one stands; the spawn areas, `spawn` rectangles and polygons that say which random encounters the game rolls where, each with its list in `wotes.monsters` when it rolls any |

The ROM names no dungeon; the entrance names are the official guide's. Everything else is the ROM's own text. These layers are references like the rest of `maps/rom/`; the next section says which of their kinds a land map may carry, and a figure pasted into `places` is refused like any tile object.

The spawn areas are the game's own rule drawn out. The game gives each 192-pixel cell of the valley one of thirteen terrains and each terrain a list of eight entries. `valley.tmj` has an area for every group of neighbouring cells of one terrain, named `terrain N` with the number in `wotes.terrain`; terrain 0, round the castle, and terrain 12 roll nothing and carry no `wotes.monsters`. An outline runs round the outside of its group, so it also covers a group that lies inside it: `terrain 4` holds the two cells of `terrain 12`, and the smaller area holds its ground only while both are on the map, so copy the two together. The city rolls no random encounters, so `city.tmj` has one area without monsters over the whole map. The jungle and the swamp are one terrain each, 11 and 10, and one area over the whole map. The game's own pause and its roll ahead of the party are the defaults, so no area sets `wotes.pause` or `wotes.all_directions`. An area is a land map's own kind: copied into a map of your own it converts as it stands, and `wotes.terrain` is ignored there.

## Objects on a map of your own

`new` gives a land map four empty object layers after `places`, in the vocabulary the extracted maps use, and adds `rom/city-figures.tsj` to the map so figures can be placed. `convert` turns them into tables named after the map; wiring them into a region is the patch's business, and `hooks/hooks.s` shows the shape for region 4.

Six map properties: `wotes.region`, an int, the `ram_region` whose entrance records the map may take (required when `transitions` holds anything; the converter accepts 0 to 255, because a patch adds regions the ROM does not list); `wotes.figure_colours`, a string of sixteen `0x` colour words separated by spaces, the palette line the map's figures are drawn with (required when `npcs` holds anything; a Genesis colour is `0x0BGR` with even digits, `0x0000` to `0x0EEE`); and `wotes.area`, `wotes.area_width`, `wotes.area_height` and `wotes.area_flag`, the VIEW MAP picture, the land pixels it covers and the event flag it waits for ("The area map" below).

| Layer | Objects | Properties |
|---|---|---|
| `transitions` | points, `wotes.kind` `entrance` | where it leads: `wotes.dungeon` and `wotes.to`, a dungeon map of your own by name and one of its entry points by name, or `wotes.map` and `wotes.entry`, one of the game's levels and its entry point by number; `wotes.level`, the least game level that may enter (absent means 0); `wotes.flag`, an event flag set on entering, from the free range `0x44` to `0xF6` (absent or 0 means none); `wotes.index`, optional, a record of the game's entrance table to take |
| `triggers` | unrotated rectangles, `wotes.kind` `shop`, `restore`, `speech` or `event` | `wotes.level`, the highest game level the trigger fires at, 0 to 4 (absent means 4, always); `wotes.shop` 0 to 3 for a shop; `wotes.text` for a restore, a speech or an event, the lines separated by newlines; for an event `wotes.flag`, the event flag that marks it done, from the free range `0x44` to `0xF6`, and optionally `wotes.item`, an item's number, `wotes.picture`, a quest picture 1 to 4, and `wotes.after`, the text shown once the flag is set |
| `npcs` | tile objects from `rom/city-figures.tsj` | the figure is the tile's; `wotes.moves` `stands` (the default), `east-west` or `north-south`; `wotes.slot`, an animation 0 to 3 for a standing figure, 0 to 19 for a pacer; `wotes.front`, a bool |
| `encounters` | unrotated rectangles and polygons, `wotes.kind` `spawn` | `wotes.monsters`, the area's random encounters, a monster a line; `wotes.pause`, the frames between two rolls (absent means the game's own); `wotes.all_directions`, a bool |
| `places` | points and rectangles | each a `NAME_place_<name>` symbol |

A `transitions` point is an entrance: the party walking onto it enters the dungeon it names, and leaving that dungeon by the matching exit puts the party back on the point. Every entrance names where it leads, a dungeon map of your own (`wotes.dungeon`, `wotes.to`; "Dungeon maps of your own" below) or one of the game's levels (`wotes.map`, `wotes.entry`). The converter emits copies of the game's two entrance tables with the map's entrances in them: an entrance with `wotes.index` takes that record of the game's table, which must be a filler (map `0x3F`) or one of the map's own region, and an entrance without one takes the next record after the game's forty, in object id order. The point's position, rounded down to a multiple of 8, the granularity the game compares at, becomes the record's coordinates. Two entrances may lead to the same entry point, but the exit tile behind that entry point leads out by the one entrance its `wotes.to` names, so a party that went in by the other comes out at a mouth it did not enter by.

Triggers are rectangles the party walks into. A `shop` opens the city shop `wotes.shop` names (0 armour, 1 weapons, 2 magic, 3 caravan) and carries no text. A `restore` shows `wotes.text` and then heals and raises everyone, as the jungle's Waters of Life do. A `speech` shows its text and waits for a button. An `event` happens once: it shows its text and, over the open box, plays quest picture `wotes.picture` (1 to 4, the game's four, each an animated portrait that raises the game level to its own, 0, 1, 3 and 4, when the party is below it), plays the picture again and again until a button, which closes it with the text, hands the party `wotes.item` through the game's own routine, which picks the character who may use it and has room in the pack, and sets `wotes.flag`; from then on the rectangle shows `wotes.after`, or nothing without one. When every pack is full the item is not handed over and the event is not marked, so the party hears the dungeon's refusal sound and meets the event again on its next entry: a quest item is never lost. A trigger fires on a walking step into its rectangle, once until the party stands in no rectangle, so a party placed inside one fires it on its next step, and two rectangles that touch do not fire one after the other.

Texts are upper-case: at most 8 lines of 26 characters from the font's set (the space, `!"#$%&'()*+,-./`, digits, `:;<=>?@` and A to Z; no lower case), the lines separated by newlines in the property. An empty line is refused, because it would end the text. The box shows a blank line above and below the text, as the game's own short speeches do.

Figures are tile objects from `rom/city-figures.tsj`. Figures 1 to 12 stand 8 pixels left of and above their position, so place the tile where you want the figure drawn. A figure draws on palette line 3 with the map's `wotes.figure_colours`; colour 1 of that line is the panel's background and stays the game's, so the map's colours 2 to 15 are the ones drawn. `wotes.moves` picks the routine. `wotes.slot` is the record's animation slot. A pacing figure takes 1 to 19 (slot 0 runs no animation) and never shares it with another; a standing figure names the animation it plays, 1 to 3, or 0 for none. Without a slot a standing figure gets 0 and each pacing figure the lowest free slot from 1. `wotes.front` draws the figure in front of every other sprite. To make a figure speak, draw a `speech` rectangle over the tile it stands on. A figure of the extracted city that the game shows only when a flag is set, or at random, cannot be placed: a land map's figures are always shown.

Spawn areas say where random encounters happen. Each is a rectangle or a polygon on `encounters`. The party meets the monsters of the area it stands in, and none where it stands in no area or in one without `wotes.monsters`, so an area without monsters keeps a village quiet inside a larger one. Where areas overlap, the smallest holds the ground. `wotes.monsters` lists up to eight monsters, one a line, as `NAME FEWEST-MOST`: `TIGER 2-6`. The names are the battle roster's and only the monsters the game's roll accepts, `0x62` to `0x93`, may stand in a list; write a monster's number, as `0x86 1-2`, where two share a name. A `#` starts a remark that runs to the line's end, which is how an extracted list names such a monster: `0x80 1-1 # HYDRA`. The game draws from eight slots, so fewer lines repeat in order to fill them. A line may end in `affinity N`, 0 to 5, and `level N`: the first monster picked chooses by its affinity which others may join it (0, the default, fights alone), and level 0 keeps the group from casting spells (the default is 1, as in every list of the game). `wotes.pause` is the frames between two rolls, 60 a second; the game's own is 240 at game levels 0 and 1 and 180 above, and a roll that places no one halves the next wait. The game puts a group 160 pixels ahead of the party and gives the roll up when no way leads from there; with `wotes.all_directions` it tries the party's other seven sides too, which ground broken up by cliffs and plants needs. The converter gives every metatile cell the area over its centre and writes `NAME_spawns`, a record an area (the pause, the every-side word and the list or 0), the lists `NAME_spawn_N`, and `NAME_spawn_grid`, the areas' bounding rectangle in cells and a byte a cell; `hooks/hooks.s` shows how region 4 rolls from them.

### The area map

The camp menu's VIEW MAP draws a picture of the region with a blinking white square at the party's position: 26 by 26 tiles, 208 pixels square, in a gold frame, the party panel beside it. The game's three pictures are extracted as `maps/rom/area/REGION.png`. A land map gets its own through `wotes.area`, a file property naming a PNG of 208 by 208 pixels relative to the map (`area/oltec.png`); without it VIEW MAP answers that the party has no map of the area. The frame, the filler around it and the marker are the game's; the PNG is the picture inside the frame.

The picture's colours are the sixteen of palette line 2: black, three blues (`0x400`, `0xA42`, `0xC84`), two browns (`0x048`, `0x024`), gold `0x08A`, yellow `0x4EE`, two greens (`0x080`, `0x4C4`), two reds (`0x006`, `0x00C`), black again, two grey-blues (`0x422`, `0xA88`) and white, as `maps/rom/area/valley.png` shows them. A transparent pixel is colour 0, the backdrop, and an opaque black pixel is colour 12; any other colour is an error that names the pixel. The converter cuts the picture into tiles and reuses a tile that an earlier one equals in some flip; the picture may hold 582 distinct tiles, the room below the tile the party panel's background is drawn with, so a picture of 676 cells all different, such as dense foliage, is refused. `render --area` prints how many tiles its picture takes.

`task tiled:render -- maps/NAME.tmj --area --out maps/area/NAME.png` renders the picture from the land map: the covered extent of the land at full size, each 208th of it averaged into one pixel, each pixel given the nearest colour of palette line 2. The file is yours afterwards: retouch it in an image editor, or render it again when the land changes. `wotes.area_width` and `wotes.area_height` are the land pixels the picture covers from the map's north-west corner, 2880 for a land of 15 by 15 cells; absent means 6144, the whole grid, as the valley's picture. The game scales the party's position by a divisor each way, position times 8 over the divisor, which the converter derives from the coverage.

VIEW MAP waits for the event flag in `wotes.area_flag`, a map property from `0x1` to `0xFE`, as the game's jungle and swamp do: their extracted maps carry `0x43` and `0x1E`. An event whose `wotes.flag` is that flag hands the party the map. Without the property the map is there from the start, as the valley's and the city's are.

The converter writes `NAME_area_tiles`, a chunk list of the game's shared tiles and the picture's, `NAME_area_plane`, the packed plane, `NAME_area_scale_x` and `NAME_area_scale_y`, the divisors, and `NAME_area_flag`, the map's `wotes.area_flag`, 0 for from the start and 255 for no picture; `hooks/hooks.s` shows how region 4 draws from them.

`maps/oltec.tmj` is a working example: `wotes.region` 4, eighteen entrances, the Tradeway's landing and the seventeen mouths of `dungeons/oltec_caves.tmj`, three shop rectangles, a restore, six speeches, an event with an item, a picture and an after-text, an event that sets the flag its area map waits for, six figures, two of them pacing, and three spawn areas: the land with its eight monsters, and two places with none; its area map is `maps/area/oltec.png`, covering 2880 by 2880.

The extracted maps' object layers describe the ROM's own systems and carry kinds a land map has no use for (`zone`, `tileset`, `trigger`, `gravestone`, `townsfolk`, the fixed fights), so an extracted map no longer converts as it stands; `render --region` draws a ROM region straight from the ROM, and copying terrain from an extracted map into your own is unaffected, and so is copying a spawn area.

## Dungeons on the extracted maps

`extract` also writes the 38 dungeon levels, one map per level under `maps/rom/dungeons/`, named by the level's `ram_map` number and the guide's name: `0c-oltec-tradeway-level-3.tmj`. Levels that share one wall grid, the six beastman caves and the eight castle interiors, still get a file each, because their objects differ.

A level is drawn top-down, one 16-pixel tile per dungeon tile. The engine stores four side bytes per tile, north, east, south and west, and the map holds them on four tile layers of the same names, each cell showing the tile of that side's value from `dungeon-sides.tsj`: nothing for 0, otherwise a coloured bar along that edge. Stacked, the four layers read like the guide's room drawings. The tileset names the values whose meaning is known in each tile's `wotes.name`: 1 a wall, 2 to 6 a decorated wall, 7 a door and 8 a secret door. Values 9 and 10 let the party through but keep only their number, and a bar's colour tells the values apart. The map's properties carry the level's number and name, its look (`wotes.look`, one of the seven looks a level can have: `cave`, `cave-06`, `azcan`, `azcan-10`, `stone`, `elven`, `elven-11`, named after the style and, where a style has two sets of colours, the first level that uses the second), its style's name (`wotes.style`) and the two pointers of its map table record the style does not cover (`wotes.graphics`, `wotes.grid`).

A step tests one side: the side of the tile stepped onto that faces the party. 0 and 7 to 10 let the party on, 1 to 6 refuse it, and the grid wraps, so a step off the east edge lands on the west column. The game's own levels put their walls on the rock: a rock tile carries value 1 on each face a passage touches, and the floor tile beside it leaves that face open. A level's exits sit on rock tiles behind the entry points, showing a dark opening (value 7) towards the entry: the party steps onto that tile to leave.

The objects are points at tile centres, in the order the ROM lists them:

| Layer | Objects |
|---|---|
| `entries` | one `Entry` per entry point, named `entry N facing west`, with `wotes.entry` and `wotes.facing` (0 north to 3 west) |
| `transitions` | `stairs` named for the level they lead to, with `wotes.map` and `wotes.entry` for where the party lands; `exit` points named `out to REGION (entrance N)`, with the entrance's index and its landing position in party pixels as `wotes.landing` |
| `events` | one `Event` per event record, named by the first sentence of its text or by `action N`, with `wotes.action`, `wotes.amount`, `wotes.uses`, `wotes.extra`, `wotes.gate`, `wotes.flag` and the whole text in `wotes.text` |
| `items` | one `Item` per item record, named from the item table, with `wotes.item` and the same five fields |
| `monsters` | one `Monster` per monster record, named from the monster table when the value has a name and `monster N` otherwise, with `wotes.monster`, `wotes.facing` (the way it stands, 0 north to 3 west) and the five fields; the one `actor` record rides here |

Every object carries `wotes.kind` (`entry`, `stairs`, `exit`, `event`, `item`, `monster` or `actor`), the tile as `wotes.x` and `wotes.y`, and, apart from entry points, its ten ROM bytes as `wotes.record` and its kind byte as `wotes.kind_byte`, so nothing is lost whatever a later reading of the bytes says. A record of any kind that names a text carries it in `wotes.text`, the lines separated by newlines. A side value above 15 has no tile and stops the extract with the level and tile named.

What a side value looks like from inside is on the style's sheet, `dungeon-walls-NAME.png`, and every level carries its style's sheet as a hidden image layer named `legend`: switch the layer on and the sheet appears one tile to the right of the grid. A style whose levels use more than one set of wall colours has one sheet per set, the extra ones named after the first level that uses them (`dungeon-walls-cave-06.png`), and each level's legend points at the sheet in its own colours. Each row of the sheet is one side value, starting with a swatch in the colour the overhead map draws that value with, followed by the twelve pieces the game cuts the wall into, the seventh (12 by 7 tiles) being the largest. Row 1 is the bare wall; a decorated value shows the same wall with its overlay; a door row shows the doorway; a value whose overlay draws nothing in that style, like the secret door in three of the four, looks the same as row 1. A row shows the pieces as the game's renderer lays them, so a piece whose decoration reaches its last tile ends in a bare one. The sheets are flat: no perspective, no distance.

`wotes.monster` on a dungeon monster is the battle type, the roster id its name comes from, the same meaning as on the outdoor encounter objects; the raw record byte is in `wotes.record`.

The levels are a reference, like the rest of `maps/rom/`; the next section says how a dungeon map of your own, drawn in the same vocabulary, becomes levels of the game.

## Dungeon maps of your own

`tools/wotes-tiled/wotes-tiled new NAME --dungeon --size WxH --look LOOK` writes `maps/dungeons/NAME.tmj`: the four side layers over `rom/dungeons/dungeon-sides.tsj` with every side of every tile a wall, the empty object layers `entries`, `transitions`, `events`, `items` and `monsters`, and the look's sheet as a hidden `legend` layer. The name is letters, digits and underscores, like a land map's; the look is one of the seven; the size is the grid's, at most 1360 tiles and 64 a side. One 16-pixel tile is one dungeon tile.

Dig a tile by clearing its four sides. A passage between two dug tiles needs nothing more: the walls stand on the rock round them, as in the game's own caves, so the rock keeps its four walls and a floor tile's faces stay open. Paint a side only where a wall, a decoration, a door or a crack is meant: 1 a wall, 2 to 6 the look's decorated walls (the `cave-06` look has a dark opening, a crack, rubble, iron bars and bare rock), 7 a door, 8 a secret door, 9 and 10 passable decorations. A side value the look has no art for is refused. Where a passage ends at a tile that must stop the party without a wall's look, give the rock tile's face a decoration; to let the party through a decorated face, use 7 to 10.

A step and the view use one side, the face of the tile ahead that is turned towards the party, so a face painted between two floor tiles makes a wall where no rock is, and the two faces of one passage need not agree. Between a floor tile A and the floor tile B east of it:

| Faces | What the party meets |
|---|---|
| B's west 8, A's east 8 | a secret door both ways: it is drawn as the bare wall in the cave, Azcan and stone looks (the elven look draws it as a door), the party walks through it, the automap shows a wall, and after a step or a turn that faces it a living character of class 6 finds it one time in six (`NAME FOUND A SECRET DOOR`); finding it changes nothing |
| A's east 8, B's west 1 | a one-way secret door: from B the party walks west through what looks like rock; from A the way east is a wall |
| A's east 1, B's west 0 | a one-way passage: from A the party sees B and walks in; from B the way back is a wall |
| both faces 1 | a partition: a wall with no rock behind it |
| 9 or 10 | a crack or rubble the party walks through, drawn like the decoration |

The game's own levels hold 149 two-way secret doors and 13 one-way ones. No moving monster passes a face above 0, so a secret door also keeps a room's monsters inside it. The cave walk follows the same rule: a secret door joins the caves on its two sides into one level, a one-way face joins them from the side that can enter.

The objects take the form the extracted levels have, one object a record, so a level copied from `maps/rom/dungeons/` converts as it stands and a map of your own reads like one: a monster is a point named after the monster with its battle type in `wotes.monster`, three cave bears on a tile are three such points, an item is named after the item with its number in `wotes.item`, and an event takes the first sentence of its text as its name. Each object carries a `wotes.kind`; the properties the extractor writes only for reference (`wotes.x`, `wotes.y`, `wotes.record`, `wotes.kind_byte`, `wotes.map`, `wotes.entry`, `wotes.region`, `wotes.landing`) are ignored.

| Layer | `wotes.kind` | Properties |
|---|---|---|
| `entries` | `entry` | the object's name, unique in the map, which entrances and staircases lead to it by; `wotes.facing`, 0 north to 3 west |
| `transitions` | `exit` | `wotes.to`, the name of an entrance on the land map being converted, or `wotes.entrance`, the number of one of the game's own entrance records |
| `transitions` | `stairs` | `wotes.to`, the name of an entry point, and `wotes.dungeon` when that entry point is in another dungeon map; or `wotes.value`, one of the game's own stairs records by number |
| `events` | `event` | `wotes.action` 0 (a trap), 1 (a reward or a message), 2 (a quest picture) or 3 (an item gate); `wotes.amount`, `wotes.uses`, `wotes.extra`; `wotes.text`, the lines separated by newlines |
| `items` | `item` | `wotes.item`, the item's number in the game's table, as the extracted levels carry it |
| `monsters` | `monster` | `wotes.monster`, the battle type number, `0x13` to `0x61`; `wotes.facing`; `wotes.amount`, its hit points, and `wotes.uses`, its behaviour (below), which the game's most frequent record of that type supplies when absent |
| `monsters` | `actor` | the fields the extractor wrote, passed through |

A monster's `wotes.uses` is two nibbles: the high one is how many tiles away it notices the party (the two axes' distances added, 1 to 6 in the game's levels), the low one what it does. 1 stands where it is and turns to face the party within that distance; 2 stands until the party comes within it, then hunts the party down; 3 paces back and forth from its tile and hunts from within the distance; 4 wanders its room, turning where it is blocked, and hunts from within the distance. The game's levels use 1 and 2 only (`0x32`, three tiles then the hunt, is the commonest; its trolls are `0x11` and its gargoyles `0x31`, standing); 3 and 4 are in the engine and work. No monster passes a door or a secret door on its own, so a room's monsters stay in it whatever they do.

Every event, item, monster and transition may carry `wotes.gate` and `wotes.flag`, event flags from the free range `0x44` to `0xF6` (0 for none; an item gate's flag may be `0xFF`). A text is upper-case, at most 9 lines of 26 characters from the dungeon font (bytes `0x20` to `0x61`: the space, punctuation, digits and A to Z; the last seven bytes, `[`, `\`, `]`, `^`, `_`, the backquote and a lower-case `a`, draw the font's icons, so an `a` is accepted and shows an icon, not a letter); equal texts on one level share one entry.

A cave is the tiles a party reaches from an entry point by the step rule above, the grid's wrap included. A tile with an exit or a staircase is reached and never stepped beyond, since stepping onto it takes the party off the level, unless an entry point stands on it. Entry points whose reaches meet share a cave, and a dug passage between two caves makes them one. Each cave becomes one level of the game over the map's one grid: a record of the map table with the look's style and graphics, an object list and a text table of its own. Tiles no entry point reaches are rock, whatever their sides, and an object on one is refused. A cave's entry points are numbered in object id order; its records are written in object id order too, which is the ROM's order on an extracted level and the order of drawing on a new one. Caves are numbered by their first entry point, dungeon maps by name, and the first new level is the count of the game's own, 38.

The build finds the dungeon maps through the land map: `convert` reads `dungeons/NAME.tmj` beside the land map for every map its entrances name, and every map their staircases name, and emits the levels under the land map's name: `NAME_levels` and `NAME_level_directory`, the game's 38 records followed by the new ones; `NAME_stairs`, the game's thirteen records followed by one for each entry point a staircase lands on; the entrance tables of "Objects on a map of your own"; one packed grid per dungeon map, `NAME_grid_<map>`; and each cave's object list and texts, `NAME_level_<N>_objects` and `NAME_level_<N>_texts`. The report's block for each dungeon map gives its size in tiles, its look and the packed grid's bytes, then one line a cave: its level number, its entry points by name, its tiles, its records of the 127 a level places, its texts, how many different monster pictures it shows, and the flags it tests and sets; then how many levels, stairs records and entrances the game and the map use of their limits. A cave's tiles include the exit tiles behind its mouths, which a party steps onto to leave.

`maps/dungeons/oltec_caves.tmj` is a working example: 40 by 34 tiles in the `cave-06` look, five caves with seventeen entry points named as the land's places, each with its exit on the rock tile behind it, and monsters, finds and texts on every layer the table above names.

## The layout file

Everything the tool knows about where the ROM keeps its tables, and what to call the regions and dungeons, is a layout: the four region records, the outdoor object tables, the terrain cells and the random-encounter lists, the palette, the name tables, the dungeon map table, directory and stairs, the dungeon styles and the wall-piece tables, and the 38 dungeon names. The clean ROM's layout is built in. `tools/wotes-tiled/wotes-tiled layout` prints it, and `--layout FILE` on `extract`, `new`, `convert` and `render` replaces it whole, so another project can point the tool at a ROM whose tables moved or grew, or rename what the maps show. `dungeons.styles` names each look and its descriptor, and `dungeons.pieces` names the tables that cut the wall strip; renaming a style renames its sheet and every map's `wotes.style`.

The file is JSON. Addresses and lengths are strings in `0x` hex, counts are numbers, and every field is required: an unknown field, a missing or zero field, an address at or past the ROM's length, or a table running past the ROM's end is an error naming the field. Edit a dumped copy rather than writing one from scratch. What the file does not hold stays fixed in the tool: the layer, class and property names of the maps, the engine's limits, and the assembler it writes.

## Drawing a map

1. `tools/wotes-tiled/wotes-tiled new NAME --base valley`. The name is letters, digits and underscores, not starting with a digit, because it prefixes the generated assembler symbols. `--base` sets the map's `wotes.base` property, naming the region the map builds on: that region's metatiles come from the ROM at build time and its tile set loads first into VRAM, free of borrowing cost. Every other region's tiles are borrowed, which costs VRAM.
2. Open `maps/NAME.tmj`. It has the layers `ground`, `front`, `solid` and `places`, then the empty object layers `transitions`, `triggers`, `npcs` and `encounters`, and the tilesets `rom/valley-metatiles.tsj`, `rom/flags.tsj`, `rom/city-figures.tsj` and `gen/NAME-metatiles.tsj`.
3. Paint `ground` from the metatile tileset. Every cell must hold a metatile: an empty ground cell is an error, because the engine would read it as blank and walkable. Paint canopies and cliff tops on `front`; empty there means nothing drawn.
4. The `solid` layer is the map's blocking. A ground cell blocks exactly the 8-pixel tiles its mark covers, whatever its metatile blocks in the ROM, and a cell without a mark blocks nothing. Put the red cross from `flags.tsj` over a cell to block all of it, or a pattern from `rom/solid.tsj` to block some of its nine tiles; add that tileset to the map before you define any metatiles, as step 8 says. The converter derives a copy of the metatile with that blocking, once per distinct metatile, and reuses the base region's own metatile when the table already holds it. The extracted maps' `solid` layers show the ROM's blocking in these patterns, so terrain copied from one together with its `solid` layer blocks as it does in the ROM; copied without it, it blocks nothing.
5. Add points and rectangles on `places` and name them: the cave landing, a shrine, a gate. A place name is letters, digits and underscores. Each becomes a symbol `NAME_place_<name>` holding left, top, right, bottom in party pixels.
6. Place the map's objects: entrance points on `transitions`, shop, restore, speech and event rectangles on `triggers`, figures from `rom/city-figures.tsj` on `npcs`, spawn areas on `encounters`, and set `wotes.region` and `wotes.figure_colours` in Map > Map Properties. "Objects on a map of your own" has the properties.
7. Give the land a VIEW MAP picture: set `wotes.area_width` and `wotes.area_height` to the land pixels it covers, run `task tiled:render -- maps/NAME.tmj --area --out maps/area/NAME.png`, set `wotes.area` to `area/NAME.png`, and set `wotes.area_flag` to the flag of the event that hands the party the map ("The area map" above).
8. To paste metatiles from another region, add its `rom/REGION-metatiles.tsj` to the map first (Map > Add External Tileset): add every region tileset you will borrow from before you define any metatiles, so the generated `gen/NAME-metatiles.tsj` stays the last tileset of the map. Paint from the added tileset; the converter borrows the tiles those metatiles need. Add a region tileset later, once the project tileset has already grown, and `convert` reports the tilesets overlap; the fix is to reopen the map in Tiled and save it, which renumbers the cells.
9. Flipping a metatile on a land layer is an error; define the flipped form in your metatile definitions instead, so a mirrored feature, such as the right half of a symmetric ornament, is its own metatile.
10. Add `wotes_tiled_map(NAME)` to `CMakeLists.txt`, after the `add_executable(patch.elf ...)` line; a second argument caps the map's tiles below the 943 of "Limits" when the region keeps other things in VRAM, as `wotes_tiled_map(oltec 760)` does. From then on `task build` converts `maps/NAME.tmj` and links its tables. Until then, check the map by hand: `tools/wotes-tiled/wotes-tiled convert maps/NAME.tmj --rom build/m68k/base.md --out build/NAME-map.s` (or `--tilesets-only`).

## Defining metatiles

Open `maps/NAME-metatiles.tmj`. It is 192 tiles wide: metatile `k` occupies columns `(k mod 64) * 3` to `+2` and rows `(k div 64) * 3` to `+2`. On `tiles`, paint each of the nine tiles from a palette-line tileset (`rom/valley-tiles-p2.tsj` is the valley's tiles as they look on palette line 2) or from a sheet of your own; flip them with X and Y in Tiled. On `priority`, put the red cross over each tile that draws over the party where the metatile is used on `front`; on `ground` the land map's `solid` layer alone says what blocks. A metatile is either empty or complete: all nine tiles or none. A metatile used on `front` needs transparent tiles where the ground must show through. Run `task build`, or `tools/wotes-tiled/wotes-tiled convert maps/NAME.tmj --rom build/m68k/base.md --tilesets-only`, and Tiled reloads `gen/NAME-metatiles.tsj` with your metatiles rendered.

## New tiles

Draw a PNG whose width and height are multiples of 8, save it as `maps/tiles/SHEET.png`, and create a tileset over it in Tiled (New Tileset, 8 by 8, saved as `maps/tiles/SHEET.tsj`). Add it to your definitions map, last, and paint from it. Colours: transparent pixels are colour 0; every other pixel must match one of colours 1 to 15 of a single palette line. Use the extracted tilesets as your colour reference, or the table: a Genesis channel value `v` from 0 to 7 is `round(v * 255 / 7)`, so 0, 36, 73, 109, 146, 182, 219 or 255 per channel. A tile whose colours fit no line, or fit two lines, is an error naming the sheet and the tile. A fully transparent sheet tile fits every palette line and is refused for the same reason: blanks come from the base set's tiles, not from a sheet.

## Limits

| Limit | Value | Why |
|---|---|---|
| blocks | 512 | the engine shifts the index by 7 inside a 16-bit word |
| metatiles | 3640 | the engine multiplies the index by 18 inside a 16-bit word |
| tiles | 943 (`0x3AF`) | the region's tiles load at VRAM 0 and the next set loads at `0x75E0`; `--tile-budget`, the second argument of `wotes_tiled_map`, lowers it: the Oltec map takes 760 (`0x2F8`), because the game's lava check burns the party on tiles `0x2F8` to `0x2FC` in any region and the walking monsters' frames load from `0x319` |
| a dungeon grid | 1360 tiles, 64 a side | the engine unpacks a grid into 1360 slots of four side bytes, and the text ring sits right after them |
| records on a level | 127 | the engine's object table has 127 slots, shared with what appears in play |
| entry points on a level | 86 | the engine finds an entry point at three times its index in a byte; from entry 86 the high byte of that offset lands in the party's x |
| entrances | 128 | an exit names its entrance in a byte as `0xFF` minus the index |
| stairs records | 128 | a staircase names its record in a byte below `0x80` |
| levels | 256 | an entrance and a staircase name a level in a byte |
| a dungeon text | 9 lines of 26 characters, bytes `0x20` to `0x61` | the text box shows 26 columns and keeps ten lines of log; the dungeon font has no lower case |
| monsters | battle types to `0x61` | a record holds the type less `0x12` in a byte up to `0x4F` |
| monster pictures in view | 8 | the engine keeps eight picture slots |
| event flags of your own | `0x44` to `0xF6` | the game's own flags take the rest; the layout carries this ROM's range |

The converter's report, written beside the assembler as `NAME-map.txt`, shows where you stand.

## What the build does

Calling `wotes_tiled_map(NAME)` in `CMakeLists.txt`, after `add_executable(patch.elf ...)`, converts `maps/NAME.tmj` into `gen/NAME-map.s` in each build tree (`build/m68k/gen/NAME-map.s` for the clean ROM) and links it into `patch.elf` whenever the map, its definitions, its tilesets, its sheets, its dungeon maps or its area picture change; a second argument is passed on as `convert`'s `--tile-budget`. Before any tree builds, `task build` runs `tiled:extract` when `maps/rom/` is missing or the ROM or the tool changed, since the map's tilesets come from there. The file exports `NAME_front`, `NAME_ground`, `NAME_blocks`, `NAME_metatiles`, `NAME_tiles` and one `NAME_place_*` per place, and the area map's `NAME_area_tiles`, `NAME_area_plane`, `NAME_area_scale_x`, `NAME_area_scale_y` and `NAME_area_flag`, with the flag 255 when the map has no picture. When the map has objects, the file also exports `NAME_triggers`, records of `y0 y1 x0 x1` words in party pixels (inclusive), a level byte, a kind byte (1 shop, 2 restore, 3 speech, 4 event) and a long argument (the shop index, the text's address, or an event's block `NAME_event_N`: the text's address, the after-text's or 0, then the picture, the item and the flag as bytes), ended by a `0xFFFF` word; `NAME_texts`, the message texts in the game's shape (a rows byte counting the lines plus two, a zero byte the drawer shows as an empty first line, NUL-terminated lines, an empty line), each text labelled `NAME_text_N`; `NAME_figures`, the game's own 28-byte figure records ended by a zero long, whose parameter long selects palette line 3 and, for a figure with `wotes.front`, sets bit 31 so it draws in front of every other sprite; `NAME_figure_colours`, sixteen colour words; `NAME_spawns`, `NAME_spawn_N` and `NAME_spawn_grid`, the spawn areas; and, when the map places an entrance, `NAME_entrance_regions` and `NAME_entrance_points`, the game's entrance tables with the map's entrances written in and added, with `NAME_levels`, `NAME_level_directory` and `NAME_stairs`, the game's level, directory and stairs tables grown by the caves of the dungeon maps the entrances name, a packed grid `NAME_grid_<map>` per dungeon map and `NAME_level_<N>_objects` and `NAME_level_<N>_texts` per cave. A table appears only when its layer holds objects. The file contains no ROM bytes: the base region's metatiles and the game's own entrance, level and stairs records come in through `.incbin` from the build tree's copy of the ROM, and borrowed tiles are chunk-list entries pointing into the ROM. Wiring the symbols into a region, the region table, the graphics-set arm and any hook, is hand-written; `src/oltec.s` shows the shape. `hooks/hooks.s` shows how a patch reads the tables for region 4: the special-places routine walks `NAME_triggers` in place of the castle's rectangles, the city figures' record loop draws `NAME_figures`, the outdoor fade-in writes `NAME_figure_colours` to palette line 3 (only on the outdoor fade-ins, when the game's figures-drawn flag is set, because the camp and shop screens draw their stone with that line), the picture drawer's loop replays an event's picture until a button, the dungeon loaders read the map's copies of the entrance, stairs and level tables, the encounter roll reads `NAME_spawns` and `NAME_spawn_grid`, and VIEW MAP draws from the `NAME_area_*` symbols.

## Errors

Every error names the file, and when a cell is involved the layer and its coordinates, `(x 41, y 17)`, as Tiled shows them in its status bar.

| Message | What to do |
|---|---|
| `empty ground cell` | paint the cell |
| `is not a metatile` | paint from a metatile tileset; the solid marker goes on the `solid` layer, not `ground` or `front` |
| `a flipped metatile` | define the flipped form in the definitions map |
| `only the solid marker from the flags tileset or a pattern from the solid tileset goes on this layer` | `solid` takes the red cross or a pattern from `rom/solid.tsj` |
| `only the solid marker from the flags tileset goes on this layer` | `priority` takes only the red cross |
| `a flipped pattern` | clear the flip; paint the pattern that marks the tiles you mean |
| `a priority marker over an empty tile` | paint the tile under the cross, or clear the cross; a blocked blank comes from the `solid` layer of the land map |
| `tilesets … overlap` | open the map in Tiled and save it; keep generated tilesets last |
| `not an 8-pixel tile` | the definitions map takes palette-line tilesets and sheets, not metatiles |
| `metatile N has M of nine tiles` | fill all nine or clear them |
| `colours fit no palette line` / `fit palette lines [..]` | repaint the tile in one line's colours; add a colour only that line has |
| `pixel N has alpha A, use 0 for transparent or 255 for opaque` | repaint the pixel fully transparent or fully opaque |
| `a sheet is a multiple of 8 in both directions` | resize the PNG |
| `the budget is N` | fewer borrowed or new tiles, or a different base |
| `N blocks; the engine allows 512` | repeat arrangements: fewer distinct 8 by 8 groups |
| `N metatiles (…); the engine allows 3640` | reuse an existing metatile instead of defining another |
| `tile N has a diagonal flip` | the VDP has no diagonal flip; clear it in Tiled |
| `tile N has a hexagonal rotation flag` | the VDP has no such flag; clear it in Tiled |
| `tile N is past the end of tileset SOURCE` | repaint the cell from a tile that exists in that tileset |
| `tile N is past the sheet's M tiles` | repaint the cell from a tile inside the sheet's PNG |
| `GID N belongs to no tileset of this map` | re-add the tileset the cell was painted from |
| `tileset SOURCE: unknown wotes.kind "…"` | a tileset carries the `wotes.kind` that `extract` wrote, or none for a sheet of your own; run `extract` again, or clear the property |
| `two places are called "NAME"` | rename one of the places |
| `only points and rectangles, unrotated and not from a template, are places` | redraw the place as a point or a rectangle: no ellipse, polygon, polyline, text or tile object, rotation 0, and detached from any template |
| `a place is a point or a rectangle, not a line` | redraw the polyline as a point or a rectangle |
| `place NAME … is outside the 6144-pixel map` | move or resize the place to fit the map |
| `a land map is 256 by 256 tiles of 24 pixels; this one is …` | start the map from `new`, or resize it to 256 by 256 in Tiled |
| `a land map has the tile layers ground, front and solid and the object layer places` | name the layers so, or start the map from `new` |
| `layer NAME must be a tile layer` | make `ground`, `front` and `solid` tile layers |
| `layer NAME is W by H, the map is 256 by 256` | resize the layer to the map in Tiled |
| `a definitions map has 8-pixel tiles and a width and height that are multiples of 3` | start the definitions map from `new`: 8-pixel tiles, 192 tiles wide, a height that is a multiple of 3 |
| `a definitions map has the tile layers tiles and priority` | name the two layers so |
| `layer NAME: N tiles for a W by H map` | resize the layer to the map in Tiled |
| `no region called "NAME": use city, valley, jungle or swamp` | fix the region name in `--base`, `--region`, or a pasted tileset |
| `neither an extracted tileset nor an 8-pixel sheet` | use an extracted tileset, or make the sheet's tiles 8 by 8 |
| `a sheet must not use a ROM tile image` | point the sheet at your own PNG under `maps/tiles/`; paint ROM tiles from `rom/REGION-tiles-pL.tsj` in the definitions map |
| `the project tileset of NAME is gen/NAME-metatiles.tsj` | point the tileset reference at that path |
| `open PATH: no such file or directory` | run `extract` again, or check the tileset's path |
| `must be letters, digits and underscores` | rename the map, or the place |
| `tile data must use the CSV encoding` | Map > Map Properties > Tile Layer Format: CSV |
| `an entrance needs the map property wotes.region` | set `wotes.region` in Map > Map Properties |
| `wotes.region N must fit a byte, 0 to 255` | use the `ram_region` number |
| `an entrance is a point object` | redraw the entrance as a point |
| `wotes.index must be a whole number, a record of rom_region_exit_points` | write the record's number, or clear the property to take the next record after the game's |
| `entrance N is outside the M records of rom_region_exit_points` / `entrance N is past the end of rom_region_exits, which holds K records` | take one of the game's forty records, 0 to 39 |
| `PROPERTY must be a whole number from 0 to N` | write a whole number in that range: an entrance's `wotes.level`, `wotes.flag` or `wotes.entry`, a dungeon object's `wotes.facing`, `wotes.amount`, `wotes.uses`, `wotes.extra`, `wotes.gate` or `wotes.flag` |
| `wotes.map must be a whole number, one of the game's levels` / `map 0xNN is outside the 38 maps` | write a level's number, `0x0` to `0x25`, as the files under `maps/rom/dungeons/` are numbered |
| `map 0xNN (NAME) has N entry points: wotes.entry must be a whole number from 0 to M` | pick an entry point the level has; the extracted level lists them on `entries` |
| `MAP.tmj has no entry point called NAME` | set the entrance's `wotes.to` to an entry point of the dungeon map its `wotes.dungeon` names |
| `entrance N belongs to region R, not the map's region M` | take a record of the map's region or a filler record |
| `entrance N is placed twice` | one point per record |
| `a trigger is an unrotated rectangle, not from a template` | redraw it as a plain rectangle |
| `wotes.kind "…" is not shop, restore, speech or event` | set the trigger's kind to one of the four |
| `wotes.level N is outside 0 to 4` | use 0 to 4 |
| `a shop needs wotes.shop from 0 to 3` | set the shop: armour, weapon, magic or caravan |
| `a shop has no wotes.text` | clear the text |
| `a speech needs wotes.text` / `a restore needs wotes.text` / `an event needs wotes.text` | write the lines, separated by newlines |
| `an event needs wotes.flag, the event flag that marks it done, from 0x44 to 0xf6` | set the flag; the dungeon maps' events show which are taken |
| `wotes.flag 0xNN is one of the game's own flags; an event sets one from 0x44 to 0xf6` | use a flag from the free range |
| `wotes.picture N is outside 1 to 4, the game's quest pictures` | use 1 to 4, or clear the property |
| `item 0xNN: the game's items run 1 to 0xeb` | write an item's number; the extracted levels under `maps/rom/dungeons/` show each item's beside its name |
| `wotes.after: …` | the after-text follows the text rules; the rest of the message is one of the four below |
| `line N is empty, which would end the text` | remove the empty line or put a character on it |
| `is not a character the font draws` | use the space, punctuation, digits and A to Z; no lower case |
| `line N is M characters, the box shows W` | shorten the line |
| `N lines, the box shows L` | cut the text to 8 lines |
| `a figure is a tile object from the figures tileset` | place a tile of `rom/city-figures.tsj` |
| `a flipped figure` | clear the flip; the game draws every figure one way |
| `shown "…"; a land map's figures are always shown` | clear `wotes.shows` |
| `wotes.moves "…" is not stands, east-west or north-south` | use one of the three |
| `wotes.slot N is outside 0 to 19` | use a slot from 0 to 19 |
| `wotes.slot N is held by two figures` | give each figure its own slot, or none |
| `wotes.slot 0 runs no animation, so a pacing figure needs a slot from 1 to 19` | use a slot from 1 to 19, or clear it |
| `wotes.slot N on a standing figure names an animation, 1 to 3` | use a slot from 1 to 3, or clear it |
| `a figure is an unrotated tile object` | clear the rotation |
| `the name "…" holds a character outside space to ~` | rename the object with printable ASCII |
| `layer X must be an object layer` | make `places`, `transitions`, `triggers`, `npcs` and `encounters` object layers |
| `wotes.kind "…" is not spawn` | a land map's `encounters` hold spawn areas; set the kind to `spawn` |
| `a spawn area is an unrotated rectangle or polygon` | redraw it as a plain rectangle or polygon |
| `a polygon needs three corners or more` | add corners |
| `the corner (x, y) is outside the 6144-pixel map` | move the corner inside the map |
| `wotes.monsters line N: "…" is not NAME FEWEST-MOST` | write the line as `TIGER 2-6` |
| `no monster the game rolls is called "…"` | use the battle roster's name of a monster from `0x62` to `0x93`, or its number |
| `"…" names monsters 0xNN and 0xMM` | write the one you mean as its number |
| `"…" is not a monster's number` | a monster's number is written in `0x` hex, as `0x86 1-2` |
| `monster 0xNN is outside 0x62 to 0x93` | pick a monster the game's roll accepts |
| `N to M monsters; write 1 to 15, the fewest first` | write the fewest and the most appearing, 1 to 15 |
| `affinity N is outside 0 to 5`, `level N is outside 0 to 255`, `"…" is not affinity N or level N`, `affinity "…" is not a number` (also `level`) | end the line with `affinity N`, `level N`, both or neither |
| `N monsters; a list holds 8` | cut the list to eight lines |
| `wotes.pause N is outside 1 to 32767 frames` | use 1 to 32767, or clear the property for the game's own pause |
| `wotes.all_directions is a bool` | make the property a bool |
| `N spawn areas; a map holds 255` | fewer areas |
| `figures need the map property wotes.figure_colours` | set the map's sixteen colour words |
| `wotes.figure_colours holds N colour words, a palette line has 16` | write sixteen words |
| `is not a Genesis colour word` | use `0x0000` to `0x0EEE`, even digits |
| `wotes.shop is for shops` | clear `wotes.shop` on a restore, a speech or an event |
| `wotes.flag is for events` (also `wotes.item`, `wotes.picture`, `wotes.after`) | clear the property, or make the trigger an event |
| `is outside the 6144-pixel map` (an entrance, a trigger from (x, y) to (x, y), a figure) | move the object inside the map |
| `more pacing figures than the 19 slots the game keeps` | fewer pacing figures, or make some stand |
| `a tile of SOURCE is not a figure` | place figures from `rom/city-figures.tsj` |
| `no record of rom_city_npcs draws figure N` | place a figure one of the city's records draws; the extracted city map's `npcs` layer shows which |
| `wotes.kind "…" is not entrance` / `is not npc` | a land map's `transitions` hold entrance points and its `npcs` figures; only a figure may clear the kind, and an entrance point without `wotes.kind` is refused |
| `an entrance names where it leads: a dungeon map and its entry point in wotes.dungeon and wotes.to, or one of the game's levels and its entry point in wotes.map and wotes.entry` | set one pair on the point |
| `another entrance has this name, which a dungeon map's exits lead out by` | give each entrance of the land map its own name |
| `wotes.flag 0xNN is one of the game's own flags; an entrance sets one from 0x44 to 0xf6, or 0 for none` | use a flag from the free range, or clear the property |
| `open maps/dungeons/NAME.tmj: no such file or directory` | the entrance's `wotes.dungeon` names a map that is not in `dungeons/` beside the land map; start one with `new NAME --dungeon` |
| `N entrances with the game's 40; an exit reaches 128` | fewer entrances |
| `N levels with the game's 38; an entrance names its level in a byte, so 256 at most` | fewer caves |
| `N stairs records with the game's 13; a staircase reaches 128` | fewer staircases landing on different entry points |
| `the grid is W by H, N tiles; the game takes 1360 tiles and 64 a side` | a smaller map |
| `a dungeon map's tiles are 16 pixels; these are W by H` | start the map from `new NAME --dungeon`; one 16-pixel tile is one dungeon tile |
| `map property wotes.look: no look called "…": use one of …` | set `wotes.look` to one of the seven looks |
| `the map does not use the side tileset dungeon-sides.tsj` | add `rom/dungeons/dungeon-sides.tsj` to the map (Map > Add External Tileset) |
| `a dungeon map has the tile layers north, east, south, west` | name the four side layers so |
| `layer NAME is W by H, the map is W by H` | resize the layer to the map in Tiled |
| `layer NAME: tile X,Y holds no plain tile of the side tileset` | paint the cell from the side tileset, without a flip |
| `the map has no entry point, so no cave a party could be in` | place an `entry` point on `entries` |
| `the look LOOK has no art for side value N; it draws 1 to M` | use a side value the look draws |
| `tile X,Y holds a tile of the SIDE row; each layer takes its own row of the side tileset` | paint the `north` layer from the north row, and so on |
| `(x, y) is outside the W by H grid` | move the object onto the map |
| `no entry point reaches tile X,Y` | dig a way to the tile from an entry point, or move the object into a cave |
| `is KIND; this layer holds …` | put the object on its layer: entries, exits and stairs, events, items, monsters and actors |
| `the cave of entry point NAME has N records; a level places 127` | fewer objects in that cave |
| `the cave of entry point NAME has N entry points; 86 work` | fewer entry points in that cave |
| `the cave of entry point NAME has N texts; a record names one in a byte` | fewer different texts in that cave |
| `another entry point has this name` | rename one of the two entry points |
| `an entry point needs a name, which entrances and stairs lead to it by` | name the point |
| `an entry point needs wotes.facing, 0 north to 3 west` | set the facing |
| `a dungeon object is a point on its tile` | redraw the object as a point |
| `an exit names an entrance of the land map in wotes.to, or one of the game's by number in wotes.entrance` | set one of the two on the exit |
| `a staircase names an entry point in wotes.to, or one of the game's stairs records by number in wotes.value` | set one of the two on the staircase |
| `wotes.action must be 0 (a trap), 1 (a reward or a message), 2 (a quest picture) or 3 (an item gate)` | set the event's action |
| `LAND.tmj has no entrance called NAME for this exit to lead out by` | set the exit's `wotes.to` to an entrance of the land map, or `wotes.entrance` to one of the game's |
| `MAP.tmj has no entry point called NAME for this staircase to land on` | set `wotes.to` to an entry point of that map, with `wotes.dungeon` when it is another map |
| `line N of the text has M characters, the box shows 26` | shorten the line |
| `the text has N lines, a dungeon text may have 9` | cut the text |
| `line N of the text has 'c', which the dungeon font cannot draw` | use the space, punctuation, digits and A to Z |
| `wotes.monster must be the battle type's number, as the extracted levels carry it` / `wotes.item must be the item's number` / `wotes.monster is missing: the battle type's number` / `wotes.item is missing: the item's number` | write the number; the extracted levels under `maps/rom/dungeons/` show each monster's and item's beside its name |
| `wotes.count: place one object a monster, as the extracted levels do` | one point a monster, stacked on the tile |
| `battle type 0xNN is no dungeon monster; they run 0x13 to 0x61` | pick a monster a dungeon record can hold |
| `wotes.amount is missing, and the game's levels hold no monster of this type to take it from` | set `wotes.amount` and `wotes.uses` on the monster |
| `wotes.gate 0xNN is one of the game's own flags; new levels use 0x44 to 0xf6` (also `wotes.flag`) | use a flag from the free range, 0 for none |
| `pixel (X, Y) has colour #RRGGBB, not one of palette line 2's` | paint the area map with the sixteen colours of `maps/rom/area/valley.png`, or render it again |
| `pixel (X, Y) has alpha N; use 0 for transparent or 255 for opaque` | no half-transparent pixels in the area map |
| `is W by H pixels; the area map is 208 by 208` | the PNG `wotes.area` names must be 208 pixels square |
| `N distinct tiles; the picture has room for 582` | simplify the picture: fewer different 8-pixel tiles, such as flat colour where it shows foliage |
| `wotes.area_flag 0xNN names the flag of an area map the map does not have: set wotes.area` | name the picture, or drop the property |
| `wotes.area_flag 0xNN is not a flag: VIEW MAP may wait for one from 0x1 to 0xfe; leave it out for a map there from the start` | a flag is `0x1` to `0xFE`; without the property the map is there from the start |
| `wotes.area_width N is outside 1 to 65535` | the coverage is in land pixels; 6144 is the whole grid |
| `map format version X is not supported` / `tileset format version X is not supported` | save with Tiled 1.10 to 1.12 |
| `not a Tiled map (type "…")` / `not a Tiled tileset (type "…")` | the file is not what the reference expects: a tileset where a map is named, or the reverse |
| `layer "NAME" has no tile data` / `layer "NAME": N cells for a W by H layer` | save the map again in Tiled; a tile layer holds one cell per tile |
