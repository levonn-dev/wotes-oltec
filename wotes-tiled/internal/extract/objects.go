package extract

import (
	"fmt"
	"strings"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/tiled"
)

// The object layers of an extracted map, in order after places.
const (
	LayerTransitions = "transitions"
	LayerTriggers    = "triggers"
	LayerNPCs        = "npcs"
	LayerEncounters  = "encounters"
)

// Object classes, declared in the Tiled project.
const (
	ClassTransition = "Transition"
	ClassTrigger    = "Trigger"
	ClassNPC        = "Npc"
	ClassEncounter  = "Encounter"
	ClassSpawn      = "Spawn"
)

// Object kinds, the wotes.kind property.
const (
	KindEntrance   = "entrance"
	KindZone       = "zone"
	KindTileset    = "tileset"
	KindTrigger    = "trigger"
	KindGravestone = "gravestone"
	KindNPC        = "npc"
	KindTownsfolk  = "townsfolk"
	KindEncounter  = "encounter"
	KindMonster    = "monster"
)

// Object property names.
const (
	PropIndex     = "wotes.index"
	PropMap       = "wotes.map"
	PropEntry     = "wotes.entry"
	PropLevel     = "wotes.level"
	PropFlag      = "wotes.flag"
	PropTrigger   = "wotes.trigger"
	PropAction    = "wotes.action"
	PropSpeech    = "wotes.speech." // followed by the game level
	PropText      = "wotes.text"
	PropMember    = "wotes.member"
	PropFigure    = "wotes.figure"
	PropX         = "wotes.x"
	PropY         = "wotes.y"
	PropShows     = "wotes.shows"
	PropMoves     = "wotes.moves"
	PropSlot      = "wotes.slot"
	PropSpare     = "wotes.spare"
	PropMonster   = "wotes.monster"
	PropMin       = "wotes.min"
	PropMax       = "wotes.max"
	PropFlagClear = "wotes.flag_clear"
	PropFlagSet   = "wotes.flag_set"
	PropEncounter = "wotes.encounter"
)

// The vocabulary of a land map's own object layers: the trigger kinds (an event trigger is KindEvent, the
// dungeons' word), the shop index, an event's quest picture and after-text, the figures' palette line and
// the three movements a figure may have.
const (
	KindShop          = "shop"
	KindRestore       = "restore"
	KindSpeech        = "speech"
	PropShop          = "wotes.shop"
	PropPicture       = "wotes.picture" // a quest picture, 1 to rom.QuestPictures
	PropAfter         = "wotes.after"   // the text an event shows once its flag is set
	PropFigureColours = "wotes.figure_colours"
	PropFront         = "wotes.front" // drawn in front of every other sprite
	MovesStands       = "stands"
	MovesEastWest     = "east-west"
	MovesNorthSouth   = "north-south"
)

// The vocabulary of a map's spawn areas, on its encounters layer: the monsters a random encounter draws from,
// the frames between two rolls, whether a group is tried on every side, and on an extracted map the game's terrain.
const (
	KindSpawn         = "spawn"
	PropMonsters      = "wotes.monsters"
	PropPause         = "wotes.pause"
	PropAllDirections = "wotes.all_directions"
	PropTerrain       = "wotes.terrain"
)

// The castle wall's tile-set swap, left, top, right, bottom in party pixels: the city set loads inside
// wallInner, the valley set outside wallOuter.
var (
	wallInner = [4]int{0x938, 0xCF8, 0xDF8, 0x1100}
	wallOuter = [4]int{0x8B8, 0xC78, 0xE78, 0x1180}
)

// zoneEntrance maps the special-place actions that walk the party into a dungeon to the entrance each opens.
var zoneEntrance = map[int]int{8: 9, 9: 3}

// The swamp's fixed fight runs while event flag 0xFD is clear and sets flag 0xFC.
const (
	swampFightFlagClear = 0xFD
	swampFightFlagSet   = 0xFC
)

// npcNear is how far, in pixels, a figure may stand from a trigger and still be named after it.
const npcNear = 40

// hasFigures says whether a region's map draws the castle figures: only ram_region 0, the city, does.
func hasFigures(l *rom.Layout, reg rom.Region) bool {
	idx, err := l.RegionIndex(reg.Name)
	return err == nil && idx == 0
}

