package rom

import "fmt"

// FixedEncounters is the table of the fights fixed in place: records of a region word, x and y words
// and a data long, ended by a negative word. The data is a byte of least and most count nibbles, the
// monster id, then x and y words for each monster.
const (
	FixedEncounters = 0x165A0
	EncounterLimit  = 16
	EncounterReach  = 96 // pixels on each axis within which the party starts the fight
)

// Encounter is one fixed fight.
type Encounter struct {
	Index, Region, X, Y int
	Min, Max, Monster   int
	Positions           [][2]int // Max positions, x then y
}

// FixedEncounters reads the table.
func (r ROM) FixedEncounters() ([]Encounter, error) {
	var out []Encounter
	for i := 0; ; i++ {
		p := FixedEncounters + 10*i
		if int16(r.U16(p)) < 0 {
			return out, nil
		}
		if i >= EncounterLimit {
			return nil, fmt.Errorf("rom_fixed_encounters has no end within %d records", EncounterLimit)
		}
		d := int(r.U32(p + 6))
		if d+2 > len(r) {
			return nil, fmt.Errorf("rom_fixed_encounters record %d: data at %#x is outside the ROM", i, d)
		}
		e := Encounter{Index: i, Region: int(r.U16(p)), X: int(r.U16(p + 2)), Y: int(r.U16(p + 4)), Min: int(r[d] >> 4), Max: int(r[d] & 15), Monster: int(r[d+1])}
		if e.Max == 0 || e.Max < e.Min {
			return nil, fmt.Errorf("rom_fixed_encounters record %d: %d to %d monsters", i, e.Min, e.Max)
		}
		if d+2+4*e.Max > len(r) {
			return nil, fmt.Errorf("rom_fixed_encounters record %d: the positions run past the ROM", i)
		}
		for k := 0; k < e.Max; k++ {
			e.Positions = append(e.Positions, [2]int{int(r.U16(d + 2 + 4*k)), int(r.U16(d + 4 + 4*k))})
		}
		out = append(out, e)
	}
}
