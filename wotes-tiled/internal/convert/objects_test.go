package convert_test

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/convert"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/emit"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/extract"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/tiled"
)

// The fixture's figures: figure 3 and figure 5 are drawn by records 1 and 4, both with offsets -8, -8.
var (
	defFigure3 = rom.Definition{Bytes: [14]byte{0, 0, 0, 8, 0, 0x18, 0xFF, 0xF8, 0x0A, 0, 0x03, 0xCD, 0xFF, 0xF8}, XOff: -8, YOff: -8}
	defFigure5 = rom.Definition{Bytes: [14]byte{0, 0, 0, 8, 0, 0x18, 0xFF, 0xF8, 0x0A, 0, 0x03, 0xDF, 0xFF, 0xF8}, XOff: -8, YOff: -8}
)

func TestConvertObjects(t *testing.T) {
	res, err := convertFixture(t, nil)
	must(t, err)
	if !res.HasRegion || res.Region != 3 {
		t.Fatalf("region %d, %v", res.Region, res.HasRegion)
	}
	if want := []convert.Entrance{{Name: "cave", ID: 3, Index: 3, X: 0x308, Y: 0x408, Map: 2, Entry: 1}}; !reflect.DeepEqual(res.Entrances, want) {
		t.Fatalf("entrances %+v, want %+v", res.Entrances, want)
	}
	wantTriggers := []convert.Trigger{
		{Name: "armour", X0: 0x100, Y0: 0x110, X1: 0x12F, Y1: 0x12F, Level: 3, Kind: convert.TriggerShop, Shop: 1, Text: -1},
		{Name: "pool", X0: 0x200, Y0: 0x210, X1: 0x217, Y1: 0x227, Level: convert.LevelAlways, Kind: convert.TriggerRestore, Shop: -1, Text: 0},
		{Name: "elder", X0: 0x300, Y0: 0x310, X1: 0x317, Y1: 0x310, Level: convert.LevelAlways, Kind: convert.TriggerSpeech, Shop: -1, Text: 1},
	}
	if !reflect.DeepEqual(res.Triggers, wantTriggers) {
		t.Fatalf("triggers %+v, want %+v", res.Triggers, wantTriggers)
	}
	if want := [][]string{{"THE POOL RESTORES YOU."}, {"WELCOME, TRAVELLERS.", "THE HILLS ARE KIND."}}; !reflect.DeepEqual(res.Texts, want) {
		t.Fatalf("texts %q", res.Texts)
	}
	wantFigures := []convert.Figure{
		{Name: "elder", X: 0x300, Y: 0x310, Figure: 3, Moves: extract.MovesStands, Definition: defFigure3},
		{Name: "walker", X: 0x400, Y: 0x410, Figure: 5, Moves: extract.MovesEastWest, Slot: 1, Definition: defFigure5},
	}
	if !reflect.DeepEqual(res.Figures, wantFigures) {
		t.Fatalf("figures %+v, want %+v", res.Figures, wantFigures)
	}
	if len(res.FigureColours) != 16 || res.FigureColours[0] != 0 || res.FigureColours[2] != 0x0246 || res.FigureColours[15] != 0x0224 {
		t.Fatalf("colours %#x", res.FigureColours)
	}
	// A record: position, the parameter selecting the palette line, the routine, the slot, the definition.
	rec := res.Figures[1].Record(1)
	want := []byte{0x04, 0x00, 0x04, 0x10, 0, 0, 0x20, 0, 0, 0, 0xE6, 0xEA, 0, 1}
	if string(rec[:14]) != string(want) || string(rec[14:]) != string(defFigure5.Bytes[:]) {
		t.Fatalf("record % x", rec)
	}
	if r := res.Figures[0].Record(3); r[6] != 0x60 || r[10] != 0xE6 || r[11] != 0x66 || r[13] != 0 {
		t.Fatalf("standing record on line 3: % x", r)
	}
}

// A filler record belongs to no region, so any map may place it.
func TestEntranceFiller(t *testing.T) {
	res, err := convertFixture(t, func(_ string, m *tiled.Map) {
		m.Layers[4].Objects[0].Properties = []tiled.Property{tiled.Str(extract.PropKind, extract.KindEntrance), tiled.Int(extract.PropIndex, 2), tiled.Int(extract.PropMap, 2), tiled.Int(extract.PropEntry, 1)}
	})
	must(t, err)
	if len(res.Entrances) != 1 || res.Entrances[0].Index != 2 {
		t.Fatalf("entrances %+v", res.Entrances)
	}
}

