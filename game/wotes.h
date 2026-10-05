/* Prototypes and variables for the original game routines and RAM named in wotes.ld,
 * plus structs that overlay the game's data formats byte for byte (68000 is big-endian). */
#ifndef WOTES_H
#define WOTES_H

#include <stdint.h>

/* The 68000 aligns nothing to more than two bytes; pack to that so the overlays below keep
 * the game's layout under any compiler, including a host one. */
#pragma pack(push, 2)

void game_wait_vblank(void);
void game_debug_menu(void);
extern volatile uint8_t ram_frame_counter;
extern volatile uint16_t ram_cheat_menu;
extern volatile uint16_t ram_cheat_debug;
extern volatile uint8_t ram_game_level;
extern volatile uint8_t ram_show_coords;
extern volatile uint8_t ram_monsters_off;
extern volatile uint16_t ram_party_x;
extern volatile uint16_t ram_party_y;

extern volatile uint8_t ram_region;
extern volatile uint16_t ram_region_area;
void game_load_region(void);          /* the packed map x,y it moves the party to is passed in d0, so a C caller has to load d0 with inline asm */
void game_load_region_tiles(void);
void game_load_region_animations(void); /* the animated-tile list index is passed in d0, so a C caller has to load d0 with inline asm */

/* rom_region_table keeps one column per field rather than one record per region, so the whole
 * table is a single struct and every column is indexed by ram_region. */
struct region_table {
    uint32_t map[4];      /* the front plane's map words, VDP name table 0xC000, drawn in front of the party */
    uint32_t terrain[4];  /* the ground plane's map words, name table 0xE000, which the terrain and music lookup reads */
    uint32_t blocks[4];   /* 0x80-byte blocks of tile indices the map words select */
    uint32_t tiles[4];    /* the tile words a block entry resolves to */
    uint16_t area[4];     /* what game_load_region stores in ram_region_area */
    uint16_t animations[4]; /* which animated-tile list game_load_region_animations loads */
};
_Static_assert(sizeof(struct region_table) == 0x50, "region_table overlays the whole region table");
extern const struct region_table rom_region_table;

extern volatile uint8_t ram_stack_top; /* lowest byte of 0xFFFC00-0xFFFFFF a new game wrote; the stack starts at 0xFFFFF6 and grows down */

extern volatile uint16_t ram_map;
extern volatile uint16_t ram_map_entrance;
extern volatile uint16_t ram_map_pending;
extern volatile uint8_t ram_event_flags[0x20];
void game_change_map(void);        /* the map in the high word of d0 and the entry point in its low word, so a C caller has to load d0 with inline asm */
void game_run_dungeon(void);       /* the destination packed as in struct region_exit is passed in d0 and the exit code comes back in d0 */
void game_enter_dungeon(void);
int game_can_enter_tile(void);     /* the tile is passed in d0 and d1, the mover's kind in d2 and its object slot in d3 */
void game_set_event_flag(void);    /* the bit number is passed in d0 */
void game_test_event_flag(void);   /* the bit number is passed in d0 and the answer comes back in the Z flag, not in d0 */
int game_party_has_item(void);     /* the item id is passed in d0 */
void game_event_object(void);      /* the object slot is passed in d0 */
void game_event_finish(void);      /* the object slot is passed in d0 */
void game_quest_cutscene(void);    /* the quest picture number is passed in d0 */

/* One outdoor dungeon entrance. rom_region_exit_points holds 40 of them, and the region byte
 * of each is the matching entry of rom_region_exits rather than a field here. */
struct region_exit {
    uint16_t x;                /* where the party has to stand, in map pixels, rounded down to 8 */
    uint16_t y;
    uint8_t min_game_level;    /* the entrance does nothing below this ram_game_level */
    uint8_t map;               /* the ram_map the dungeon starts on */
    uint8_t enter_flag;        /* ram_event_flags bit set on the way in, 0 for none */
    uint8_t entrance;          /* which of that map's entry points the party arrives on */
};
_Static_assert(sizeof(struct region_exit) == 8, "region_exit overlays one rom_region_exit_points record");
extern const struct region_exit rom_region_exit_points[40];
extern const uint8_t rom_region_exits[41];

