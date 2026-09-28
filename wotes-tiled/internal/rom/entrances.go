package rom

import "fmt"

// The outdoor dungeon entrances: one ram_region byte per entrance in RegionExits, 0xFF ends, and the
// parallel 8-byte records in RegionExitPoints: x and y words, then level, map, flag and entry bytes.
const (
	RegionExits      = 0x10416
	RegionExitPoints = 0x104EC
	EntranceLimit    = 40   // records the tables hold at most
	FillerMap        = 0x3F // the map number of a filler record
)

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
func (r ROM) Entrances() ([]Entrance, error) {
	var out []Entrance
	for i := 0; ; i++ {
		if i > EntranceLimit {
			return nil, fmt.Errorf("rom_region_exits has no 0xFF end within %d records", EntranceLimit)
		}
		region := int(r[RegionExits+i])
		if region == 0xFF {
			return out, nil
		}
		p := RegionExitPoints + 8*i
		e := Entrance{Index: i, Region: region, X: int(r.U16(p)), Y: int(r.U16(p + 2)), Level: int(r[p+4]), Map: int(r[p+5]), Flag: int(r[p+6]), Entry: int(r[p+7])}
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
