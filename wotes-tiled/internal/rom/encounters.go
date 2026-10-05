package rom

import "fmt"

const (
	encounterRecord = 10 // a region word, x and y words and a data long
	EncounterLimit  = 16
	EncounterReach  = 96 // pixels on each axis within which the party starts the fight
)

// Encounter is one fixed fight.
type Encounter struct {
	Index, Region, X, Y int
	Min, Max, Monster   int
	Positions           [][2]int // Max positions, x then y
}

// FixedEncounters reads the table: its data is a byte of least and most count nibbles, the monster id,
// then x and y words for each monster.
func (g *Game) FixedEncounters() ([]Encounter, error) {
	table := int(g.Layout.Fights.Table)
	var out []Encounter
	for i := 0; ; i++ {
		p := table + encounterRecord*i
		if p+encounterRecord > len(g.ROM) {
			return nil, fmt.Errorf("rom_fixed_encounters record %d runs past the ROM's end", i)
		}
		if int16(g.U16(p)) < 0 {
			return out, nil
		}
		if i >= EncounterLimit {
			return nil, fmt.Errorf("rom_fixed_encounters has no end within %d records", EncounterLimit)
		}
		d := int(g.U32(p + 6))
		if d+2 > len(g.ROM) {
			return nil, fmt.Errorf("rom_fixed_encounters record %d: data at %#x is outside the ROM", i, d)
		}
		e := Encounter{Index: i, Region: int(g.U16(p)), X: int(g.U16(p + 2)), Y: int(g.U16(p + 4)), Min: int(g.ROM[d] >> 4), Max: int(g.ROM[d] & 15), Monster: int(g.ROM[d+1])}
		if e.Max == 0 || e.Max < e.Min {
			return nil, fmt.Errorf("rom_fixed_encounters record %d: %d to %d monsters", i, e.Min, e.Max)
		}
		if d+2+4*e.Max > len(g.ROM) {
			return nil, fmt.Errorf("rom_fixed_encounters record %d: the positions run past the ROM", i)
		}
		for k := 0; k < e.Max; k++ {
			e.Positions = append(e.Positions, [2]int{int(g.U16(d + 2 + 4*k)), int(g.U16(d + 4 + 4*k))})
		}
		out = append(out, e)
	}
}
