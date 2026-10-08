package convert

import (
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/extract"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/tiled"
)

// The kind byte of an emitted trigger record.
const (
	TriggerShop    = 1
	TriggerRestore = 2
	TriggerSpeech  = 3
	TriggerEvent   = 4
)

// LevelAlways is the level byte of a trigger that fires at every game level, the highest there is.
const LevelAlways = 4

// FigureLine is the palette line a map's figures draw on: line 3, the one line no outdoor tile, portrait, party or monster sprite uses.
const FigureLine = 3

// MaxSlot is the highest slot word a figure may hold: the pacer bytes from 0xFF8EE8 that the figure tile loader clears end at 0xFF8EFB, and 0x14 on is the townsfolk's.
const MaxSlot = 0x13

// Trigger is an inclusive party-pixel rectangle; Shop and Text (an index into Result.Texts) are -1 when unused,
// and Event is nil unless the kind is TriggerEvent.
type Trigger struct {
	Name           string
	X0, Y0, X1, Y1 int
	Level, Kind    int
	Shop, Text     int
	Event          *Event
}

// Event is what an event trigger does once, Flag marking it done: it hands over Item (0 for none) and shows
// quest picture Picture (0 for none); on every later entry it shows the text After (an index into Result.Texts, -1 for none).
type Event struct {
	Flag, Item, Picture, After int
	ItemName                   string
}

// Figure is a placed figure; X and Y are the record's party-pixel position, not the tile object's corner.
type Figure struct {
	Name       string
	X, Y       int
	Figure     int
	Moves      string
	Slot       int
	Front      bool
	Definition rom.Definition
}

// Routine is the game routine the movement names.
func (f Figure) Routine() int {
	switch f.Moves {
	case extract.MovesEastWest:
		return rom.NPCPacesX
	case extract.MovesNorthSouth:
		return rom.NPCPacesY
	}
	return rom.NPCAlways
}

// Record is the figure's city-NPC record; the parameter's bits 13 and 14 pick the palette line, bit 31 the front.
func (f Figure) Record(line int) [rom.NPCRecordSize]byte {
	var r [rom.NPCRecordSize]byte
	binary.BigEndian.PutUint16(r[0:], uint16(f.X))
	binary.BigEndian.PutUint16(r[2:], uint16(f.Y))
	param := uint32(line&3) << 13
	if f.Front {
		param |= 1 << 31
	}
	binary.BigEndian.PutUint32(r[4:], param)
	binary.BigEndian.PutUint32(r[8:], uint32(f.Routine()))
	binary.BigEndian.PutUint16(r[12:], uint16(f.Slot))
	copy(r[14:], f.Definition.Bytes[:])
	return r
}

// objects reads the map's object layers and its two object properties into res.
func (c *converter) objects(m *tiled.Map, infos []*tilesetInfo, ll *landLayers, res *Result) error {
	path := c.opt.MapPath
	res.Region, res.HasRegion = tiled.IntProp(m.Properties, extract.PropRegion)
	if res.HasRegion && (res.Region < 0 || res.Region > 255) {
		return fmt.Errorf("%s: %s %d must fit a byte, 0 to 255", path, extract.PropRegion, res.Region)
	}
	if ll.transitions != nil {
		if err := c.entrances(ll.transitions, res); err != nil {
			return err
		}
	}
	if ll.triggers != nil {
		for _, o := range ll.triggers.Objects {
			tr, err := c.trigger(o, res)
			if err != nil {
				return objectErr(path, extract.LayerTriggers, o, err)
			}
			res.Triggers = append(res.Triggers, tr)
		}
	}
	if ll.npcs != nil {
		held := map[int]int{}
		for _, o := range ll.npcs.Objects {
			f, err := c.figure(o, infos, held)
			if err != nil {
				return objectErr(path, extract.LayerNPCs, o, err)
			}
			res.Figures = append(res.Figures, f)
		}
		if err := assignSlots(res.Figures, held); err != nil {
			return fmt.Errorf("%s: layer %s: %v", path, extract.LayerNPCs, err)
		}
	}
	colours, err := figureColours(m)
	if err != nil {
		return fmt.Errorf("%s: %v", path, err)
	}
	if len(res.Figures) > 0 && colours == nil {
		return fmt.Errorf("%s: layer %s: figures need the map property %s, their palette line", path, extract.LayerNPCs, extract.PropFigureColours)
	}
	res.FigureColours = colours
	if ll.encounters != nil {
		return c.spawns(ll.encounters, res)
	}
	return nil
}