// triggerNames names the actions that are not plain speeches; shops take their titles instead.
var triggerNames = map[int]string{5: "Duke Barrik", 6: "Marmillian", 7: "Temple", 10: "City gate", 11: "City gate", 12: "Tree of Life", 13: "Well of Souls",
	14: "Bridge warning", 15: "Graveyard", 28: "Lizard-man curse", 29: "Swamp artifacts and map", 30: "No effect", 31: "Gold idols", 32: "Waters of Life"}

// shopActions maps the actions that open a shop to the shop's title index.
var shopActions = map[int]int{0: 0, 1: 1, 2: 2, 20: 3}

// triggerRegion is the ram_region a special-place action belongs to: the swamp, jungle and valley actions
// test for their region or lie in it; every other rectangle lies inside the castle walls.
func triggerRegion(action int) int {
	switch action {
	case 9, 12, 13, 28, 29, 30, 31:
		return 3
	case 8, 32:
		return 2
	case 14:
		return 1
	}
	return 0
}

// builder numbers objects across a map's layers.
type builder struct {
	g    *rom.Game
	next int
}

func (b *builder) object(name, class, kind string, props ...tiled.Property) tiled.Object {
	o := tiled.Object{ID: b.next, Name: name, Class: class, Visible: true, Properties: append([]tiled.Property{tiled.Str(PropKind, kind)}, props...)}
	b.next++
	return o
}

func (b *builder) point(x, y int, name, class, kind string, props ...tiled.Property) tiled.Object {
	o := b.object(name, class, kind, props...)
	o.X, o.Y, o.Point = float64(x), float64(y), true
	return o
}

// rect covers x0 to x1 and y0 to y1 inclusive.
func (b *builder) rect(x0, y0, x1, y1 int, name, class, kind string, props ...tiled.Property) tiled.Object {
	o := b.object(name, class, kind, props...)
	o.X, o.Y, o.Width, o.Height = float64(x0), float64(y0), float64(x1-x0+1), float64(y1-y0+1)
	return o
}

// ObjectLayers builds a region's transitions, triggers, npcs and encounters layers from the ROM's tables,
// with layer ids from firstLayerID and object ids from firstObjectID. figuresGID is the first GID of the
// figures tileset, which only the city's map references. It returns the layers and the next object id.
func ObjectLayers(g *rom.Game, reg rom.Region, firstLayerID, firstObjectID, figuresGID int) ([]tiled.Layer, int, error) {
	idx, err := g.Layout.RegionIndex(reg.Name)
	if err != nil {
		return nil, 0, err
	}
	b := &builder{g: g, next: firstObjectID}
	places, err := g.Places()
	if err != nil {
		return nil, 0, err
	}
	transitions, err := b.transitions(idx, places)
	if err != nil {
		return nil, 0, err
	}
	triggers, err := b.triggers(idx, places)
	if err != nil {
		return nil, 0, err
	}
	var npcs []tiled.Object
	if hasFigures(g.Layout, reg) {
		if npcs, err = b.npcs(triggers, figuresGID); err != nil {
			return nil, 0, err
		}
	}
	encounters, err := b.encounters(idx)
	if err != nil {
		return nil, 0, err
	}
	areas, err := b.spawns(idx, reg)
	if err != nil {
		return nil, 0, err
	}
	encounters = append(encounters, areas...)
	var layers []tiled.Layer
	for i, l := range []struct {
		name    string
		objects []tiled.Object
	}{{LayerTransitions, transitions}, {LayerTriggers, triggers}, {LayerNPCs, npcs}, {LayerEncounters, encounters}} {
		layers = append(layers, tiled.Layer{Type: "objectgroup", ID: firstLayerID + i, Name: l.name, Opacity: 1, Visible: true, DrawOrder: "topdown", Objects: l.objects})
	}
	return layers, b.next, nil
}

