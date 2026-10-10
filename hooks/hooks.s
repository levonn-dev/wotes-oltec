| Each .hook_XXXXXX section is placed at ROM offset XXXXXX by the HOOKS list in
| CMakeLists.txt; its .orig_ twin declares the bytes mdrom expects in the base ROM before writing.

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
    lea     region_animated_tiles,%a1
    move.w  (0,%a1,%d1.w),%d0
    addq.l  #4,%sp
    jmp     game_load_region_continue

| Tile set. game_load_region_tiles is a four-way switch that sends every region above
| the jungle to the swamp arm; region 4 loads the land's own chunk list at VRAM 0, then the
| villagers' tiles over graphics set 0 as the city loads its figures, then its area id. The
| arm runs on every region load and every return from a camp or shop screen. Touches d0, d1,
| a0 and a1, which every arm of the switch overwrites.
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
    lea     oltec_tiles,%a0
    suba.l  %a1,%a1                      | VRAM 0, where every region's set loads
    jsr     game_load_tiles
    jsr     game_load_figure_tiles
    move.w  #4,ram_region_area
    rts                                  | out of game_load_region_tiles, as the jungle arm does
1:  tst.b   %d0
    beq.s   2f                           | the city: back to the instruction after the hook
    addq.l  #4,%sp
    jmp     game_load_region_tiles_valley
2:  rts

| Area map. game_world_map gates the jungle and the swamp on an event flag and draws every region
| from four-entry tables that end before region 4; the land's picture, its divisors and its flag come
| from the map as oltec_area_*. The gate: a flag byte of 255 refuses as the game did, 0 draws, any other
| value is an event flag to test. rts keeps the condition codes for the bne.b after the hook. Touches d0.
    .section .hook_0097C6,"ax"
    jsr     region_world_map_hook
    .section .orig_0097C6,""
    .byte   0x0c,0x38,0x00,0x03,0x8f,0x3d

    .text
region_world_map_hook:
    cmpi.b  #4,ram_region
    bne.s   3f
    addq.l  #4,%sp
    moveq   #0,%d0
    move.b  oltec_area_flag,%d0
    cmpi.b  #0xff,%d0
    beq.s   1f
    tst.b   %d0
    beq.s   2f
    jsr     game_test_event_flag
    beq.s   1f
2:  jmp     game_world_map_draw
1:  jmp     game_world_map_refuse
3:  cmpi.b  #3,ram_region
    rts

| The drawing routine's scale branch, eight bytes: cmpi.b #2,d2; blt.b; bgt.b. Region 4 divides the
| party's position by the map's divisors, loads the land's chunk list and plane where the game loads
| the others, fades in and joins the wait loop with the marker's position in d3 and d4; the other
| regions get the test and its branches back. Touches d0, a0 and a1, as the game's own path does.
    .section .hook_009862,"ax"
    jsr     area_map_hook
    nop
    .section .orig_009862,""
    .byte   0x0c,0x02,0x00,0x02,0x6d,0x16,0x6e,0x0a

    .text
area_map_hook:
    cmpi.b  #4,%d2
    beq.s   1f
    cmpi.b  #2,%d2
    blt.s   4f
    bgt.s   5f
    rts                                  | the jungle's divisors follow the hook
4:  addq.l  #4,%sp
    jmp     game_world_map_scale_valley
5:  addq.l  #4,%sp
    jmp     game_world_map_scale_swamp
1:  addq.l  #4,%sp
    divu.w  oltec_area_scale_x,%d3
    divu.w  oltec_area_scale_y,%d4
    addq.w  #8,%d3
    addq.w  #8,%d4
    lea     oltec_area_tiles,%a0
    movea.l #0x4a20,%a1
    jsr     game_load_tiles
    move.w  #0x1000,%d0
    lea     oltec_area_plane,%a0
    movea.l #0xe000,%a1
    jsr     game_load_plane
    lea     rom_main_palette,%a0
    jsr     game_fade_in_jump            | through the jump every fade-in takes, as the game draws its pictures
    jmp     game_world_map_wait

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
| classification reaches, or terrain 6, the lava pit's, while ram_on_lava is set: the Tradeway's
| door sets it (below) so the fights there sound as they do from the valley, and outdoors the land
| has no lava to set it. Touches d0, which every arm of the classifier overwrites.
    .section .hook_012F38,"ax"
    jsr     terrain_id_hook
    .section .orig_012F38,""
    .byte   0x0c,0x40,0x00,0x02,0x66,0x04

    .text