// Pacing figures without a slot take distinct slots from 1; a given slot is kept and skipped by the others.
func TestFigureSlots(t *testing.T) {
	res, err := convertFixture(t, func(_ string, m *tiled.Map) {
		l := &m.Layers[6]
		l.Objects[0].Properties = []tiled.Property{tiled.Str(extract.PropMoves, extract.MovesNorthSouth), tiled.Int(extract.PropSlot, 1)}
		third := l.Objects[1]
		third.ID, third.Name, third.X = 9, "third", 0x500-8
		l.Objects = append(l.Objects, third)
		m.NextObjectID = 10
	})
	must(t, err)
	if len(res.Figures) != 3 || res.Figures[0].Slot != 1 || res.Figures[1].Slot != 2 || res.Figures[2].Slot != 3 {
		t.Fatalf("slots %d %d %d", res.Figures[0].Slot, res.Figures[1].Slot, res.Figures[2].Slot)
	}
}

// A standing figure may name a slot; the pacers that take one automatically skip it.
func TestFigureStandingSlot(t *testing.T) {
	res, err := convertFixture(t, func(_ string, m *tiled.Map) {
		m.Layers[6].Objects[0].Properties = []tiled.Property{tiled.Int(extract.PropSlot, 1)}
	})
	must(t, err)
	if res.Figures[0].Slot != 1 || res.Figures[1].Slot != 2 {
		t.Fatalf("slots %d %d", res.Figures[0].Slot, res.Figures[1].Slot)
	}
}

// wotes.front sets the parameter's bit 31, which draws the figure in front of every other sprite.
func TestFigureFront(t *testing.T) {
	res, err := convertFixture(t, func(_ string, m *tiled.Map) {
		m.Layers[6].Objects[0].Properties = []tiled.Property{tiled.Bool(extract.PropFront, true)}
	})
	must(t, err)
	if !res.Figures[0].Front || res.Figures[1].Front {
		t.Fatalf("front flags %v %v", res.Figures[0].Front, res.Figures[1].Front)
	}
	if r := res.Figures[0].Record(3); r[4] != 0x80 || r[6] != 0x60 {
		t.Fatalf("record parameter % x", r[4:8])
	}
}

// A map without object layers, or with empty ones, converts as before.
func TestNoObjects(t *testing.T) {
	res, err := convertFixture(t, func(_ string, m *tiled.Map) {
		m.Layers = m.Layers[:4]
		m.Properties = m.Properties[:1]
	})
	must(t, err)
	if res.HasRegion || len(res.Entrances) != 0 || len(res.Triggers) != 0 || len(res.Texts) != 0 || len(res.Figures) != 0 || res.FigureColours != nil {
		t.Fatalf("objects from nowhere: %+v", res)
	}
	res, err = convertFixture(t, func(_ string, m *tiled.Map) {
		for i := 4; i < 7; i++ {
			m.Layers[i].Objects = nil
		}
	})
	must(t, err)
	if len(res.Entrances) != 0 || len(res.Triggers) != 0 || len(res.Figures) != 0 || len(res.FigureColours) != 16 {
		t.Fatalf("empty layers: %+v", res)
	}
}