// objectErr names the file, layer and object an error is about.
func objectErr(path, layer string, o tiled.Object, err error) error {
	return fmt.Errorf("%s: layer %s: object %d %q: %v", path, layer, o.ID, o.Name, err)
}

// round is an object coordinate as a whole pixel.
func round(v float64) int { return int(math.Round(v)) }

// inMap reports a rectangle inside the map.
func inMap(x0, y0, x1, y1 int) bool {
	return x0 >= 0 && y0 >= 0 && x1 < rom.MapPixels && y1 < rom.MapPixels && x1 >= x0 && y1 >= y0
}

// trigger reads a special-place rectangle and, for a restore, a speech or an event, its text.
func (c *converter) trigger(o tiled.Object, res *Result) (Trigger, error) {
	if err := checkName(o.Name); err != nil {
		return Trigger{}, err
	}
	if o.Point || o.Width <= 0 || o.Height <= 0 || o.Ellipse || len(o.Polygon) != 0 || len(o.Polyline) != 0 || len(o.Text) != 0 || o.GID != 0 || o.Rotation != 0 || o.Template != "" {
		return Trigger{}, fmt.Errorf("a trigger is an unrotated rectangle, not from a template")
	}
	t := Trigger{Name: o.Name, X0: round(o.X), Y0: round(o.Y), Level: LevelAlways, Shop: -1, Text: -1}
	t.X1, t.Y1 = round(o.X+o.Width)-1, round(o.Y+o.Height)-1
	if !inMap(t.X0, t.Y0, t.X1, t.Y1) {
		return Trigger{}, fmt.Errorf("from (%d, %d) to (%d, %d) is outside the %d-pixel map", t.X0, t.Y0, t.X1, t.Y1, rom.MapPixels)
	}
	if v, ok := tiled.IntProp(o.Properties, extract.PropLevel); ok {
		if v < 0 || v > LevelAlways {
			return Trigger{}, fmt.Errorf("%s %d is outside 0 to %d", extract.PropLevel, v, LevelAlways)
		}
		t.Level = v
	}
	text := tiled.StringProp(o.Properties, extract.PropText)
	shop, hasShop := tiled.IntProp(o.Properties, extract.PropShop)
	kind := tiled.StringProp(o.Properties, extract.PropKind)
	if kind != extract.KindEvent {
		for _, p := range []string{extract.PropFlag, extract.PropItem, extract.PropPicture, extract.PropAfter} {
			if _, has := tiled.IntProp(o.Properties, p); has || tiled.StringProp(o.Properties, p) != "" {
				return Trigger{}, fmt.Errorf("%s is for events", p)
			}
		}
	}
	switch kind {
	case extract.KindShop:
		if !hasShop || shop < 0 || shop >= rom.ShopCount {
			return Trigger{}, fmt.Errorf("a shop needs %s from 0 to %d: armour, weapon, magic, caravan", extract.PropShop, rom.ShopCount-1)
		}
		if text != "" {
			return Trigger{}, fmt.Errorf("a shop has no %s", extract.PropText)
		}
		t.Kind, t.Shop = TriggerShop, shop
	case extract.KindRestore, extract.KindSpeech, extract.KindEvent:
		if hasShop {
			return Trigger{}, fmt.Errorf("%s is for shops", extract.PropShop)
		}
		if text == "" {
			a := "a"
			if kind == extract.KindEvent {
				a = "an"
			}
			return Trigger{}, fmt.Errorf("%s %s needs %s, its lines separated by newlines", a, kind, extract.PropText)
		}
		var err error
		if t.Text, err = message(text, res); err != nil {
			return Trigger{}, err
		}
		switch kind {
		case extract.KindRestore:
			t.Kind = TriggerRestore
		case extract.KindSpeech:
			t.Kind = TriggerSpeech
		default:
			t.Kind = TriggerEvent
			if t.Event, err = c.event(o, res); err != nil {
				return Trigger{}, err
			}
		}
	default:
		return Trigger{}, fmt.Errorf("%s %q is not %s, %s, %s or %s", extract.PropKind, kind, extract.KindShop, extract.KindRestore, extract.KindSpeech, extract.KindEvent)
	}
	return t, nil
}