terrain_id_hook:
    cmpi.w  #4,%d0
    bne.s   1f
    moveq   #0x0d,%d0
    tst.b   ram_on_lava
    beq.s   3f
    moveq   #6,%d0
3:  addq.l  #4,%sp
    rts                                  | out of game_terrain_id with the answer
1:  cmpi.w  #2,%d0
    beq.s   2f
    addq.l  #4,%sp
    jmp     game_terrain_id_other
2:  rts                                  | the jungle: back to the moveq #0xb after the hook

| Dungeon music. game_enter_dungeon starts the wandering track of the terrain the party stands
| on before the dungeon runs, and nothing underground picks another, so a dungeon sounds like its
| door: the Tradeway's valley door stands on lava, whose flag makes game_music_wander play the
| lava pit's track. From the land's door, map 0x0C, the flag is set here instead, and the track
| byte cleared so the picker starts the lava pit's track at once over the land's, as the camp's
| MUSIC toggle restarts a track; the first outdoor frame back on the land clears the flag again.
| The land's own caves keep the land's track. Eight bytes: the timer clear and the picker's call.
| Touches nothing the picker does not; a2 is the entrance record's second long.
    .section .hook_01044A,"ax"
    jsr     dungeon_music_hook
    nop
    .section .orig_01044A,""
    .byte   0x51,0xf8,0xa9,0x0f,0x4e,0xb8,0x4e,0xdc

    .text
dungeon_music_hook:
    sf      ram_music_timer
    cmpi.b  #4,ram_region
    bne.s   1f
    cmpi.b  #0x0c,1(%a2)                 | the record's map byte
    bne.s   1f
    st      ram_on_lava
    st      ram_music_track
1:  jmp     game_music_wander

| Encounter groups. The roll indexes a thirteen-pointer table with the terrain id, and the slot
| terrain 0x0D would read is code, so its list is the one of the spawn area the party stands in;
| the roll has left by then where no area with a list lies. Touches d0 and a0, which the replaced
| pair overwrites; the cmpa.l after the hook sets its own condition codes.
    .section .hook_012D36,"ax"
    jsr     encounter_group_hook
    .section .orig_012D36,""
    .byte   0xe5,0x48,0x20,0x7b,0x00,0x62

    .text
encounter_group_hook:
    lsl.w   #2,%d0
    cmpi.w  #0x0d,%d4
    bne.s   1f
    movea.l spawn_area,%a0
    movea.l 4(%a0),%a0
    rts
1:  lea     rom_terrain_encounter_groups,%a0
    movea.l (0,%a0,%d0.w),%a0
    rts