func TestObjectErrors(t *testing.T) {
	transitions, triggers, npcs := 4, 5, 6
	set := func(o *tiled.Object, ps ...tiled.Property) { o.Properties = ps }
	cases := []struct {
		name string
		want string
		edit func(m *tiled.Map)
	}{
		{"entrance without region", "layer transitions: object 3 \"cave\": an entrance needs the map property wotes.region", func(m *tiled.Map) {
			m.Properties = []tiled.Property{m.Properties[0], m.Properties[2]}
		}},
		{"entrance that leads nowhere", "an entrance names where it leads: a dungeon map and its entry point in wotes.dungeon and wotes.to, or one of the game's levels and its entry point in wotes.map and wotes.entry", func(m *tiled.Map) {
			set(&m.Layers[transitions].Objects[0], tiled.Str(extract.PropKind, extract.KindEntrance))
		}},
		{"entrance of another region", "entrance 0 belongs to region 1, not the map's region 3", func(m *tiled.Map) {
			set(&m.Layers[transitions].Objects[0], tiled.Str(extract.PropKind, extract.KindEntrance), tiled.Int(extract.PropIndex, 0), tiled.Int(extract.PropMap, 2), tiled.Int(extract.PropEntry, 1))
		}},
		{"entrance outside the table", "entrance 40 is outside the 40 records", func(m *tiled.Map) {
			set(&m.Layers[transitions].Objects[0], tiled.Str(extract.PropKind, extract.KindEntrance), tiled.Int(extract.PropIndex, 40))
		}},
		{"entrance past the end", "entrance 5 is past the end of rom_region_exits", func(m *tiled.Map) {
			set(&m.Layers[transitions].Objects[0], tiled.Str(extract.PropKind, extract.KindEntrance), tiled.Int(extract.PropIndex, 5))
		}},
		{"entrance twice", "entrance 3 is placed twice", func(m *tiled.Map) {
			l := &m.Layers[transitions]
			second := l.Objects[0]
			second.ID = 9
			l.Objects = append(l.Objects, second)
		}},
		{"entrance not a point", "an entrance is a point object", func(m *tiled.Map) {
			m.Layers[transitions].Objects[0].Point = false
			m.Layers[transitions].Objects[0].Width, m.Layers[transitions].Objects[0].Height = 8, 8
		}},
		{"zone on a land map", `wotes.kind "zone" is not entrance; a land map's transitions are entrance points`, func(m *tiled.Map) {
			set(&m.Layers[transitions].Objects[0], tiled.Str(extract.PropKind, extract.KindZone), tiled.Int(extract.PropIndex, 3))
		}},
		{"trigger kind", `layer triggers: object 4 "armour": wotes.kind "trigger" is not shop, restore or speech`, func(m *tiled.Map) {
			set(&m.Layers[triggers].Objects[0], tiled.Str(extract.PropKind, extract.KindTrigger))
		}},
		{"shop index", "a shop needs wotes.shop from 0 to 3", func(m *tiled.Map) {
			set(&m.Layers[triggers].Objects[0], tiled.Str(extract.PropKind, extract.KindShop), tiled.Int(extract.PropShop, 4))
		}},
		{"shop without index", "a shop needs wotes.shop from 0 to 3", func(m *tiled.Map) {
			set(&m.Layers[triggers].Objects[0], tiled.Str(extract.PropKind, extract.KindShop))
		}},
		{"shop with text", "a shop has no wotes.text", func(m *tiled.Map) {
			set(&m.Layers[triggers].Objects[0], tiled.Str(extract.PropKind, extract.KindShop), tiled.Int(extract.PropShop, 0), tiled.Str(extract.PropText, "BUY"))
		}},
		{"speech without text", `object 6 "elder": a speech needs wotes.text, its lines separated by newlines`, func(m *tiled.Map) {
			set(&m.Layers[triggers].Objects[2], tiled.Str(extract.PropKind, extract.KindSpeech))
		}},
		{"restore without text", `object 5 "pool": a restore needs wotes.text`, func(m *tiled.Map) {
			set(&m.Layers[triggers].Objects[1], tiled.Str(extract.PropKind, extract.KindRestore))
		}},
		{"speech with shop", "wotes.shop is for shops", func(m *tiled.Map) {
			set(&m.Layers[triggers].Objects[2], tiled.Str(extract.PropKind, extract.KindSpeech), tiled.Str(extract.PropText, "HELLO"), tiled.Int(extract.PropShop, 1))
		}},
		{"lower-case text", `object 6 "elder": line 1: "e" is not a character the font draws`, func(m *tiled.Map) {
			set(&m.Layers[triggers].Objects[2], tiled.Str(extract.PropKind, extract.KindSpeech), tiled.Str(extract.PropText, "Hello"))
		}},
		{"bad character", `"~" is not a character the font draws`, func(m *tiled.Map) {
			set(&m.Layers[triggers].Objects[2], tiled.Str(extract.PropKind, extract.KindSpeech), tiled.Str(extract.PropText, "HELLO~"))
		}},
		{"long line", "the box shows", func(m *tiled.Map) {
			set(&m.Layers[triggers].Objects[2], tiled.Str(extract.PropKind, extract.KindSpeech), tiled.Str(extract.PropText, strings.Repeat("A", rom.MessageWidth+1)))
		}},
		{"trigger level", "wotes.level 5 is outside 0 to 4", func(m *tiled.Map) {
			set(&m.Layers[triggers].Objects[0], tiled.Str(extract.PropKind, extract.KindShop), tiled.Int(extract.PropShop, 0), tiled.Int(extract.PropLevel, 5))
		}},
		{"trigger is a point", `object 4 "armour": a trigger is an unrotated rectangle`, func(m *tiled.Map) {
			m.Layers[triggers].Objects[0].Point = true
		}},
		{"trigger outside", "outside the 6144-pixel map", func(m *tiled.Map) {
			m.Layers[triggers].Objects[0].X = 6140
		}},
		{"figure not a tile object", `layer npcs: object 7 "elder": a figure is a tile object from the figures tileset`, func(m *tiled.Map) {
			m.Layers[npcs].Objects[0].GID = 0
		}},
		{"figure from another tileset", "is not a figure; place figures from rom/city-figures.tsj", func(m *tiled.Map) {
			m.Layers[npcs].Objects[0].GID = gValley1
		}},
		{"flipped figure", "a flipped figure", func(m *tiled.Map) {
			m.Layers[npcs].Objects[0].GID = tiled.MakeGID(gFigures+3, true, false)
		}},
		{"undrawn figure", "no record of rom_city_npcs draws figure 2", func(m *tiled.Map) {
			m.Layers[npcs].Objects[0].GID = gFigures + 2
		}},
		{"figure movement", `wotes.moves "dances" is not stands, east-west or north-south`, func(m *tiled.Map) {
			set(&m.Layers[npcs].Objects[0], tiled.Str(extract.PropMoves, "dances"))
		}},
		{"figure shown by a flag", `shown "flag 0xFF8F51 bit 2"; a land map's figures are always shown`, func(m *tiled.Map) {
			set(&m.Layers[npcs].Objects[0], tiled.Str(extract.PropShows, "flag 0xFF8F51 bit 2"))
		}},
		{"townsfolk point", `wotes.kind "townsfolk" is not npc`, func(m *tiled.Map) {
			set(&m.Layers[npcs].Objects[0], tiled.Str(extract.PropKind, extract.KindTownsfolk))
		}},
		{"figure slot taken twice", "wotes.slot 1 is held by two figures", func(m *tiled.Map) {
			set(&m.Layers[npcs].Objects[0], tiled.Int(extract.PropSlot, 1))
			set(&m.Layers[npcs].Objects[1], tiled.Str(extract.PropMoves, extract.MovesEastWest), tiled.Int(extract.PropSlot, 1))
		}},
		{"figure slot range", "wotes.slot 99 is outside", func(m *tiled.Map) {
			set(&m.Layers[npcs].Objects[0], tiled.Int(extract.PropSlot, 99))
		}},
		{"figure outside", "outside the 6144-pixel map", func(m *tiled.Map) {
			m.Layers[npcs].Objects[0].Y = 6200
		}},
		{"figures without colours", "figures need the map property wotes.figure_colours", func(m *tiled.Map) {
			m.Properties = m.Properties[:2]
		}},
		{"fifteen colours", "wotes.figure_colours holds 15 colour words, a palette line has 16", func(m *tiled.Map) {
			m.Properties[2] = tiled.Str(extract.PropFigureColours, strings.TrimSuffix(fixtureColours, " 0x0224"))
		}},
		{"odd colour", `wotes.figure_colours: "0x0123" is not a Genesis colour word`, func(m *tiled.Map) {
			m.Properties[2] = tiled.Str(extract.PropFigureColours, strings.Replace(fixtureColours, "0x0222", "0x0123", 1))
		}},
		{"region above a byte", "wotes.region 256 must fit a byte", func(m *tiled.Map) {
			m.Properties[1] = tiled.Int(extract.PropRegion, 256)
		}},
		{"region below a byte", "wotes.region -1 must fit a byte", func(m *tiled.Map) {
			m.Properties[1] = tiled.Int(extract.PropRegion, -1)
		}},
		{"standing figure slot above 3", "wotes.slot 4 on a standing figure names an animation, 1 to 3", func(m *tiled.Map) {
			set(&m.Layers[npcs].Objects[0], tiled.Int(extract.PropSlot, 4))
		}},
		{"rotated figure", "a figure is an unrotated tile object", func(m *tiled.Map) {
			m.Layers[npcs].Objects[0].Rotation = 90
		}},
		{"control character in a name", "holds a character outside space to ~", func(m *tiled.Map) {
			m.Layers[npcs].Objects[0].Name = "elder\nrts"
		}},
		{"pacing figure on slot 0", "wotes.slot 0 runs no animation", func(m *tiled.Map) {
			set(&m.Layers[npcs].Objects[1], tiled.Str(extract.PropMoves, extract.MovesEastWest), tiled.Int(extract.PropSlot, 0))
		}},
		{"too many pacing figures", "more pacing figures than the 19 slots", func(m *tiled.Map) {
			l := &m.Layers[npcs]
			for i := 0; i < convert.MaxSlot; i++ {
				p := l.Objects[1]
				p.ID = 20 + i
				l.Objects = append(l.Objects, p)
			}
		}},
		{"transitions not an object layer", "layer transitions must be an object layer", func(m *tiled.Map) {
			l := &m.Layers[transitions]
			l.Type, l.Width, l.Height = "tilelayer", extract.MapSize, extract.MapSize
			l.Objects = nil
			l.SetData(make([]uint32, extract.MapSize*extract.MapSize))
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := convertFixture(t, func(_ string, m *tiled.Map) { c.edit(m) })
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("want an error containing %q, got %v", c.want, err)
			}
		})
	}
}

