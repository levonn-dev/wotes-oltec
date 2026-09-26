| Each .hook_XXXXXX section is placed at ROM offset XXXXXX by the HOOKS list in
| CMakeLists.txt; its .orig_ twin declares the clean-ROM bytes mdrom checks before writing.

| Startup hook at 0x300: the stack pointer is not valid yet, so jmp rather than jsr.
| poc_entry runs the lea the game executes at 0x306, calls C, then resumes at 0x306.
| No registers are saved: the code from 0x306 on reinitialises every register.
    .section .hook_000300,"ax"
    jmp     poc_entry
    .section .orig_000300,""
    .byte   0x4e,0x71,0x4e,0x71,0x4e,0x71

    .text
    .globl  poc_entry
poc_entry:
    lea     0xfffffff6,%sp
    jsr     poc_init
poc_ret:
    jmp     0x306

| Class list on the character-creation screen: FIGHTER becomes WARRIOR in both
| copies, the plain list at 0x22DA and the menu list with control codes at 0x230D.
    .section .hook_0022E1,"a"
    .ascii  "WARRIOR"
    .section .orig_0022E1,""
    .ascii  "FIGHTER"

    .section .hook_002316,"a"
    .ascii  "WARRIOR"
    .section .orig_002316,""
    .ascii  "FIGHTER"

| Region tables. game_load_region reads six four-entry columns at 0x8D2-0x910 with
| (d8,PC,Xn) addressing, which cannot reach the expansion, so the whole run is redone
| here against the five-entry columns in src/oltec.s and control resumes at the
| monsters call. Touches d0, d1, a0 and a1, all of which that run overwrites.
| d1 arrives zeroed: the clean ROM's moveq #0,D1 at 0x8D0 runs right before the hook's jsr,
| so move.b ram_region,%d1 fills only the low byte and the word index stays clean.
    .section .hook_0008D2,"ax"
    jsr     region_tables_hook
    .section .orig_0008D2,""
    .byte   0x12,0x38,0x8f,0x3d,0xe5,0x41

    .text
region_tables_hook:
    move.b  ram_region,%d1
    asl.w   #2,%d1
    lea     region_map,%a1
    movea.l (0,%a1,%d1.w),%a1
    move.l  %a1,0x18(%a0)                | front plane cells into ram_map_layer_front
    lea     ram_map_layer_ground,%a0
    move.l  %d2,%d0
    jsr     game_load_region_place_party
    jsr     game_load_region_party_pos
    jsr     game_load_region_after_place
    lea     region_terrain,%a1
    movea.l (0,%a1,%d1.w),%a1
    move.l  %a1,0x18(%a0)                | ground plane cells into ram_map_layer_ground
    lea     region_blocks,%a1
    move.l  (0,%a1,%d1.w),ram_map_blocks
    lea     region_tiles,%a1
    move.l  (0,%a1,%d1.w),ram_map_metatiles
    lsr.w   #1,%d1
    lea     region_area,%a1
    move.w  (0,%a1,%d1.w),ram_region_area
    lea     region_monsters,%a1
    move.w  (0,%a1,%d1.w),%d0
    addq.l  #4,%sp
    jmp     game_load_region_continue

| Tile set. game_load_region_tiles is a four-way switch that sends every region above
| the jungle to the swamp arm; region 4 does the jungle arm's two steps with its own
| area id: graphics set 10, then 4 into ram_region_area. Touches d0.
    .section .hook_00BB90,"ax"
    jsr     region_tiles_hook
    .section .orig_00BB90,""
    .byte   0x10,0x38,0x8f,0x3d,0x66,0x22

    .text
region_tiles_hook:
    move.b  ram_region,%d0
    cmpi.b  #4,%d0
    bne.s   1f
    addq.l  #4,%sp
    moveq   #10,%d0
    jsr     game_load_graphics_set
    move.w  #4,ram_region_area
    rts                                  | out of game_load_region_tiles, as the jungle arm does
1:  tst.b   %d0
    beq.s   2f                           | the city: back to the instruction after the hook
    addq.l  #4,%sp
    jmp     game_load_region_tiles_valley
2:  rts

| World map. game_world_map names the swamp and the jungle and draws every other region
| from four-entry tables; region 4 takes the refusal instead. rts keeps the condition
| codes for the bne.b that follows the hook. Touches nothing.
    .section .hook_0097C6,"ax"
    jsr     region_world_map_hook
    .section .orig_0097C6,""
    .byte   0x0c,0x38,0x00,0x03,0x8f,0x3d

    .text