// message checks a text's lines and adds them to the map's texts, returning their index.
func message(text string, res *Result) (int, error) {
	lines := strings.Split(text, "\n")
	if err := rom.CheckMessage(lines); err != nil {
		return 0, err
	}
	res.Texts = append(res.Texts, lines)
	return len(res.Texts) - 1, nil
}

// event reads what an event does: the flag that marks it done, and its optional item, quest picture and after-text.
func (c *converter) event(o tiled.Object, res *Result) (*Event, error) {
	e := &Event{After: -1}
	free := c.opt.Game.Layout.Dungeons.Flags
	flag, has := tiled.IntProp(o.Properties, extract.PropFlag)
	if !has {
		return nil, fmt.Errorf("an event needs %s, the event flag that marks it done, from %#x to %#x", extract.PropFlag, int(free.First), int(free.Last))
	}
	// The game's own quest flags lie outside the free range; an event may set only a free one.
	if flag < int(free.First) || flag > int(free.Last) {
		return nil, fmt.Errorf("%s %#x is one of the game's own flags; an event sets one from %#x to %#x", extract.PropFlag, flag, int(free.First), int(free.Last))
	}
	e.Flag = flag
	if item, has := tiled.IntProp(o.Properties, extract.PropItem); has {
		name, err := c.opt.Game.ItemName(item)
		if err != nil {
			return nil, err
		}
		e.Item, e.ItemName = item, name
	}
	if picture, has := tiled.IntProp(o.Properties, extract.PropPicture); has {
		if picture < 1 || picture > rom.QuestPictures {
			return nil, fmt.Errorf("%s %d is outside 1 to %d, the game's quest pictures", extract.PropPicture, picture, rom.QuestPictures)
		}
		e.Picture = picture
	}
	if after := tiled.StringProp(o.Properties, extract.PropAfter); after != "" {
		var err error
		if e.After, err = message(after, res); err != nil {
			return nil, fmt.Errorf("%s: %v", extract.PropAfter, err)
		}
	}
	return e, nil
}

// standingSlots is the animation slots a standing figure indexes: the game keeps three for them.
const standingSlots = 3

// checkName refuses a name that would end the assembler comment it is written into.
func checkName(name string) error {
	for _, c := range name {
		if c < 0x20 || c > 0x7E {
			return fmt.Errorf("the name %q holds a character outside space to ~", name)
		}
	}
	return nil
}

