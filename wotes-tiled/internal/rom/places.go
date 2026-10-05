package rom

import "fmt"

// Limits on how far a table without an end is walked before the reader gives up.
const (
	PlaceLimit     = 64
	GraveyardLimit = 64
)

// Record and entry sizes of the place tables.
const (
	placeRecord = 10 // y0, y1, x0, x1 words, a level byte and an action byte
	graveRecord = 8  // a position long and a text long
	pointerSize = 4  // a long to a text or a table
	ShopCount   = 4
)

// Place is one special-place record: a party-pixel rectangle, inclusive on both ends, that fires its
// action once per entry while ram_game_level is at most Level.
type Place struct {
	Index          int
	X0, Y0, X1, Y1 int
	Level, Action  int
}

// Places reads the special-place table.
func (g *Game) Places() ([]Place, error) {
	table, actions := int(g.Layout.Places.Table), g.Layout.Places.Actions
	var out []Place
	for i := 0; ; i++ {
		p := table + placeRecord*i
		if p+placeRecord > len(g.ROM) {
			return nil, fmt.Errorf("rom_special_places record %d runs past the ROM's end", i)
		}
		if g.U16(p) == 0 {
			return out, nil
		}
		if i >= PlaceLimit {
			return nil, fmt.Errorf("rom_special_places has no zero end within %d records", PlaceLimit)
		}
		pl := Place{Index: i, Y0: int(g.U16(p)), Y1: int(g.U16(p + 2)), X0: int(g.U16(p + 4)), X1: int(g.U16(p + 6)), Level: int(g.ROM[p+8]), Action: int(int8(g.ROM[p+9]))}
		if pl.Action < 0 || pl.Action >= actions {
			return nil, fmt.Errorf("rom_special_places record %d: action %d is outside the %d actions", i, pl.Action, actions)
		}
		if pl.X1 < pl.X0 || pl.Y1 < pl.Y0 {
			return nil, fmt.Errorf("rom_special_places record %d: the rectangle x %#x-%#x, y %#x-%#x is inside out", i, pl.X0, pl.X1, pl.Y0, pl.Y1)
		}
		out = append(out, pl)
	}
}

// Speech is the text a special-place action shows at a game level, or nil when it has none there.
func (g *Game) Speech(level, action int) ([]string, error) {
	pt := g.Layout.Places
	if level < 0 || level >= pt.Levels || action < 0 || action >= pt.Actions {
		return nil, fmt.Errorf("speech: level %d, action %d: outside %d levels and %d actions", level, action, pt.Levels, pt.Actions)
	}
	if int(pt.SpeechIndex)+action >= len(g.ROM) {
		return nil, fmt.Errorf("rom_speech_index entry %d is past the ROM's end", action)
	}
	b := int(g.ROM[int(pt.SpeechIndex)+action])
	if b == 0xFF {
		return nil, nil
	}
	entry := int(pt.Speeches) + pointerSize*level
	if entry+pointerSize > len(g.ROM) {
		return nil, fmt.Errorf("rom_city_speeches entry %d runs past the ROM's end", level)
	}
	tbl := int(g.U32(entry))
	if tbl+b+2 > len(g.ROM) {
		return nil, fmt.Errorf("speech table for level %d at %#x is outside the ROM", level, tbl)
	}
	if b >= int(g.U16(tbl)) {
		return nil, nil
	}
	lines, err := g.Message(tbl + int(g.U16(tbl+b)))
	if err != nil {
		return nil, fmt.Errorf("speech table for level %d, action %d: %v", level, action, err)
	}
	return lines, nil
}

// ShopTitle is the title of shop i, 0 to 3.
func (g *Game) ShopTitle(i int) (string, error) {
	if i < 0 || i >= ShopCount {
		return "", fmt.Errorf("shop %d is outside the %d shops", i, ShopCount)
	}
	p := int(g.Layout.Places.ShopTitles) + pointerSize*i
	if p+pointerSize > len(g.ROM) {
		return "", fmt.Errorf("rom_shop_titles entry %d runs past the ROM's end", i)
	}
	return g.CString(int(g.U32(p))), nil
}

// Gravestone is one stone of the castle graveyard.
type Gravestone struct {
	X, Y   int
	Text   []string // the epitaph's lines
	Member int      // -1 or -3: a party member picked from the game level; Text is nil then
}

// Gravestones reads the graveyard table, leaving out stones without a text.
func (g *Game) Gravestones() ([]Gravestone, error) {
	table := int(g.Layout.Places.Graveyard)
	var out []Gravestone
	for i := 0; ; i++ {
		p := table + graveRecord*i
		if p+graveRecord > len(g.ROM) {
			return nil, fmt.Errorf("rom_graveyard record %d runs past the ROM's end", i)
		}
		if int32(g.U32(p)) < 0 {
			return out, nil
		}
		if i >= GraveyardLimit {
			return nil, fmt.Errorf("rom_graveyard has no end within %d records", GraveyardLimit)
		}
		text := int32(g.U32(p + 4))
		gs := Gravestone{X: int(g.U16(p)), Y: int(g.U16(p + 2))}
		switch {
		case text == 0:
			continue
		case text < 0:
			gs.Member = int(text)
		default:
			lines, err := g.Message(int(text))
			if err != nil {
				return nil, fmt.Errorf("rom_graveyard record %d: %v", i, err)
			}
			gs.Text = lines
		}
		out = append(out, gs)
	}
}