region_world_map_hook:
    cmpi.b  #4,ram_region
    bne.s   1f
    addq.l  #4,%sp
    jmp     game_world_map_refuse
1:  cmpi.b  #3,ram_region
    rts

| Save-slot names. The slot list indexes rom_region_names with the region times four;
| the five-entry copy takes its place. Touches a0 and d0, as the replaced pair did.
    .section .hook_009D6E,"ax"
    jsr     region_name_hook
    .section .orig_009D6E,""
    .byte   0x20,0x7b,0x00,0x2a,0x20,0x03

    .text
region_name_hook:
    lea     region_names,%a0
    movea.l (0,%a0,%d0.w),%a0
    move.l  %d3,%d0
    rts

| Terrain id. game_terrain_id turns the area id into the terrain id the three music pickers
| and the encounter roll index their tables with; area id 4 is terrain 0x0D, a slot no tile
| classification reaches. Touches d0, which every arm of the classifier overwrites.
    .section .hook_012F38,"ax"
    jsr     terrain_id_hook
    .section .orig_012F38,""
    .byte   0x0c,0x40,0x00,0x02,0x66,0x04

    .text
terrain_id_hook:
    cmpi.w  #4,%d0
    bne.s   1f
    moveq   #0x0d,%d0
    addq.l  #4,%sp
    rts                                  | out of game_terrain_id with the answer
1:  cmpi.w  #2,%d0
    beq.s   2f
    addq.l  #4,%sp
    jmp     game_terrain_id_other
2:  rts                                  | the jungle: back to the moveq #0xb after the hook

| Encounter groups. The roll indexes a thirteen-pointer table with the terrain id, and the slot
| terrain 0x0D would read is code, so its list comes from here. Touches d0 and a0, which the
| replaced pair overwrites; the cmpa.l after the hook sets its own condition codes.
    .section .hook_012D36,"ax"
    jsr     encounter_group_hook
    .section .orig_012D36,""
    .byte   0xe5,0x48,0x20,0x7b,0x00,0x62

    .text
encounter_group_hook:
    lsl.w   #2,%d0
    cmpi.w  #0x0d,%d4
    bne.s   1f
    lea     oltec_encounters,%a0
    rts
1:  lea     rom_terrain_encounter_groups,%a0
    movea.l (0,%a0,%d0.w),%a0
    rts

| Terrain 0x0D's slots in the three track tables and the encounter chance table. These four
| slots are zero in the clean ROM because no terrain reached them.
    .section .hook_004F8D,"a"
    .byte   0x11                        | OLTEC THEME while wandering
    .section .orig_004F8D,""
    .byte   0

    .section .hook_00509B,"a"
    .byte   0x01                        | AZCAN COMBAT in a fight
    .section .orig_00509B,""
    .byte   0

    .section .hook_0050DD,"a"
    .byte   0x02                        | AZCAN WIN after one
    .section .orig_0050DD,""
    .byte   0

    .section .hook_012CC5,"a"
    .byte   0x0a                        | the encounter chance terrains 0 to 11 have
    .section .orig_012CC5,""
    .byte   0

| Entrance 14 of rom_region_exits and rom_region_exit_points, filler in the clean ROM, is
| the two-way door between the Tradeway bottom and region 4: leaving map 0x0C lands here.
    .section .hook_010424,"a"
    .byte   4
    .section .orig_010424,""
    .byte   0x3f

    .section .hook_01055C,"a"
    .short  0x0c70, 0x0588              | x, y in map pixels
    .byte   0, 0x0c, 0, 1               | any game level, map 0x0C, no flag set, entry point 1
    .section .orig_01055C,""
    .byte   0x00,0x00,0x00,0x00,0x3f,0x3f,0x3f,0x3f

| Map 0x0C's object list: the merchant event and the medallion give way to a kind-6
| transition on tile 8,0 with exit code -15, entrance 14; the records behind them move up.
    .section .hook_0D70A8,"a"
    .byte   0x06, 0xf1, 8, 0, 0, 0, 0, 0, 0, 0
    .incbin "clean.md", 0xd70bc, 340
    .byte   0                           | the ten clean bytes behind the terminator are dead
    .section .orig_0D70A8,""
    .incbin "clean.md", 0xd70a8, 351

    .section .note.GNU-stack,"",%progbits