| Spawn areas. The land's map says where random encounters happen: oltec_spawn_grid gives every
| metatile cell of a rectangle an area number, 0 for none, and oltec_spawns holds a record an area:
| the frames between two rolls (0 for the game's own), a word set when a group is tried on every
| side of the party, and the area's list of eight entries or 0 when nothing spawns there. The area is
| looked up where the roll reloads its countdown, and the three hooks after it read spawn_area.

| The countdown's reload, a routine of its own that the roll and two other places call: the game's
| pause for the level, then in region 4 the pause of the area under the party when it has one.
| The routine's PC-relative table read cannot move, so the whole of it is redone here. Leaves d0
| as the routine does and touches nothing else.
    .section .hook_012CC8,"ax"
    jmp     encounter_pause_hook
    .section .orig_012CC8,""
    .byte   0x70,0x00,0x10,0x38,0x8f,0x3e

    .text
encounter_pause_hook:
    moveq   #0,%d0
    move.b  ram_game_level,%d0
    add.w   %d0,%d0
    movem.l %d0-%d1/%a0,-(%sp)
    lea     rom_encounter_pauses,%a0
    move.w  (0,%a0,%d0.w),ram_encounter_countdown
    clr.l   spawn_area
    cmpi.b  #4,ram_region
    bne.s   1f
    bsr.s   spawn_lookup
    move.l  %a0,spawn_area
    beq.s   1f
    tst.w   (%a0)
    beq.s   1f
    move.w  (%a0),ram_encounter_countdown
1:  movem.l (%sp)+,%d0-%d1/%a0
    rts

| Answers in a0 the record of the spawn area under the party, or 0. The cell is the one of the tile
| the party stands on, 8 pixels in from its position. Touches d0 and d1.
spawn_lookup:
    lea     oltec_spawn_grid,%a0
    moveq   #0,%d0
    move.w  ram_party_x,%d0
    addq.w  #8,%d0
    divu.w  #24,%d0
    sub.w   (%a0),%d0                    | from the grid's west edge
    bcs.s   1f
    cmp.w   4(%a0),%d0
    bcc.s   1f
    moveq   #0,%d1
    move.w  ram_party_y,%d1
    addq.w  #8,%d1
    divu.w  #24,%d1
    sub.w   2(%a0),%d1                   | from its north edge
    bcs.s   1f
    cmp.w   6(%a0),%d1
    bcc.s   1f
    mulu.w  4(%a0),%d1
    andi.l  #0xffff,%d0
    add.l   %d0,%d1
    moveq   #0,%d0
    move.b  8(%a0,%d1.l),%d0
    beq.s   1f
    subq.w  #1,%d0
    lsl.w   #3,%d0                       | eight bytes a record
    lea     oltec_spawns,%a0
    adda.w  %d0,%a0
    rts
1:  suba.l  %a0,%a0
    rts

| The roll's city test, which leaves without an encounter: region 4 leaves the same way where the
| party stands in no area or in one without a list. rts keeps the condition codes for the beq.w
| after the hook. Touches a0, which the roll loads before it reads it.
    .section .hook_012D00,"ax"
    jsr     encounter_area_hook
    .section .orig_012D00,""
    .byte   0x0c,0x78,0x00,0x00,0x8d,0x3e

    .text
encounter_area_hook:
    cmpi.b  #4,ram_region
    bne.s   1f
    tst.l   spawn_area
    beq.s   2f
    movea.l spawn_area,%a0
    tst.l   4(%a0)
2:  rts
1:  cmpi.w  #0,ram_region_area
    rts

| Placement. A group that finds no way to the party from 160 pixels ahead of it is given up,
| unless the roll is forced: then the game tries the party's other seven sides. An area that asks
| for every side takes that retry too. The forced roll goes back to its own test after the hook.
| Touches a0, which the placement loads before it reads it.
    .section .hook_013132,"ax"
    jsr     encounter_place_hook
    .section .orig_013132,""
    .byte   0x4a,0x38,0x8f,0x41,0x67,0x1a

    .text
encounter_place_hook:
    tst.b   ram_encounter_forced
    bne.s   3f
    addq.l  #4,%sp
    cmpi.b  #4,ram_region
    bne.s   2f
    tst.l   spawn_area
    beq.s   2f
    movea.l spawn_area,%a0
    tst.w   2(%a0)
    beq.s   2f
    jmp     game_encounter_place_next
2:  jmp     game_encounter_place_fail
3:  rts

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

| The entrance tables. rom_region_exits and rom_region_exit_points hold forty records; the land's
| map emits copies with its own entrances written in and added, the Tradeway's two-way door at
| record 14 among them, and game_enter_dungeon reads the copies. On the way in its walk takes
| their addresses in a2 and a1; eight bytes because six would split the second lea.
    .section .hook_0103F0,"ax"
    jsr     entrance_tables_hook
    nop
    .section .orig_0103F0,""
    .byte   0x45,0xfa,0x00,0xfa,0x43,0xfa,0x00,0x20

    .text
entrance_tables_hook:
    lea     oltec_entrance_points,%a2
    lea     oltec_entrance_regions,%a1
    rts

| On the way out d0.w is the entrance: its region byte goes to ram_region and the first long of
| its record, the position, comes back in d0. The twelve bytes are the two table reads and the
| shift between them.
    .section .hook_010482,"ax"
    jsr     entrance_leave_hook
    nop
    nop
    nop
    .section .orig_010482,""
    .byte   0x11,0xfb,0x00,0x92,0x8f,0x3d,0xe7,0x40,0x20,0x3b,0x00,0x60

    .text
entrance_leave_hook:
    move.l  %a0,-(%sp)
    lea     oltec_entrance_regions,%a0
    move.b  (%a0,%d0.w),ram_region
    asl.w   #3,%d0
    lea     oltec_entrance_points,%a0
    move.l  (%a0,%d0.w),%d0
    movea.l (%sp)+,%a0
    rts

| The stairs table. A staircase's value picks a record of rom_dungeon_stairs, thirteen in the
| game; the land's map emits a copy with its own after them. d0.w is the record's offset; the
| game goes on with a0 on the record's third byte and the destination map in d0.w.
    .section .hook_010CEE,"ax"
    jsr     stairs_hook
    .section .orig_010CEE,""
    .byte   0x41,0xfb,0x00,0x1a,0x30,0x18

    .text
stairs_hook:
    lea     oltec_stairs,%a0
    adda.w  %d0,%a0
    move.w  (%a0)+,%d0
    rts

| The level tables. game_change_map holds rom_map_table's address and the object loader
| rom_map_directory's; the land's map emits copies with a level for each of its caves after the
| game's 38. A text table's entries stay counted from rom_map_directory, which 0x21814 keeps.
    .section .hook_0113AC,"a"
    .long   oltec_levels
    .section .orig_0113AC,""
    .long   0x0001145c

    .section .hook_010EEE,"a"
    .long   oltec_level_directory
    .section .orig_010EEE,""
    .long   0x000d5e36

| Map 0x0C's object list: the merchant event and the medallion give way to a kind-6
| transition on tile 8,0 with exit code -15, entrance 14; the records behind them move up.
    .section .hook_0D70A8,"a"
    .byte   0x06, 0xf1, 8, 0, 0, 0, 0, 0, 0, 0
    .incbin "base.md", 0xd70bc, 340
    .byte   0                           | the ten clean bytes behind the terminator are dead
    .section .orig_0D70A8,""
    .incbin "base.md", 0xd70a8, 351

| Special places. game_special_places walks rom_special_places, whose castle records would fire
| on the land's coordinates; region 4 walks oltec_triggers instead, with the same rectangle,
| level and latch rules, and its four kinds: 1 opens shop arg through game_shop, 2 shows text
| arg and restores the party as the Waters of Life do, 3 shows text arg and waits for a button
| as the speeches do, 4 fires the event block arg once: its text with its quest picture played
| over the open box as the swamp's curse place has it, raising the game level as
| game_quest_cutscene would, and played again until a button, which takes the text's page with
| it;
| then its item through game_give_item (nobody can take it: the dungeon's refusal sound, and
| the event waits for the next entry, so a quest item is never lost), then its flag; once the
| flag is set the block's after-text speaks instead. The picture's tiles land on the first two
| figure blocks, which are queued again behind it. A record is y0 y1 x0 x1 words, a
| level byte, a kind byte and a long argument; a 0xFFFF word ends the table. An event block
| is the text's address, the after-text's or 0, then the picture, the item and the flag as
| bytes. The hook covers eight bytes because six would split the move.w at 0xD310; the other
| regions redo the two replaced instructions and resume at 0xD314. Touches d0, d1, a0 and a1,
| as the routine's own actions do; a5 is the VDP control port.
    .section .hook_00D30C,"ax"
    jsr     special_places_hook
    nop
    .section .orig_00D30C,""
    .byte   0x48,0xe7,0x3e,0x00,0x30,0x38,0x91,0x4c

    .text
special_places_hook:
    addq.l  #4,%sp                       | the hook stands at the routine's entry: its return is not needed
    movem.l %d2-%d6,-(%sp)               | the registers the routine itself saves: the other regions resume inside it
    move.w  ram_party_x,%d0
    cmpi.b  #4,ram_region
    beq.s   1f
    jmp     game_special_places_continue
1:  move.l  %a2,-(%sp)
    move.w  ram_party_y,%d1
    lea     oltec_triggers,%a0
2:  cmpi.w  #-1,(%a0)
    beq     7f                           | no rectangle holds the party: the latch opens again
    movem.w (%a0)+,%d2-%d5               | y0, y1, x0, x1
    cmp.w   %d1,%d2
    bgt.s   3f
    cmp.w   %d1,%d3
    blt.s   3f
    cmp.w   %d0,%d4
    bgt.s   3f
    cmp.w   %d0,%d5
    blt.s   3f
    tst.b   ram_place_latch
    bne     8f                           | still inside a rectangle from an earlier step
    st      ram_place_latch
    move.b  (%a0)+,%d6                   | the highest game level the trigger fires at
    cmp.b   ram_game_level,%d6
    blt     8f
    move.b  (%a0)+,%d6
    subq.b  #1,%d6
    beq.s   4f
    subq.b  #1,%d6
    beq.s   5f
    subq.b  #1,%d6
    bne.s   6f
    movea.l (%a0),%a0                    | a speech: its text
    bsr     land_speech
    bra     8f
3:  addq.l  #6,%a0                       | past the level, the kind and the argument
    bra     2b
4:  move.l  (%a0),%d0                    | a shop: its index, as the action table passes it
    jsr     game_shop
    bra     8f
5:  movea.l (%a0),%a0                    | a restore: the Waters of Life sequence with the land's text
    jsr     game_message_box
    moveq   #0x40,%d0
    jsr     game_play_sound
    moveq   #0,%d0
    jsr     game_party_restore
    jsr     game_wait_button
    move.w  #0x9200,(%a5)
    bra     8f
6:  movea.l (%a0),%a2                    | an event: its block
    moveq   #0,%d0
    move.b  10(%a2),%d0
    jsr     game_test_event_flag
    beq.s   9f
    move.l  4(%a2),%d0                   | done already: its after-text speaks, if it has one
    beq     8f
    movea.l %d0,%a0
    bsr     land_speech
    bra     8f
9:  movea.l (%a2),%a0
    jsr     game_message_box             | the text, with the picture over it as the game's scenes have it
    moveq   #0,%d0
    move.b  8(%a2),%d0                   | the quest picture
    beq     10f
    subq.w  #1,%d0
    lea     rom_quest_levels,%a0
    move.b  (%a0,%d0.w),%d1
    cmp.b   ram_game_level,%d1
    ble.s   11f
    move.b  %d1,ram_game_level
11: cmpi.b  #4,8(%a2)                    | the merchant's picture opens the castle gate, as game_quest_cutscene has it
    bne.s   13f
    sf      ram_castle_gate_shut
13: lsl.w   #2,%d0
    lea     rom_quest_pictures,%a0
    movea.l (%a0,%d0.w),%a0
    lea     rom_quest_picture_sprites,%a1
    movea.l (%a1,%d0.w),%a1
    move.l  #0x00680000,%d0
    move.l  #0x6000,%d1
    move.l  %a1,picture_script           | for picture_loop_hook, which plays it again until a button
    jsr     game_draw_picture
    lea     rom_city_figure_tiles,%a0    | the picture's tiles ran over the first two figure blocks: load them again
    moveq   #2,%d0
    move.l  #0x75E001E0,%d1
    jsr     game_vdp_queue
    lea     rom_city_figure_tiles_2,%a0
    moveq   #2,%d0
    move.l  #0x79A00240,%d1
    jsr     game_vdp_queue
    bsr     land_speech_next             | that button turns the text's page, or closes it with the picture
    bra.s   14f
10: bsr     land_speech_wait             | a button, the text's other pages, the box hidden
14: moveq   #0,%d0
    move.b  9(%a2),%d0                   | the item
    beq.s   12f
    jsr     game_give_item
    tst.w   %d0
    bpl.s   12f
    moveq   #0x36,%d0                    | every pack is full: the dungeon's refusal sound, and the event waits
    jsr     game_play_sound
    bra     8f
12: moveq   #0,%d0
    move.b  10(%a2),%d0                  | the flag: the event is done
    jsr     game_set_event_flag
    bra.s   8f
7:  sf      ram_place_latch
8:  movea.l (%sp)+,%a2
    movem.l (%sp)+,%d2-%d6
    rts

| A speech: the text at a0, paged on a button until none is left, then the window plane hidden;
| land_speech_wait does the same for a box already up, land_speech_next for one whose button
| has already been pressed.
land_speech:
    jsr     game_message_box
land_speech_wait:
    st      ram_scene
1:  jsr     game_outdoor_frame
    tst.w   ram_party_alive
    beq.s   2f
    jsr     game_read_pad
    andi.w  #0xf0,%d0
    beq.s   1b
land_speech_next:
    st      ram_scene
    suba.l  %a0,%a0
    jsr     game_message_box             | the next page, until none is left
    bne.s   1b
2:  move.w  #0x9200,(%a5)
    clr.b   ram_scene
    rts

| Shops at game level 4. The game reaches level 4 only for the walk home, and game_shop was
| written for levels 0 to 3: its greeting table has four texts a shop, so level 4 reads the next
| shop's first, and its price adjuster moves every price by a quarter of itself per level, so at
| level 4 buying costs double and selling pays the price's remainder by four, 0 to 3 gold, which
| the shop refuses as nothing. In the land the shops stay open after the merchant's quest raises
| the level, so both read the level as 3 at most. Eight bytes each: the register clear, the
| level read and the instruction after it, redone here. Each touches only the registers those
| instructions set.
    .section .hook_00C504,"ax"
    jsr     shop_greeting_hook
    nop
    .section .orig_00C504,""
    .byte   0x74,0x00,0x14,0x38,0x8f,0x3e,0x66,0x30

    .text
shop_greeting_hook:
    moveq   #0,%d2
    move.b  ram_game_level,%d2
    cmpi.b  #3,%d2
    ble.s   1f
    moveq   #3,%d2
1:  tst.b   %d2
    beq.s   2f
    addq.l  #4,%sp
    jmp     game_shop_greeting           | past the caravan's level-0 test, as the replaced bne.b went
2:  rts

    .section .hook_00CEC2,"ax"
    jsr     shop_price_hook
    nop
    .section .orig_00CEC2,""
    .byte   0x72,0x00,0x12,0x38,0x8f,0x3e,0x70,0x1f

    .text
shop_price_hook:
    moveq   #0,%d1
    move.b  ram_game_level,%d1
    cmpi.b  #3,%d1
    ble.s   1f
    moveq   #3,%d1
1:  moveq   #0x1f,%d0                    | the 32 prices the adjuster's loop counts down
    rts

| Pictures. game_draw_picture's script loop ends each frame, after its vblank wait, on a test of
| the next script byte: another step, or the tail that returns (the lizard-man picture alone is
| held still there until a button). In region 4 the picture plays over an event's text, so it
| plays its script again from picture_script whenever it ends, and a button ends it at once
| through the drawer's own tail; every other region runs the three replaced instructions. The
| hook covers eight bytes, the test and its two branches, and jumps back itself. Touches d0,
| which the loop sets before it reads it.
    .section .hook_018452,"ax"
    jsr     picture_loop_hook
    nop
    .section .orig_018452,""
    .byte   0x4a,0x13,0x6c,0x92,0x60,0x00,0x00,0x58

    .text
picture_loop_hook:
    addq.l  #4,%sp
    cmpi.b  #4,ram_region
    bne.s   1f
    jsr     game_read_pad                | a button ends the picture at once
    andi.w  #0xf0,%d0
    bne.s   2f
    tst.b   (%a3)
    bge.s   3f
    movea.l picture_script,%a3           | the script's end: its first step is read once this one's frames are spent
3:  jmp     game_draw_picture_loop
1:  tst.b   (%a3)                        | the game's own test: the next step, or the tail
    bge.s   3b
2:  jmp     game_draw_picture_tail

| Figures. game_city_npcs draws rom_city_npcs in the city only; region 4 enters its record loop
| with oltec_figures instead, past the townsfolk and the level test, and every other region
| leaves as before. The replaced bne.b is PC-relative, so the hook branches itself: the city
| resumes at the level test. Touches d0: the loop overwrites it in region 4, and elsewhere it
| leaves holding the region byte, which game_draw_sprites ignores and the party's sprite routine
| it jumps to at 0x65AA writes before it reads.
    .section .hook_00E282,"ax"
    jsr     figures_hook
    .section .orig_00E282,""
    .byte   0x4a,0x38,0x8f,0x3d,0x66,0x72

    .text
figures_hook:
    addq.l  #4,%sp
    move.b  ram_region,%d0
    beq.s   1f
    cmpi.b  #4,%d0
    bne.s   2f
    movem.l %d2-%d5/%a2,-(%sp)
    move.l  ram_map_layer_front,%d3      | the camera: the view box the loop tests records against
    subi.l  #0x180018,%d3
    move.l  %d3,%d5
    addi.l  #0x1000100,%d5
    move.l  %d3,%d2
    swap    %d2
    move.l  %d5,%d4
    swap    %d4
    lea     oltec_figures,%a2
    jmp     game_city_npcs_loop
1:  jmp     game_city_npcs_level
2:  rts                                  | out of game_city_npcs, as the other regions leave it

| Fade-in. Every outdoor, camp and shop screen fades ram_palette in to rom_main_palette, so
| colours written earlier are lost. In region 4 the outdoor fades are given a copy of the ROM
| palette with the villagers' colours over line 3, colours 2 to 15, instead; colour 1 of that
| line is the panel's background and stays the ROM's. The camp and shop screens draw their stone
| with line 3 colours 2 and 3 and keep the ROM's palette, told apart by ram_figures_on: every
| path that sets it fades to the outdoors next (0x48C to 0x5FE, 0x8330 to 0x836C, 0x104AC to
| 0x104DC, 0xDDE6 to 0xDE44, 0x17BC0 to 0x17BDA), and every camp and shop screen clears it at
| 0x8784 before its own fade (e.g. 0x886E, 0x8C36, 0x8EFE, 0xC3A4). The jmp the hook replaces is
| redone for every other target, region and screen. Touches a0, the target, which game_fade_in reads.
    .section .hook_00B3A4,"ax"
    jsr     fade_in_hook
    .section .orig_00B3A4,""
    .byte   0x4e,0xf9,0x00,0x00,0xb3,0xbe

    .text
fade_in_hook:
    addq.l  #4,%sp
    cmpi.b  #4,ram_region
    bne.s   1f
    tst.b   ram_figures_on
    beq.s   1f                           | a camp or shop screen
    cmpa.l  #rom_main_palette,%a0
    bne.s   1f
    movem.l %d0/%a1,-(%sp)
    lea     land_palette,%a1
    moveq   #31,%d0                      | 64 colours, two a long
2:  move.l  (%a0)+,(%a1)+
    dbf     %d0,2b
    lea     oltec_figure_colours+4,%a0   | colours 2 to 15
    lea     land_palette+100,%a1         | line 3, colour 2
    moveq   #6,%d0
3:  move.l  (%a0)+,(%a1)+
    dbf     %d0,3b
    movem.l (%sp)+,%d0/%a1
    lea     land_palette,%a0
1:  jmp     game_fade_in

    .section .bss
    .balign 2
land_palette:
    .space  128                          | the ROM palette with the villagers' line, the fade's target in the land
spawn_area:
    .space  4                            | the record of the spawn area the last countdown reload found the party in, or 0
picture_script:
    .space  4                            | the sprite script of the quest picture an event is showing, for its replay

    .section .note.GNU-stack,"",%progbits
