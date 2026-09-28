package rom

import "fmt"

// The outdoor special places and the texts their actions show.
const (
	SpecialPlaces  = 0xD3E4 // records of y0, y1, x0, x1 words, a level byte and an action byte; a zero word ends
	ActionCount    = 35
	PlaceLimit     = 64
	SpeechIndex    = 0xDBA0  // one byte per action: the text's word offset in a level's speech table, 0xFF for none
	CitySpeeches   = 0xFAF14 // five longs, one speech table per game level
	GameLevels     = 5
	ShopTitles     = 0xC55E // four longs to the shop titles
	Graveyard      = 0xD7CE // records of a position long and a text long; a negative position ends
	GraveyardLimit = 64
)

// Place is one special-place record: a party-pixel rectangle, inclusive on both ends, that fires its
// action once per entry while ram_game_level is at most Level.
type Place struct {
	Index          int
	X0, Y0, X1, Y1 int
	Level, Action  int
}

// Places reads the special-place table.
func (r ROM) Places() ([]Place, error) {
	var out []Place
	for i := 0; ; i++ {
		p := SpecialPlaces + 10*i
		if r.U16(p) == 0 {
			return out, nil
		}
		if i >= PlaceLimit {
			return nil, fmt.Errorf("rom_special_places has no zero end within %d records", PlaceLimit)
		}
		pl := Place{Index: i, Y0: int(r.U16(p)), Y1: int(r.U16(p + 2)), X0: int(r.U16(p + 4)), X1: int(r.U16(p + 6)), Level: int(r[p+8]), Action: int(int8(r[p+9]))}
		if pl.Action < 0 || pl.Action >= ActionCount {
			return nil, fmt.Errorf("rom_special_places record %d: action %d is outside the %d actions", i, pl.Action, ActionCount)
		}
		if pl.X1 < pl.X0 || pl.Y1 < pl.Y0 {
			return nil, fmt.Errorf("rom_special_places record %d: the rectangle x %#x-%#x, y %#x-%#x is inside out", i, pl.X0, pl.X1, pl.Y0, pl.Y1)
		}
		out = append(out, pl)
	}
}

// Speech is the text a special-place action shows at a game level, or nil when it has none there.
func (r ROM) Speech(level, action int) ([]string, error) {
	if level < 0 || level >= GameLevels || action < 0 || action >= ActionCount {
		return nil, fmt.Errorf("speech: level %d, action %d: outside %d levels and %d actions", level, action, GameLevels, ActionCount)
	}
	b := int(r[SpeechIndex+action])
	if b == 0xFF {
		return nil, nil
	}
	tbl := int(r.U32(CitySpeeches + 4*level))
	if tbl+b+2 > len(r) {
		return nil, fmt.Errorf("speech table for level %d at %#x is outside the ROM", level, tbl)
	}
	if b >= int(r.U16(tbl)) {
		return nil, nil
	}
	lines, err := r.Message(tbl + int(r.U16(tbl+b)))
	if err != nil {
		return nil, fmt.Errorf("speech table for level %d, action %d: %v", level, action, err)
	}
	return lines, nil
}

// ShopTitle is the title of shop i, 0 to 3.
func (r ROM) ShopTitle(i int) string { return r.CString(int(r.U32(ShopTitles + 4*i))) }

// Gravestone is one stone of the castle graveyard.
type Gravestone struct {
	X, Y   int
	Text   []string // the epitaph's lines
	Member int      // -1 or -3: a party member picked from the game level; Text is nil then
}

// Gravestones reads the graveyard table, leaving out stones without a text.
func (r ROM) Gravestones() ([]Gravestone, error) {
	var out []Gravestone
	for i := 0; ; i++ {
		p := Graveyard + 8*i
		if int32(r.U32(p)) < 0 {
			return out, nil
		}
		if i >= GraveyardLimit {
			return nil, fmt.Errorf("rom_graveyard has no end within %d records", GraveyardLimit)
		}
		text := int32(r.U32(p + 4))
		g := Gravestone{X: int(r.U16(p)), Y: int(r.U16(p + 2))}
		switch {
		case text == 0:
			continue
		case text < 0:
			g.Member = int(text)
		default:
			lines, err := r.Message(int(text))
			if err != nil {
				return nil, fmt.Errorf("rom_graveyard record %d: %v", i, err)
			}
			g.Text = lines
		}
		out = append(out, g)
	}
}