func (b *builder) transitions(idx int, places []rom.Place) ([]tiled.Object, error) {
	entrances, err := b.g.Entrances()
	if err != nil {
		return nil, err
	}
	var out []tiled.Object
	for _, e := range entrances {
		if e.Region != idx || e.Unreachable {
			continue
		}
		name, err := b.g.Layout.DungeonName(e.Map)
		if err != nil {
			return nil, fmt.Errorf("entrance %d: %v", e.Index, err)
		}
		props := []tiled.Property{tiled.Int(PropIndex, e.Index), tiled.Int(PropMap, e.Map), tiled.Int(PropEntry, e.Entry), tiled.Int(PropLevel, e.Level)}
		if e.Flag != 0 {
			props = append(props, tiled.Int(PropFlag, e.Flag))
		}
		out = append(out, b.point(e.X, e.Y, name, ClassTransition, KindEntrance, props...))
	}
	for _, p := range places {
		target, ok := zoneEntrance[p.Action]
		if !ok || triggerRegion(p.Action) != idx {
			continue
		}
		e, err := findEntrance(entrances, target)
		if err != nil {
			return nil, fmt.Errorf("special place %d: %v", p.Index, err)
		}
		name, err := b.g.Layout.DungeonName(e.Map)
		if err != nil {
			return nil, fmt.Errorf("special place %d: %v", p.Index, err)
		}
		out = append(out, b.rect(p.X0, p.Y0, p.X1, p.Y1, name, ClassTransition, KindZone,
			tiled.Int(PropIndex, e.Index), tiled.Int(PropMap, e.Map), tiled.Int(PropEntry, e.Entry), tiled.Int(PropLevel, p.Level), tiled.Int(PropTrigger, p.Index)))
	}
	if idx <= 1 {
		out = append(out,
			b.rect(wallInner[0], wallInner[1], wallInner[2]-1, wallInner[3]-1, "city tile set", ClassTransition, KindTileset, tiled.Int(PropSet, 2)),
			b.rect(wallOuter[0], wallOuter[1], wallOuter[2]-1, wallOuter[3]-1, "valley tile set", ClassTransition, KindTileset, tiled.Int(PropSet, 6)))
	}
	return out, nil
}

// findEntrance returns the entrance at index i of the tables.
func findEntrance(entrances []rom.Entrance, i int) (rom.Entrance, error) {
	for _, e := range entrances {
		if e.Index == i {
			return e, nil
		}
	}
	return rom.Entrance{}, fmt.Errorf("entrance %d is not in rom_region_exit_points", i)
}

func (b *builder) triggers(idx int, places []rom.Place) ([]tiled.Object, error) {
	var out []tiled.Object
	for _, p := range places {
		if _, zone := zoneEntrance[p.Action]; zone || triggerRegion(p.Action) != idx {
			continue
		}
		speeches := make([][]string, b.g.Layout.Places.Levels)
		props := []tiled.Property{tiled.Int(PropIndex, p.Index), tiled.Int(PropAction, p.Action), tiled.Int(PropLevel, p.Level)}
		for level := range speeches {
			lines, err := b.g.Speech(level, p.Action)
			if err != nil {
				return nil, fmt.Errorf("special place %d: %v", p.Index, err)
			}
			speeches[level] = lines
			if len(lines) != 0 {
				props = append(props, tiled.Str(fmt.Sprintf("%s%d", PropSpeech, level), strings.Join(lines, "\n")))
			}
		}
		name, err := b.triggerName(p, speeches)
		if err != nil {
			return nil, fmt.Errorf("special place %d: %v", p.Index, err)
		}
		out = append(out, b.rect(p.X0, p.Y0, p.X1, p.Y1, name, ClassTrigger, KindTrigger, props...))
	}
	if idx == 0 {
		stones, err := b.g.Gravestones()
		if err != nil {
			return nil, err
		}
		for _, s := range stones {
			prop := tiled.Int(PropMember, s.Member)
			if s.Text != nil {
				prop = tiled.Str(PropText, strings.Join(s.Text, "\n"))
			}
			out = append(out, b.point(s.X, s.Y, "Gravestone", ClassTrigger, KindGravestone, prop))
		}
	}
	return out, nil
}

// triggerName names a special place: its shop's title, its known role, the first sentence of its
// lowest-level speech, or its action number.
func (b *builder) triggerName(p rom.Place, speeches [][]string) (string, error) {
	if shop, ok := shopActions[p.Action]; ok {
		return b.g.ShopTitle(shop)
	}
	if name, ok := triggerNames[p.Action]; ok {
		return name, nil
	}
	for _, lines := range speeches {
		if len(lines) != 0 {
			return firstSentence(lines), nil
		}
	}
	return fmt.Sprintf("Action %d", p.Action), nil
}

// firstSentence joins lines with spaces and cuts after the first full stop, exclamation or question mark.
func firstSentence(lines []string) string {
	text := strings.Join(lines, " ")
	if i := strings.IndexAny(text, ".!?"); i >= 0 {
		return text[:i+1]
	}
	return text
}