// An extracted map carries the ROM's own object kinds, which a land map refuses; without its object
// layers it converts as terrain.
func TestExtractedCityObjects(t *testing.T) {
	dir, g := fixture(t)
	city := filepath.Join(dir, "rom", "city.tmj")
	_, err := convert.Convert(convert.Options{Game: g, MapPath: city})
	if err == nil || !strings.Contains(err.Error(), "layer transitions") || !strings.Contains(err.Error(), `"tileset"`) {
		t.Fatalf("the extracted city must be refused at its first ROM-only object, got %v", err)
	}
	terrainOnly(t, city)
	res, err := convert.Convert(convert.Options{Game: g, MapPath: city})
	must(t, err)
	if len(res.Places) != 0 || len(res.Figures) != 0 || res.Counts.Solid != 0 {
		t.Fatalf("an extracted map has no objects of its own: %+v", res)
	}
}

// terrainOnly rewrites a map with its tile layers and places only.
func terrainOnly(t *testing.T, path string) {
	t.Helper()
	m, err := tiled.ReadMap(path)
	must(t, err)
	var kept []tiled.Layer
	for _, l := range m.Layers {
		switch l.Name {
		case extract.LayerGround, extract.LayerFront, extract.LayerSolid, extract.LayerPlaces:
			kept = append(kept, l)
		}
	}
	m.Layers = kept
	must(t, tiled.WriteMap(path, m))
}