/* One entry of a terrain's random-encounter list. appearing holds the fewest in its high nibble
 * and the most in its low nibble. level is a switch: non-zero lets the monsters cast in battle, and
 * every list in the game uses 1. The first pick's affinity selects a mask from rom_encounter_affinity;
 * a later pick joins the fight only when its own affinity ANDs non-zero with it and its id is not
 * already in the group, so 0 fights alone and a repeated id only adds weight. */
struct encounter_entry {
    uint8_t monster;           /* 0x62 to 0x93, the battle roster */
    uint8_t appearing;
    uint8_t level;
    uint8_t affinity;
};
_Static_assert(sizeof(struct encounter_entry) == 4, "encounter_entry overlays one random-encounter list entry");
extern const struct encounter_entry *const rom_terrain_encounter_groups[13];
extern const uint8_t rom_terrain_encounter_chance[14];
extern const uint8_t rom_terrain_wander_tracks[14];
extern const uint8_t rom_terrain_combat_tracks[14];
extern const uint8_t rom_terrain_win_tracks[14];
extern volatile uint8_t ram_music_track;
extern volatile uint8_t ram_music_on;
extern volatile uint8_t ram_encounter_forced;
extern volatile uint8_t ram_encounter_level;
extern volatile uint16_t ram_encounter_kinds;
extern volatile uint16_t ram_encounter_total;
int game_terrain_id(void);         /* the terrain id comes back in d0 */
void game_music_wander(void);
void game_music_combat(void);
void game_music_win(void);

/* Where one staircase leads. rom_dungeon_stairs is indexed by ram_map_pending minus one. The party
 * arrives at entrance_forward when the destination's number is above the map being left and at
 * entrance_back when it is not, so one record serves both ends of the stair. */
struct dungeon_stairs {
    uint16_t map;
    uint8_t entrance_forward;
    uint8_t entrance_back;
};
_Static_assert(sizeof(struct dungeon_stairs) == 4, "dungeon_stairs overlays one rom_dungeon_stairs record");
extern const struct dungeon_stairs rom_dungeon_stairs[13];

/* What a dungeon map is drawn from. rom_map_table holds one per ram_map. */
struct map_entry {
    uint32_t style;    /* the ten-byte wall and sprite descriptor that maps of the same look share */
    uint32_t graphics; /* the map's graphics block; 0x40 into it is the palette line the map draws with */
    uint32_t layout;   /* the compressed grid of wall bytes, four per tile */
};
_Static_assert(sizeof(struct map_entry) == 12, "map_entry overlays one rom_map_table record");
extern const struct map_entry rom_map_table[38];

/* One thing standing on a dungeon tile. A map's list follows its entry points, three bytes each,
 * and ends with a zero byte. rom_map_directory says where each map's list is. */
struct map_object {
    uint8_t kind;      /* low nibble: 1 item, 2 monster, 3 another moving actor, 5 event, 6 transition */
    uint8_t value;     /* item id, monster type, event action, or, on a transition, one less than the
                        * rom_dungeon_stairs entry it takes; negative to leave by rom_region_exits */
    uint8_t x;         /* tile column */
    uint8_t y;         /* tile row */
    uint8_t amount;    /* by action: gold, damage, quest picture, or the item the step needs */
    uint8_t uses;      /* steps on it before it is taken off the map, 0xFF for never */
    uint8_t extra;     /* the second amount, for the actions that take two */
    uint8_t text;      /* one plus the index into the map's text table, 0 for silent */
    uint8_t gate;      /* ram_event_flags bit that keeps it off the map; 0 means no gate at all, and
                        * 0xFF means test for item value instead. Unlike flag, where 0 is bit 0 */
    uint8_t flag;      /* ram_event_flags bit set once it is used, 0xFF for none */
};
_Static_assert(sizeof(struct map_object) == 10, "map_object overlays one map object record");