func (b *builder) npcs(triggers []tiled.Object, figuresGID int) ([]tiled.Object, error) {
	records, err := b.g.CityNPCs()
	if err != nil {
		return nil, err
	}
	var out []tiled.Object
	for _, n := range records {
		figure, err := b.g.Layout.Figure(n.Tile)
		if err != nil {
			return nil, fmt.Errorf("rom_city_npcs record %d: %v", n.Index, err)
		}
		if n.Palette != rom.FigurePalette {
			return nil, fmt.Errorf("rom_city_npcs record %d: palette line %d, the figures tileset is drawn on line %d", n.Index, n.Palette, rom.FigurePalette)
		}
		name := fmt.Sprintf("Figure %d", figure)
		if near := nearestTrigger(triggers, n.X, n.Y); near != "" {
			name += " at " + near
		}
		props := []tiled.Property{tiled.Int(PropIndex, n.Index), tiled.Int(PropFigure, figure), tiled.Int(PropTile, n.Tile), tiled.Int(PropX, n.X), tiled.Int(PropY, n.Y), tiled.Str(PropShows, shows(n))}
		switch n.Routine {
		case rom.NPCPacesX:
			props = append(props, tiled.Str(PropMoves, MovesEastWest))
		case rom.NPCPacesY:
			props = append(props, tiled.Str(PropMoves, MovesNorthSouth))
		}
		if n.Front {
			props = append(props, tiled.Bool(PropFront, true))
		}
		if n.Slot != 0 {
			props = append(props, tiled.Int(PropSlot, n.Slot))
		}
		o := b.object(name, ClassNPC, KindNPC, props...)
		o.GID = uint32(figuresGID + figure)
		o.X, o.Y = float64(n.X+n.XOff), float64(n.Y+n.YOff+FigureHeight)
		o.Width, o.Height = FigureWidth, FigureHeight
		out = append(out, o)
	}
	for _, s := range b.g.TownsfolkSpots() {
		name, spare := "Townsperson spot", s.Index >= rom.TownsfolkUsed
		if spare {
			name += " (spare)"
		}
		out = append(out, b.point(s.X, s.Y, name, ClassNPC, KindTownsfolk, tiled.Int(PropIndex, s.Index), tiled.Int(PropFigure, s.Figure), tiled.Bool(PropSpare, spare)))
	}
	return out, nil
}

// shows says when the game draws a figure.
func shows(n rom.NPC) string {
	switch n.Routine {
	case rom.NPCFlagBit1:
		return "flag 0xFF8F51 bit 1"
	case rom.NPCFlagBit2:
		return "flag 0xFF8F51 bit 2"
	case rom.NPCFlagBit3:
		return "flag 0xFF8F51 bit 3"
	case rom.NPCFlagBit4:
		return "flag 0xFF8F51 bit 4"
	case rom.NPCRandom:
		return "random"
	}
	return "always"
}

// nearestTrigger names the trigger rectangle nearest to (x, y) within npcNear pixels, or "".
func nearestTrigger(triggers []tiled.Object, x, y int) string {
	best, name := npcNear+1, ""
	for _, t := range triggers {
		if t.Point {
			continue
		}
		d := max(int(t.X)-x, x-int(t.X+t.Width)+1, int(t.Y)-y, y-int(t.Y+t.Height)+1, 0)
		if d < best {
			best, name = d, t.Name
		}
	}
	return name
}

func (b *builder) encounters(idx int) ([]tiled.Object, error) {
	fights, err := b.g.FixedEncounters()
	if err != nil {
		return nil, err
	}
	var out []tiled.Object
	for _, e := range fights {
		if e.Region != idx {
			continue
		}
		monster := b.g.MonsterName(e.Monster)
		name := fmt.Sprintf("%d %s", e.Max, monster)
		if e.Min != e.Max {
			name = fmt.Sprintf("%d TO %d %s", e.Min, e.Max, monster)
		}
		props := []tiled.Property{tiled.Int(PropIndex, e.Index), tiled.Int(PropMonster, e.Monster), tiled.Int(PropMin, e.Min), tiled.Int(PropMax, e.Max), tiled.Int(PropX, e.X), tiled.Int(PropY, e.Y)}
		if idx == 3 {
			props = append(props, tiled.Int(PropFlagClear, swampFightFlagClear), tiled.Int(PropFlagSet, swampFightFlagSet))
		}
		out = append(out, b.rect(e.X-rom.EncounterReach, e.Y-rom.EncounterReach, e.X+rom.EncounterReach, e.Y+rom.EncounterReach, name, ClassEncounter, KindEncounter, props...))
		for _, pos := range e.Positions {
			out = append(out, b.point(pos[0], pos[1], monster, ClassEncounter, KindMonster, tiled.Int(PropEncounter, e.Index)))
		}
	}
	return out, nil
}