// An entrance position rounds down to a multiple of 8, never to the nearest pixel first.
func TestEntranceRoundsDown(t *testing.T) {
	res, err := convertFixture(t, func(_ string, m *tiled.Map) {
		m.Layers[4].Objects[0].X, m.Layers[4].Objects[0].Y = 6143.6, 7.6
	})
	must(t, err)
	if len(res.Entrances) != 1 || res.Entrances[0].X != 6136 || res.Entrances[0].Y != 0 {
		t.Fatalf("entrances %+v", res.Entrances)
	}
}

// The fixture converted as another map of another region, with that region's entrance index, emits the
// same tables; only the map's name and the record its entrance takes in the entrance tables differ.
func TestConvertUnderAnotherName(t *testing.T) {
	dir, g := fixture(t)
	first, err := convert.Convert(convert.Options{Game: g, MapPath: filepath.Join(dir, "test.tmj")})
	must(t, err)
	m, err := tiled.ReadMap(filepath.Join(dir, "test.tmj"))
	must(t, err)
	for i := range m.Tilesets {
		if m.Tilesets[i].Source == "gen/test-metatiles.tsj" {
			m.Tilesets[i].Source = "gen/hills-metatiles.tsj"
		}
	}
	for i := range m.Properties {
		if m.Properties[i].Name == extract.PropRegion {
			m.Properties[i] = tiled.Int(extract.PropRegion, 1)
		}
	}
	for _, l := range m.Layers {
		if l.Name == extract.LayerTransitions {
			l.Objects[0].Properties = []tiled.Property{tiled.Str(extract.PropKind, extract.KindEntrance), tiled.Int(extract.PropIndex, 0), tiled.Int(extract.PropMap, 2), tiled.Int(extract.PropEntry, 1)}
		}
	}
	must(t, tiled.WriteMap(filepath.Join(dir, "hills.tmj"), m))
	defs, err := os.ReadFile(filepath.Join(dir, "test-metatiles.tmj"))
	must(t, err)
	must(t, os.WriteFile(filepath.Join(dir, "hills-metatiles.tmj"), defs, 0o644))
	second, err := convert.Convert(convert.Options{Game: g, MapPath: filepath.Join(dir, "hills.tmj")})
	must(t, err)
	// cut splits the assembler round the two entrance tables.
	cut := func(s string) (before, tables, after string) {
		before, rest, _ := strings.Cut(s, "test_entrance_regions:")
		tables, after, _ = strings.Cut(rest, "test_levels:")
		return before, tables, after
	}
	a0, at, a1 := cut(emit.Assembler(first, "maps/land.tmj"))
	b0, bt, b1 := cut(strings.ReplaceAll(emit.Assembler(second, "maps/land.tmj"), "hills_", "test_"))
	if a0 != b0 || a1 != b1 || a1 == "" {
		t.Fatalf("the two maps differ beyond their names and their entrance tables:\n%s\n----\n%s", a0+a1, b0+b1)
	}
	if !strings.Contains(at, "    .byte   3 | entrance 3, cave\n") || !strings.Contains(bt, "    .byte   1 | entrance 0, cave\n    .incbin \"clean.md\", 0x10417, 3\n") {
		t.Fatalf("the entrance tables:\n%s\n----\n%s", at, bt)
	}
}

