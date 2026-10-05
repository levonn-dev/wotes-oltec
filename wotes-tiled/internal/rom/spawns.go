package rom

import "fmt"

// What the game's random encounters allow.
const (
	SpawnSlots   = 8    // the entries of a list; the game draws from all eight
	FirstRolled  = 0x62 // the first monster id the game's group builder accepts
	LastRolled   = 0x93 // the last one
	MaxAppearing = 15   // fewest and most are a nibble each
	MaxAffinity  = 5    // an index into the game's six affinity masks
	spawnEntry   = 4    // a monster byte, the fewest and most nibbles, a level byte and an affinity byte
)

// Spawn is one entry of a terrain's random-encounter list.
type Spawn struct {
	Monster, Min, Max int
	Level, Affinity   int
}

// Terrains reads the terrain of every map cell, row major: what game_terrain_id answers in the city and the valley.
func (g *Game) Terrains() ([]int, error) {
	t := g.Layout.Spawns
	if int(t.Terrain)+len(Cells{}) > len(g.ROM) {
		return nil, fmt.Errorf("rom_terrain_cells runs past the ROM's end")
	}
	out := make([]int, len(Cells{}))
	for i := range out {
		out[i] = int(g.ROM[int(t.Terrain)+i])
		if out[i] >= t.Terrains {
			return nil, fmt.Errorf("rom_terrain_cells cell (%d, %d): terrain %d, the layout has 0 to %d", i%MapCells, i/MapCells, out[i], t.Terrains-1)
		}
	}
	return out, nil
}

// SpawnList reads the list a terrain rolls from, nil for a terrain that rolls none.
func (g *Game) SpawnList(terrain int) ([]Spawn, error) {
	t := g.Layout.Spawns
	if terrain < 0 || terrain >= t.Terrains {
		return nil, fmt.Errorf("terrain %d: the layout has 0 to %d", terrain, t.Terrains-1)
	}
	p := int(t.Lists) + pointerSize*terrain
	if p+pointerSize > len(g.ROM) {
		return nil, fmt.Errorf("rom_terrain_encounter_groups entry %d runs past the ROM's end", terrain)
	}
	list := int(g.U32(p))
	// The roll leaves at terrain 0 before it reads the table.
	if terrain == 0 || list == 0 {
		return nil, nil
	}
	if list+spawnEntry*SpawnSlots > len(g.ROM) {
		return nil, fmt.Errorf("rom_terrain_encounter_groups entry %d: the list at 0x%X runs past the ROM's end", terrain, list)
	}
	out := make([]Spawn, SpawnSlots)
	for i := range out {
		e := g.ROM[list+spawnEntry*i:]
		s := Spawn{Monster: int(e[0]), Min: int(e[1] >> 4), Max: int(e[1] & 15), Level: int(e[2]), Affinity: int(e[3])}
		switch {
		case s.Monster < FirstRolled || s.Monster > LastRolled:
			return nil, fmt.Errorf("rom_terrain_encounter_groups entry %d, slot %d: monster 0x%02X is outside 0x%02X to 0x%02X, the ones the game rolls", terrain, i, s.Monster, FirstRolled, LastRolled)
		case s.Min < 1 || s.Max < s.Min:
			return nil, fmt.Errorf("rom_terrain_encounter_groups entry %d, slot %d: %d to %d monsters", terrain, i, s.Min, s.Max)
		case s.Affinity > MaxAffinity:
			return nil, fmt.Errorf("rom_terrain_encounter_groups entry %d, slot %d: affinity %d is outside 0 to %d", terrain, i, s.Affinity, MaxAffinity)
		}
		out[i] = s
	}
	return out, nil
}

// RolledNames lists, by name, the monsters the game's random encounters may hold.
func (g *Game) RolledNames() map[string][]int {
	names := map[string][]int{}
	for id := FirstRolled; id <= LastRolled; id++ {
		if n := g.MonsterName(id); n != "" {
			names[n] = append(names[n], id)
		}
	}
	return names
}
