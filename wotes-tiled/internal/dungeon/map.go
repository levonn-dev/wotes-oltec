package dungeon

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/extract"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
)

// The properties a transition names where it leads with.
const (
	PropTo      = "wotes.to"
	PropDungeon = "wotes.dungeon"
)

// itemKind is an item record's kind byte in every level of the game.
const itemKind = 0x31

// Map is a dungeon map as drawn: one grid and the caves it falls into.
type Map struct {
	Name  string
	Path  string
	Look  extract.Look
	Grid  rom.Grid
	Caves []*Cave // in the order of their first entry points
}

// Cave is a part of the grid a party can walk, which the game is given as one level.
type Cave struct {
	Tiles   int      // the tiles a party can stand on
	Entries []Entry  // in object id order
	Records []Record // in object id order
	Texts   []string // by a record's Text less one, lines joined with \n
}

// Entry is an entry point: where a party arrives and the way it faces, 0 north to 3 west.
type Entry struct {
	Name         string
	X, Y, Facing int
}

// Record is one object as the game's list holds it. A transition named for where it leads keeps the name
// in To and leaves Value to whoever binds the names.
type Record struct {
	Name    string // the object's, for errors and the report
	ID      int    // the object's id in Tiled
	Kind    int    // the kind byte
	Value   int
	X, Y    int
	Amount  int
	Uses    int
	Extra   int
	Text    int // into the cave's texts from 1, 0 for none
	Gate    int
	Flag    int
	Exit    bool   // a transition that leaves the dungeon; a staircase otherwise
	To      string // an exit's entrance or a staircase's entry point, by name
	Dungeon string // the map a staircase's entry point is on, "" for this one
}

// Bytes is the record's ten bytes.
func (r Record) Bytes() [10]byte {
	return [10]byte{byte(r.Kind), byte(r.Value), byte(r.X), byte(r.Y), byte(r.Amount), byte(r.Uses), byte(r.Extra), byte(r.Text), byte(r.Gate), byte(r.Flag)}
}

// Transition reports a staircase or an exit.
func (r Record) Transition() bool { return r.Kind&0x0F == rom.ObjectTransition }

// Entry finds one of the map's entry points by name: its cave and its number there.
func (m *Map) Entry(name string) (cave, entry int, ok bool) {
	for c, cv := range m.Caves {
		for e, en := range cv.Entries {
			if en.Name == name {
				return c, e, true
			}
		}
	}
	return 0, 0, false
}

// checkText reports a text the dungeon's box cannot show.
func checkText(text string) error {
	lines := strings.Split(text, "\n")
	if len(lines) > rom.TextLines {
		return fmt.Errorf("the text has %d lines, a dungeon text may have %d", len(lines), rom.TextLines)
	}
	for i, l := range lines {
		if n := utf8.RuneCountInString(l); n > rom.TextColumns {
			return fmt.Errorf("line %d of the text has %d characters, the box shows %d", i+1, n, rom.TextColumns)
		}
		for _, c := range l {
			if c < 0x20 || c > rom.LastGlyph {
				return fmt.Errorf("line %d of the text has %q, which the dungeon font cannot draw", i+1, c)
			}
		}
	}
	return nil
}
