| Outdoor region 4: the Oltec land, entered from the bottom of the Oltec Tradeway. Every table is the
| clean ROM's four entries read straight out of the image, with the region's own entry appended;
| clean.md is the copy CMake keeps in the build directory, and the land's own tables, its random
| encounters among them, come from maps/oltec.tmj, converted into gen/oltec-map.s at build time.
    .section .rodata
    .balign 2

    .globl  region_map, region_terrain, region_blocks, region_tiles
    .globl  region_area, region_animated_tiles, region_names
    .globl  oltec_name

| rom_region_table, column by column: longs at 0x938, 0x948, 0x958, 0x968; words at 0x978, 0x980.
region_map:      .incbin "clean.md", 0x938, 16
                 .long   oltec_front
region_terrain:  .incbin "clean.md", 0x948, 16
                 .long   oltec_ground
region_blocks:   .incbin "clean.md", 0x958, 16
                 .long   oltec_blocks
region_tiles:    .incbin "clean.md", 0x968, 16
                 .long   oltec_metatiles
region_area:     .incbin "clean.md", 0x978, 8
                 .short  4                       | area id 4: terrain 0x0D, with music and encounters of its own
region_animated_tiles: .incbin "clean.md", 0x980, 8
                 .short  3                       | the swamp's animated tiles: the land's tile set begins with the swamp's

| rom_region_names at 0x9D9A, then the fifth name in the same "- SWAMP" shape.
region_names:    .incbin "clean.md", 0x9d9a, 16
                 .long   oltec_name
oltec_name:      .asciz  "- OLTEC"
    .balign 2

    .section .note.GNU-stack,"",%progbits