// TestCityFiguresReal converts the extracted city's always-shown figures and requires their records byte
// for byte as rom_city_npcs holds them: the format, the position, the routine and the definition. The slot
// word is left out: the city's records share pacer slots, which two land figures may not, so the converter
// assigns its own. It needs the clean ROM in WOTES_CLEAN_ROM.
func TestCityFiguresReal(t *testing.T) {
	path := os.Getenv("WOTES_CLEAN_ROM")
	if path == "" {
		t.Skip("WOTES_CLEAN_ROM not set")
	}
	l := rom.Clean
	g, err := rom.Load(path, &l)
	must(t, err)
	dir := t.TempDir()
	must(t, extract.Run(g, dir))
	city := filepath.Join(dir, "rom", "city.tmj")
	m, err := tiled.ReadMap(city)
	must(t, err)
	var kept []tiled.Layer
	for _, layer := range m.Layers {
		switch layer.Name {
		case extract.LayerGround, extract.LayerFront, extract.LayerSolid, extract.LayerPlaces:
			kept = append(kept, layer)
		case extract.LayerNPCs:
			var figures []tiled.Object
			for _, o := range layer.Objects {
				if tiled.StringProp(o.Properties, extract.PropKind) == extract.KindNPC && tiled.StringProp(o.Properties, extract.PropShows) == "always" {
					var props []tiled.Property
					for _, p := range o.Properties {
						if p.Name != extract.PropSlot {
							props = append(props, p)
						}
					}
					o.Properties = props
					figures = append(figures, o)
				}
			}
			layer.Objects = figures
			kept = append(kept, layer)
		}
	}
	m.Layers = kept
	m.Properties = append(m.Properties, tiled.Str(extract.PropFigureColours, fixtureColours))
	must(t, tiled.WriteMap(city, m))
	res, err := convert.Convert(convert.Options{Game: g, MapPath: city})
	must(t, err)
	records, err := g.CityNPCs()
	must(t, err)
	var always []rom.NPC
	for _, n := range records {
		if n.Routine == rom.NPCAlways || n.Routine == rom.NPCPacesX || n.Routine == rom.NPCPacesY {
			always = append(always, n)
		}
	}
	if len(res.Figures) != len(always) || len(always) < 20 {
		t.Fatalf("%d figures converted, %d always-shown records", len(res.Figures), len(always))
	}
	base := int(l.Figures.Records)
	for i, f := range res.Figures {
		p := base + rom.NPCRecordSize*always[i].Index
		want := g.ROM[p : p+rom.NPCRecordSize]
		got := f.Record(rom.FigurePalette)
		if !bytes.Equal(got[:12], want[:12]) || !bytes.Equal(got[14:], want[14:]) {
			t.Errorf("record %d (%s): converted % x, ROM % x", always[i].Index, f.Name, got, want)
		}
	}
}
