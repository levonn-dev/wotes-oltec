| Outdoor region 4: a copy of the jungle, entered from the bottom of the Oltec Tradeway.
| Every table is the clean ROM's four entries read straight out of the image, with the
| region's own entry appended; clean.md is the copy CMake keeps in the build directory.
    .section .rodata
    .balign 2

    .globl  region_map, region_terrain, region_blocks, region_tiles
    .globl  region_area, region_monsters, region_names
    .globl  oltec_front, oltec_ground, oltec_name, oltec_encounters

| rom_region_table, column by column: longs at 0x938, 0x948, 0x958, 0x968; words at 0x978, 0x980.
region_map:      .incbin "clean.md", 0x938, 16
                 .long   oltec_front
region_terrain:  .incbin "clean.md", 0x948, 16
                 .long   oltec_ground
region_blocks:   .incbin "clean.md", 0x958, 16
                 .long   0x091db8                | the jungle's blocks
region_tiles:    .incbin "clean.md", 0x968, 16
                 .long   0x08fad8                | the jungle's metatiles
region_area:     .incbin "clean.md", 0x978, 8
                 .short  4                       | area id 4: terrain 0x0D, with music and encounters of its own
region_monsters: .incbin "clean.md", 0x980, 8
                 .short  2                       | the jungle's monster list

| rom_region_names at 0x9D9A, then the fifth name in the same "- SWAMP" shape.
region_names:    .incbin "clean.md", 0x9d9a, 16
                 .long   oltec_name
oltec_name:      .asciz  "- OLTEC"
    .balign 2

| Random encounters of terrain 0x0D: monster id, fewest and most appearing as two nibbles, level 1
| like every list in the game, and affinity 0 so each kind fights alone.
oltec_encounters:
    .byte   0x76, 0x26, 1, 0                | TIGER
    .byte   0x75, 0x24, 1, 0                | GRIZZLY BEAR
    .byte   0x73, 0x24, 1, 0                | HILL GIANT
    .byte   0x7e, 0x13, 1, 0                | LION
    .byte   0x63, 0x48, 1, 0                | BOAR
    .byte   0x7d, 0x12, 1, 0                | RED DRAGON
    .byte   0x77, 0x24, 1, 0                | DIRE WOLF
    .byte   0x6b, 0x12, 1, 0                | PANTHER

| The jungle's two 32 by 32 cell arrays, 0x800 bytes each. Ground cell column 17, row 7 is
| the pyramid block 0x0016 instead of the path's 0x0021: the landmark beside the door's landing cell.
oltec_front:     .incbin "clean.md", 0x8ead8, 0x800
oltec_ground:    .incbin "clean.md", 0x8f2d8, 0x1e2
                 .short  0x0016
                 .incbin "clean.md", 0x8f2d8 + 0x1e4, 0x800 - 0x1e4

    .section .note.GNU-stack,"",%progbits