// figure reads a figure tile object; a slot it names is recorded in held.
func (c *converter) figure(o tiled.Object, infos []*tilesetInfo, held map[int]int) (Figure, error) {
	if err := checkName(o.Name); err != nil {
		return Figure{}, err
	}
	if k := tiled.StringProp(o.Properties, extract.PropKind); k != "" && k != extract.KindNPC {
		return Figure{}, fmt.Errorf("%s %q is not %s; a land map's npcs are figures", extract.PropKind, k, extract.KindNPC)
	}
	if o.GID == 0 || o.Point || o.Template != "" {
		return Figure{}, fmt.Errorf("a figure is a tile object from the figures tileset")
	}
	if o.Rotation != 0 {
		return Figure{}, fmt.Errorf("a figure is an unrotated tile object")
	}
	info, local, h, v, err := lookup(infos, o.GID)
	if err != nil {
		return Figure{}, err
	}
	if info.kind != extract.KindFigures {
		return Figure{}, fmt.Errorf("a tile of %s is not a figure; place figures from rom/%s.tsj", info.ref.Source, extract.FiguresName)
	}
	if h || v {
		return Figure{}, fmt.Errorf("a flipped figure; the game draws every figure one way")
	}
	if shows := tiled.StringProp(o.Properties, extract.PropShows); shows != "" && shows != "always" {
		return Figure{}, fmt.Errorf("shown %q; a land map's figures are always shown", shows)
	}
	def, err := c.opt.Game.FigureDefinition(local)
	if err != nil {
		return Figure{}, err
	}
	moves := tiled.StringProp(o.Properties, extract.PropMoves)
	if moves == "" {
		moves = extract.MovesStands
	}
	switch moves {
	case extract.MovesStands, extract.MovesEastWest, extract.MovesNorthSouth:
	default:
		return Figure{}, fmt.Errorf("%s %q is not %s, %s or %s", extract.PropMoves, moves, extract.MovesStands, extract.MovesEastWest, extract.MovesNorthSouth)
	}
	// A tile object's position is its bottom-left corner; the record's is the sprite's origin.
	f := Figure{Name: o.Name, Figure: local, Moves: moves, Front: tiled.BoolProp(o.Properties, extract.PropFront), Definition: def, X: round(o.X) - def.XOff, Y: round(o.Y) - extract.FigureHeight - def.YOff}
	if !inMap(f.X, f.Y, f.X, f.Y) {
		return Figure{}, fmt.Errorf("(%d, %d) is outside the %d-pixel map", f.X, f.Y, rom.MapPixels)
	}
	if slot, ok := tiled.IntProp(o.Properties, extract.PropSlot); ok {
		if slot < 0 || slot > MaxSlot {
			return Figure{}, fmt.Errorf("%s %d is outside 0 to %d", extract.PropSlot, slot, MaxSlot)
		}
		if moves == extract.MovesStands && slot > standingSlots {
			return Figure{}, fmt.Errorf("%s %d on a standing figure names an animation, 1 to %d", extract.PropSlot, slot, standingSlots)
		}
		if slot == 0 && moves != extract.MovesStands {
			return Figure{}, fmt.Errorf("%s 0 runs no animation, so a pacing figure needs a slot from 1 to %d", extract.PropSlot, MaxSlot)
		}
		if other, taken := held[slot]; taken && slot != 0 {
			return Figure{}, fmt.Errorf("%s %d is held by two figures, %d and %d", extract.PropSlot, slot, other, o.ID)
		}
		held[slot] = o.ID
		f.Slot = slot
	} else {
		f.Slot = -1
	}
	return f, nil
}

// assignSlots reads held as: an object id for a slot a figure named, -1 for one handed out here, 0 for free.
// It gives every pacing figure without a slot the lowest free one from 1; standing figures
// without one take 0, which runs no animation.
func assignSlots(figures []Figure, held map[int]int) error {
	next := 1
	for i := range figures {
		if figures[i].Slot >= 0 {
			continue
		}
		if figures[i].Moves == extract.MovesStands {
			figures[i].Slot = 0
			continue
		}
		for held[next] != 0 {
			next++
		}
		if next > MaxSlot {
			return fmt.Errorf("more pacing figures than the %d slots the game keeps", MaxSlot)
		}
		figures[i].Slot = next
		held[next] = -1
	}
	return nil
}

// figureColours reads the sixteen colour words of wotes.figure_colours, or nil without the property.
func figureColours(m *tiled.Map) ([]uint16, error) {
	s := tiled.StringProp(m.Properties, extract.PropFigureColours)
	if s == "" {
		return nil, nil
	}
	words := strings.Fields(s)
	if len(words) != 16 {
		return nil, fmt.Errorf("%s holds %d colour words, a palette line has 16", extract.PropFigureColours, len(words))
	}
	out := make([]uint16, 16)
	for i, w := range words {
		v, err := strconv.ParseUint(strings.TrimPrefix(strings.ToLower(w), "0x"), 16, 16)
		if err != nil || !strings.HasPrefix(strings.ToLower(w), "0x") || v&^0x0EEE != 0 {
			return nil, fmt.Errorf("%s: %q is not a Genesis colour word, 0x0000 to 0x0EEE with even digits", extract.PropFigureColours, w)
		}
		out[i] = uint16(v)
	}
	return out, nil
}
