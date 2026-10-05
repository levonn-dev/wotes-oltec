package rom

import "fmt"

// FillerMap is the map number of a filler entrance record.
const FillerMap = 0x3F

// entranceRecord is the bytes of one rom_region_exit_points record.
const entranceRecord = 8

// Entrance is one outdoor position that opens a dungeon map.
type Entrance struct {
	Index       int  // the record's position in both tables
	Region      int  // the ram_region value the entrance belongs to
	X, Y        int  // in party pixels
	Level       int  // the highest ram_game_level admitted
	Map, Entry  int  // the dungeon map and its entry point
	Flag        int  // the event flag set on entry, 0 for none
	Unreachable bool // an earlier record has the same region and position and is taken first
}

// Entrances reads the entrance tables, leaving fillers out.
func (g *Game) Entrances() ([]Entrance, error) {
	exits, points, limit := int(g.Layout.Entrances.Regions), int(g.Layout.Entrances.Points), g.Layout.Entrances.Count
	var out []Entrance
	for i := 0; ; i++ {
		if i > limit {
			return nil, fmt.Errorf("rom_region_exits has no 0xFF end within %d records", limit)
		}
		if exits+i >= len(g.ROM) {
			return nil, fmt.Errorf("rom_region_exits record %d runs past the ROM's end", i)
		}
		region := int(g.ROM[exits+i])
		if region == 0xFF {
			return out, nil
		}
		p := points + entranceRecord*i
		if p+entranceRecord > len(g.ROM) {
			return nil, fmt.Errorf("rom_region_exit_points record %d runs past the ROM's end", i)
		}
		e := Entrance{Index: i, Region: region, X: int(g.U16(p)), Y: int(g.U16(p + 2)), Level: int(g.ROM[p+4]), Map: int(g.ROM[p+5]), Flag: int(g.ROM[p+6]), Entry: int(g.ROM[p+7])}
		if e.Map == FillerMap {
			continue
		}
		for _, earlier := range out {
			if earlier.Region == e.Region && earlier.X == e.X && earlier.Y == e.Y {
				e.Unreachable = true
				break
			}
		}
		out = append(out, e)
	}
}

// EntranceAt reads record i of the entrance tables, fillers included, so a caller can tell a record a
// patch may take over (Map == FillerMap) from one another region owns.
func (g *Game) EntranceAt(i int) (Entrance, error) {
	exits, points, count := int(g.Layout.Entrances.Regions), int(g.Layout.Entrances.Points), g.Layout.Entrances.Count
	if i < 0 || i >= count {
		return Entrance{}, fmt.Errorf("entrance %d is outside the %d records of rom_region_exit_points", i, count)
	}
	p := points + entranceRecord*i
	if exits+i >= len(g.ROM) || p+entranceRecord > len(g.ROM) {
		return Entrance{}, fmt.Errorf("rom_region_exit_points record %d runs past the ROM's end", i)
	}
	for k := 0; k <= i; k++ {
		if g.ROM[exits+k] == 0xFF {
			return Entrance{}, fmt.Errorf("entrance %d is past the end of rom_region_exits, which holds %d records", i, k)
		}
	}
	return Entrance{Index: i, Region: int(g.ROM[exits+i]), X: int(g.U16(p)), Y: int(g.U16(p + 2)), Level: int(g.ROM[p+4]), Map: int(g.ROM[p+5]), Flag: int(g.ROM[p+6]), Entry: int(g.ROM[p+7])}, nil
}