void game_save_game(void);
void game_load_game(void);      /* the slot, 0 to 3, is passed in d0, so a C caller has to load d0 with inline asm */
void game_save_ram_read(void);  /* the save-RAM offset in a0, the work RAM to fill in a1, the byte count in d0; the carry comes back set if the read never matched */
void game_save_ram_write(void); /* the work RAM in a0, the save-RAM offset in a1, the byte count in d0; the carry comes back set if the write never read back */
void game_init_save_ram(void);
void game_check_save_ram(void);

/* One saved game: the work RAM from ram_save_state_start on, copied byte for byte. Save RAM holds
 * four of these, at save-RAM offsets 0xB, 0x70D, 0xE0F and 0x1511, and a save or a load moves the
 * whole 0x702 bytes in one go. The four character records fill 0x4AA to the end of the slot, 0x96
 * bytes each, so party_x and party_y below are also the first character's own position. Everything
 * named here has a work RAM symbol of its own; the rest is state no name has been pinned to yet. */
struct save_slot {
    uint8_t unknown_000[0xBC];
    uint16_t region_area;      /* ram_region_area */
    uint8_t unknown_0BE[0x1FD];
    uint8_t region;            /* ram_region */
    uint8_t game_level;        /* ram_game_level */
    uint8_t unknown_2BD[0xA];
    uint8_t show_coords;       /* ram_show_coords */
    uint8_t unknown_2C8[2];
    uint8_t monsters_off;      /* ram_monsters_off */
    uint8_t unknown_2CB[5];
    uint8_t event_flags[0x20]; /* ram_event_flags */
    uint8_t unknown_2F0[0x1BA];
    uint8_t unknown_4AA[0x20]; /* the first character record starts here */
    uint16_t party_x;          /* ram_party_x */
    uint16_t party_y;          /* ram_party_y */
    uint8_t unknown_4CE[0x234];
};
_Static_assert(sizeof(struct save_slot) == 0x702, "save_slot overlays one save slot");
extern volatile struct save_slot ram_save_state_start;

/* The outdoor renderer.
 *
 * An outdoor region is drawn on both VDP planes at once, each from its own cell array: the map
 * column of rom_region_table feeds the plane in front of the party and the terrain column feeds the
 * ground plane behind it. A cell array carries no header. It is 32 by 32 uint16_t, row major, so
 * cell (column, row) is at byte offset row * 64 + column * 2, and each cell covers 24 by 24 tiles,
 * that is 192 by 192 pixels; a region map is therefore 768 tiles, 6144 pixels, square.
 *
 * A cell resolves in three steps, all of them plain array indexing with no compression:
 *
 *   cell      = map[y / 192][x / 192]                      a struct map_block index
 *   metatile  = blocks[cell].metatile[y / 24 % 8][x / 24 % 8]   a struct map_metatile index
 *   tile word = tiles[metatile].tile[y / 8 % 3][x / 8 % 3]      the VDP name table word
 *
 * where blocks and tiles are the blocks and tiles columns of rom_region_table. The tile word is a
 * normal name table word except that bit 15, the VDP priority bit, means blocked on the ground
 * plane: game_map_tile_at reads it for collision, the ground plane's copy in ram_ground_shadow
 * keeps it, and the copy sent to the VDP has it cleared. On the front plane bit 15 is left set, so
 * those tiles draw over the party, and its empty cells are tile 0, which is blank.
 *
 * Tile graphics are raw 32-byte 4bpp tiles, one chunk list per region in rom_graphics_sets, loaded
 * to VRAM 0 by game_load_tiles; only a chunk whose count has bit 15 set goes through
 * game_decompress. Every region shares rom_main_palette, so a new region's tiles have to be drawn
 * to that palette. */

/* One block: 8 by 8 metatiles, the whole of one map cell. */
struct map_block {
    uint16_t metatile[8][8];
};
_Static_assert(sizeof(struct map_block) == 0x80, "map_block overlays one block of a region's blocks column");

/* One metatile: 3 by 3 VDP name table words, 24 pixels square. */
struct map_metatile {
    uint16_t tile[3][3];
};
_Static_assert(sizeof(struct map_metatile) == 18, "map_metatile overlays one entry of a region's tiles column");

/* One of the two planes the outdoors is drawn on: ram_map_layer_front and ram_map_layer_ground,
 * which sit exactly one record apart in work RAM. */
struct map_layer {
    uint16_t x;             /* map pixel the plane's left edge is on */
    uint16_t y;             /* map pixel its top edge is on */
    uint16_t name_table;    /* VDP address of this plane's name table */
    uint16_t hscroll;       /* work RAM address of this plane's horizontal word in ram_scroll */
    uint16_t vscroll;       /* and of its vertical word */
    uint8_t wrap;           /* non-zero lets the plane run past the map edge and wrap round */
    uint8_t frozen;         /* non-zero scrolls the plane without redrawing the rows and columns coming into view */
    uint16_t width;         /* map width in tiles */
    uint16_t height;        /* map height in tiles */
    uint16_t unknown_10[4]; /* read only while wrap is clear, and zero on every outdoor map */
    uint32_t map;           /* the 32 by 32 cell array this plane draws */
    uint16_t row[64];       /* one name table row, built here and transferred from here */
    uint16_t column[32];    /* one name table column, likewise */
    uint16_t step_x;        /* pixels to scroll per frame, 8.8 fixed point */
    uint16_t step_y;
    uint16_t frac_x;        /* how far into the current pixel the plane has scrolled */
    uint16_t frac_y;
    uint16_t unknown_E4[3];
    uint32_t shadow;        /* work RAM copy of the name table, 0 on the plane that keeps none */
};
_Static_assert(sizeof(struct map_layer) == 0xEE, "map_layer overlays one outdoor plane record");
extern volatile struct map_layer ram_map_layer_front;
extern volatile struct map_layer ram_map_layer_ground;

/* One queued VDP transfer. ram_vdp_queue is a ring of 64 that the VBlank handler drains. */
struct vdp_transfer {
    uint16_t kind;   /* high byte: 0 VRAM, 1 colour RAM, 2 scroll RAM, 3 a fill, 4 a VRAM to VRAM copy; low byte: the VDP auto-increment */
    uint32_t source; /* 68000 address the transfer reads from */
    uint16_t dest;   /* VDP address it writes to */
    uint16_t count;  /* words to move */
};
_Static_assert(sizeof(struct vdp_transfer) == 10, "vdp_transfer overlays one ram_vdp_queue record");
extern volatile struct vdp_transfer ram_vdp_queue[64];
extern volatile uint16_t ram_vdp_queue_read;
extern volatile uint16_t ram_vdp_queue_write;
extern volatile uint16_t ram_scroll[4];
extern volatile uint16_t ram_ground_shadow[32][64];
extern volatile uint16_t ram_palette[64];
extern volatile uint8_t ram_gfx_buffer[];
extern volatile uint32_t ram_map_blocks;
extern volatile uint32_t ram_map_metatiles;

extern const uint32_t rom_graphics_sets[18];
extern const uint16_t rom_graphics_vram[18];
extern const uint16_t rom_main_palette[64];

void game_draw_outdoor(void);
void game_draw_outdoor_layer(void);  /* the plane is passed in a0 */
void game_draw_outdoor_row(void);    /* the plane in a0, the name table row in d0 and the map row in d1 */
void game_draw_outdoor_column(void); /* the plane in a0, the name table column in d0 and the map column in d1 */
void game_map_cells_to_tiles(void);  /* the plane in a0, the first map tile in d0 and d1, the output in a1 */
void game_scroll_outdoor(void);      /* the old map position in d3 and the new one in d4 */
void game_map_tile_at(void);         /* the map position is passed in d0 and the tile index comes back in d0, with the blocked flag in the carry */
void game_vdp_queue(void);           /* the arguments are passed in d0, d1 and a0 */
void game_vdp_dma(void);
void game_vdp_wait(void);
void game_load_graphics_set(void);   /* the set number is passed in d0 */
void game_load_tiles(void);          /* the chunk list in a0 and the VRAM address in a1 */
void game_decompress(void);          /* the source in a0 and the destination in a1 */
void game_world_map(void);

#pragma pack(pop)

#endif
