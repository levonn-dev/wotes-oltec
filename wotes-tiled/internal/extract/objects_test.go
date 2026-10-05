package extract

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/romtest"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/tiled"
)

func intProp(t *testing.T, o tiled.Object, name string) int {
	v, ok := tiled.IntProp(o.Properties, name)
	if !ok {
		t.Fatalf("object %d %q has no %s", o.ID, o.Name, name)
	}
	return v
}

func hasProp(o tiled.Object, name string) bool {
	for _, p := range o.Properties {
		if p.Name == name {
			return true
		}
	}
	return false
}

func byName(t *testing.T, l tiled.Layer, name string) tiled.Object {
	for _, o := range l.Objects {
		if o.Name == name {
			return o
		}
	}
	t.Fatalf("layer %s has no object %q: %+v", l.Name, name, l.Objects)
	return tiled.Object{}
}

func checkLayers(t *testing.T, layers []tiled.Layer, firstLayerID, firstObjectID, next int) {
	names := []string{LayerTransitions, LayerTriggers, LayerNPCs, LayerEncounters}
	if len(layers) != 4 {
		t.Fatalf("%d layers", len(layers))
	}
	id := firstObjectID
	for i, l := range layers {
		if l.Type != "objectgroup" || l.Name != names[i] || l.ID != firstLayerID+i || l.DrawOrder != "topdown" || !l.Visible || l.Opacity != 1 {
			t.Fatalf("layer %d: %+v", i, l)
		}
		for _, o := range l.Objects {
			if o.ID != id || !o.Visible || tiled.StringProp(o.Properties, PropKind) == "" {
				t.Fatalf("object %+v: want id %d, visible, a kind", o, id)
			}
			id++
		}
	}
	if id != next {
		t.Fatalf("next object id %d, want %d", next, id)
	}
}

func TestObjectLayersCity(t *testing.T) {
	g := romtest.Game()
	layers, next, err := ObjectLayers(g, romtest.City, 5, 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	checkLayers(t, layers, 5, 1, next)
	transitions, triggers, npcs, encounters := layers[0], layers[1], layers[2], layers[3]
	// No synthetic entrance is the city's; the wall swap is.
	if len(transitions.Objects) != 2 {
		t.Fatalf("transitions: %+v", transitions.Objects)
	}
	inner := byName(t, transitions, "city tile set")
	if inner.Class != ClassTransition || tiled.StringProp(inner.Properties, PropKind) != KindTileset || inner.X != 0x938 || inner.Y != 0xCF8 || inner.Width != 0xDF8-0x938 || inner.Height != 0x1100-0xCF8 || intProp(t, inner, PropSet) != 2 {
		t.Fatalf("inner wall: %+v", inner)
	}
	outer := byName(t, transitions, "valley tile set")
	if outer.X != 0x8B8 || outer.Width != 0xE78-0x8B8 || intProp(t, outer, PropSet) != 6 {
		t.Fatalf("outer wall: %+v", outer)
	}
	// Places 0, 1 and 4 are the city's, plus two gravestones.
	if len(triggers.Objects) != 5 {
		t.Fatalf("triggers: %+v", triggers.Objects)
	}
	shop := byName(t, triggers, "ARMOR SHOP")
	if shop.Class != ClassTrigger || tiled.StringProp(shop.Properties, PropKind) != KindTrigger || shop.X != 0xB20 || shop.Y != 0x1000 || shop.Width != 0x49 || shop.Height != 0x49 ||
		intProp(t, shop, PropIndex) != 0 || intProp(t, shop, PropAction) != 0 || intProp(t, shop, PropLevel) != 3 || hasProp(shop, PropSpeech+"0") {
		t.Fatalf("shop: %+v", shop)
	}
	ferry := byName(t, triggers, "THE FERRYMAN SAYS THE RIVER RUNS BACKWARD AT NIGHT.")
	if ferry.Width != 0x31 || ferry.Height != 1 || tiled.StringProp(ferry.Properties, PropSpeech+"0") != "THE FERRYMAN SAYS THE\nRIVER RUNS BACKWARD AT\nNIGHT." ||
		tiled.StringProp(ferry.Properties, PropSpeech+"1") != "THE BRIDGE TOLL WENT UP\nAGAIN." || hasProp(ferry, PropSpeech+"2") {
		t.Fatalf("ferry: %+v", ferry)
	}
	geese := byName(t, triggers, "MIND THE GEESE BY THE WELL")
	if hasProp(geese, PropSpeech+"0") || tiled.StringProp(geese.Properties, PropSpeech+"1") != "MIND THE GEESE BY THE WELL" || intProp(t, geese, PropAction) != 21 {
		t.Fatalf("geese: %+v", geese)
	}
	var stones []tiled.Object
	for _, o := range triggers.Objects {
		if o.Name == "Gravestone" {
			stones = append(stones, o)
		}
	}
	if len(stones) != 2 || !stones[0].Point || stones[0].X != 0xD00 || stones[0].Y != 0xFB8 || intProp(t, stones[0], PropMember) != -1 || hasProp(stones[0], PropText) ||
		tiled.StringProp(stones[1].Properties, PropText) != "GONE TO SEEK A DRIER CELLAR." || tiled.StringProp(stones[1].Properties, PropKind) != KindGravestone {
		t.Fatalf("stones: %+v", stones)
	}
	// Five figures and ten spots.
	if len(npcs.Objects) != 15 {
		t.Fatalf("npcs: %+v", npcs.Objects)
	}
	f0 := byName(t, npcs, "Figure 0 at ARMOR SHOP")
	if f0.Class != ClassNPC || f0.GID != 10 || f0.X != 0xB40-12 || f0.Y != 0x1010-8+FigureHeight || f0.Width != FigureWidth || f0.Height != FigureHeight ||
		intProp(t, f0, PropFigure) != 0 || intProp(t, f0, PropTile) != 0x3AF || intProp(t, f0, PropX) != 0xB40 || intProp(t, f0, PropY) != 0x1010 ||
		tiled.StringProp(f0.Properties, PropShows) != "always" || hasProp(f0, PropMoves) || hasProp(f0, PropSlot) || hasProp(f0, PropFront) {
		t.Fatalf("figure 0: %+v", f0)
	}
	f3 := byName(t, npcs, "Figure 3")
	if f3.GID != 13 || tiled.StringProp(f3.Properties, PropMoves) != "east-west" || intProp(t, f3, PropSlot) != 1 || intProp(t, f3, PropIndex) != 1 {
		t.Fatalf("figure 3: %+v", f3)
	}
	f12 := byName(t, npcs, "Figure 12")
	if f12.X != 0x200 || f12.Y != 0x100+FigureHeight || tiled.StringProp(f12.Properties, PropShows) != "flag 0xFF8F51 bit 2" {
		t.Fatalf("figure 12: %+v", f12)
	}
	f1 := byName(t, npcs, "Figure 1")
	if f1.GID != 11 || tiled.StringProp(f1.Properties, PropShows) != "random" || intProp(t, f1, PropSlot) != 4 || !tiled.BoolProp(f1.Properties, PropFront) {
		t.Fatalf("figure 1: %+v", f1)
	}
	if f5 := byName(t, npcs, "Figure 5"); tiled.StringProp(f5.Properties, PropMoves) != "north-south" {
		t.Fatalf("figure 5: %+v", f5)
	}
	var spots, spare int
	for _, o := range npcs.Objects {
		if strings.HasPrefix(o.Name, "Townsperson spot") {
			spots++
			if tiled.BoolProp(o.Properties, PropSpare) {
				spare++
				if o.Name != "Townsperson spot (spare)" || intProp(t, o, PropIndex) < 8 {
					t.Fatalf("spare spot: %+v", o)
				}
			}
			if !o.Point || tiled.StringProp(o.Properties, PropKind) != KindTownsfolk || o.Y != 0xF38 {
				t.Fatalf("spot: %+v", o)
			}
		}
	}
	if spots != 10 || spare != 2 {
		t.Fatalf("%d spots, %d spare", spots, spare)
	}
	if len(encounters.Objects) != 1 || tiled.StringProp(encounters.Objects[0].Properties, PropKind) != KindSpawn {
		t.Fatalf("the city has no fixed fight, only its spawn area: %+v", encounters.Objects)
	}
}

func TestObjectLayersValley(t *testing.T) {
	g := romtest.Game()
	layers, next, err := ObjectLayers(g, romtest.Region, 5, 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	checkLayers(t, layers, 5, 1, next)
	transitions, triggers, npcs, encounters := layers[0], layers[1], layers[2], layers[3]
	if len(transitions.Objects) != 3 {
		t.Fatalf("transitions: %+v", transitions.Objects)
	}
	gate := byName(t, transitions, "Gateway Passage")
	if !gate.Point || gate.X != 0x100 || gate.Y != 0x200 || tiled.StringProp(gate.Properties, PropKind) != KindEntrance || gate.Class != ClassTransition ||
		intProp(t, gate, PropIndex) != 0 || intProp(t, gate, PropMap) != 2 || intProp(t, gate, PropEntry) != 0 || intProp(t, gate, PropLevel) != 0 || intProp(t, gate, PropFlag) != 0xF7 {
		t.Fatalf("entrance 0: %+v", gate)
	}
	for _, o := range transitions.Objects {
		if v, _ := tiled.IntProp(o.Properties, PropMap); v == 0x0C {
			t.Fatalf("the shadowed entrance was emitted: %+v", o)
		}
	}
	byName(t, transitions, "city tile set")
	if len(triggers.Objects) != 0 || len(npcs.Objects) != 0 {
		t.Fatalf("valley triggers %+v, npcs %+v", triggers.Objects, npcs.Objects)
	}
	// The fight and its three monsters come first, then the three spawn areas.
	if len(encounters.Objects) != 7 || tiled.StringProp(encounters.Objects[0].Properties, PropKind) != KindEncounter || tiled.StringProp(encounters.Objects[4].Properties, PropKind) != KindSpawn {
		t.Fatalf("encounters: %+v", encounters.Objects)
	}
	fight := byName(t, encounters, "2 TO 3 HAIRY BEASTMAN")
	if fight.Class != ClassEncounter || tiled.StringProp(fight.Properties, PropKind) != KindEncounter || fight.X != 0x8FC-96 || fight.Y != 0x900-96 || fight.Width != 193 || fight.Height != 193 ||
		intProp(t, fight, PropMonster) != 0x74 || intProp(t, fight, PropMin) != 2 || intProp(t, fight, PropMax) != 3 || intProp(t, fight, PropX) != 0x8FC || intProp(t, fight, PropY) != 0x900 || hasProp(fight, PropFlagClear) {
		t.Fatalf("fight: %+v", fight)
	}
	monster := byName(t, encounters, "HAIRY BEASTMAN")
	if !monster.Point || monster.X != 0x908 || monster.Y != 0x990 || tiled.StringProp(monster.Properties, PropKind) != KindMonster || intProp(t, monster, PropEncounter) != 0 {
		t.Fatalf("monster: %+v", monster)
	}
}

func TestObjectLayersSwamp(t *testing.T) {
	g := romtest.Game()
	layers, next, err := ObjectLayers(g, romtest.Other, 5, 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	checkLayers(t, layers, 5, 1, next)
	transitions, triggers, npcs, encounters := layers[0], layers[1], layers[2], layers[3]
	if len(transitions.Objects) != 2 {
		t.Fatalf("transitions: %+v", transitions.Objects)
	}
	var zone tiled.Object
	for _, o := range transitions.Objects {
		if tiled.StringProp(o.Properties, PropKind) == KindZone {
			zone = o
		}
	}
	if zone.Name != "Gateway Passage" || zone.X != 0x4A0 || zone.Y != 0xAA0 || zone.Width != 0x109 || zone.Height != 0x141 ||
		intProp(t, zone, PropIndex) != 3 || intProp(t, zone, PropMap) != 2 || intProp(t, zone, PropEntry) != 1 || intProp(t, zone, PropLevel) != 4 || intProp(t, zone, PropTrigger) != 3 {
		t.Fatalf("zone: %+v", zone)
	}
	entrance := byName(t, transitions, "Gateway Passage")
	if !entrance.Point {
		t.Fatalf("the entrance point is missing: %+v", transitions.Objects)
	}
	tree := byName(t, triggers, "Tree of Life")
	if len(triggers.Objects) != 1 || intProp(t, tree, PropAction) != 12 || intProp(t, tree, PropLevel) != 4 || tree.X != 0x178 || tree.Width != 0x49 {
		t.Fatalf("triggers: %+v", triggers.Objects)
	}
	if len(npcs.Objects) != 0 {
		t.Fatalf("swamp npcs: %+v", npcs.Objects)
	}
	fight := byName(t, encounters, "2 LIZARD MAN")
	if len(encounters.Objects) != 4 || intProp(t, fight, PropFlagClear) != 0xFD || intProp(t, fight, PropFlagSet) != 0xFC {
		t.Fatalf("encounters: %+v", encounters.Objects)
	}
}

func TestObjectLayersRejectsOtherRegions(t *testing.T) {
	reg := romtest.Region
	reg.Name = "desert"
	if _, _, err := ObjectLayers(romtest.Game(), reg, 5, 1, 0); err == nil {
		t.Fatal("a region outside the four did not fail")
	}
}

func TestObjectLayersErrors(t *testing.T) {
	g := romtest.Game()
	g.ROM[int(rom.Clean.Entrances.Points)+5] = 0x26
	if _, _, err := ObjectLayers(g, romtest.Region, 5, 1, 0); err == nil || !strings.Contains(err.Error(), "entrance 0:") || !strings.Contains(err.Error(), "map 0x26") {
		t.Fatalf("an entrance to an unnamed map: %v", err)
	}
	g = romtest.Game()
	g.ROM[int(rom.Clean.Entrances.Points)+3*8+5] = rom.FillerMap
	if _, _, err := ObjectLayers(g, romtest.Other, 5, 1, 0); err == nil || !strings.Contains(err.Error(), "special place 3:") ||
		!strings.Contains(err.Error(), "entrance 3 is not in rom_region_exit_points") {
		t.Fatalf("a zone into a filler entrance: %v", err)
	}
	g = romtest.Game()
	g.ROM[int(rom.Clean.Figures.Records)+8], g.ROM[int(rom.Clean.Figures.Records)+9], g.ROM[int(rom.Clean.Figures.Records)+10], g.ROM[int(rom.Clean.Figures.Records)+11] = 0x00, 0x00, 0x12, 0x34
	if _, _, err := ObjectLayers(g, romtest.City, 5, 1, 10); err == nil || !strings.Contains(err.Error(), "rom_city_npcs record 0") {
		t.Fatalf("a figure with an unknown routine: %v", err)
	}
	l := *romtest.Layout()
	l.Places.ShopTitles = rom.Addr(len(g.ROM) - 2) // every shop's title long runs past the end
	g = &rom.Game{ROM: romtest.New(), Layout: &l}
	if _, _, err := ObjectLayers(g, romtest.City, 5, 1, 10); err == nil || !strings.Contains(err.Error(), "special place") ||
		!strings.Contains(err.Error(), "rom_shop_titles entry") || !strings.Contains(err.Error(), "runs past the ROM's end") {
		t.Fatalf("shop titles past the end: %v", err)
	}
}

func TestRegionMapsCarryObjects(t *testing.T) {
	dir := t.TempDir()
	if err := Run(romtest.Game(), dir); err != nil {
		t.Fatal(err)
	}
	city, err := tiled.ReadMap(filepath.Join(dir, "rom", "city.tmj"))
	if err != nil {
		t.Fatal(err)
	}
	if len(city.Layers) != 8 || city.Layers[4].Name != LayerTransitions || city.Layers[7].Name != LayerEncounters || city.NextLayerID != 9 {
		t.Fatalf("city layers: %+v, next layer %d", city.Layers, city.NextLayerID)
	}
	if len(city.Tilesets) != 4 || city.Tilesets[2].Source != FiguresName+".tsj" || city.Tilesets[2].FirstGID != 1+3+FlagCount ||
		city.Tilesets[3].Source != SolidName+".tsj" || city.Tilesets[3].FirstGID != 1+3+FlagCount+rom.FigureCount {
		t.Fatalf("city tilesets: %+v", city.Tilesets)
	}
	last := city.Layers[7].Objects[len(city.Layers[7].Objects)-1]
	if city.NextObjectID != last.ID+1 || len(city.Layers[3].Objects) != 0 {
		t.Fatalf("next object id %d after %d; places %+v", city.NextObjectID, last.ID, city.Layers[3].Objects)
	}
	if f0 := byName(t, city.Layers[6], "Figure 0 at ARMOR SHOP"); f0.GID != uint32(1+3+FlagCount) {
		t.Fatalf("figure 0 GID %d", f0.GID)
	}
	valley, err := tiled.ReadMap(filepath.Join(dir, "rom", "valley.tmj"))
	if err != nil {
		t.Fatal(err)
	}
	if len(valley.Tilesets) != 3 || valley.Tilesets[2].Source != SolidName+".tsj" || len(valley.Layers) != 8 {
		t.Fatalf("valley: %d tilesets, %d layers", len(valley.Tilesets), len(valley.Layers))
	}
}

func TestNearestTrigger(t *testing.T) {
	a := tiled.Object{Name: "A", X: 100, Y: 100, Width: 50, Height: 20}
	b := tiled.Object{Name: "B", X: 300, Y: 100, Width: 10, Height: 10}
	c := tiled.Object{Name: "C", X: 170, Y: 100, Width: 10, Height: 10}
	p := tiled.Object{Name: "P", Point: true, X: 189, Y: 110}
	for _, tc := range []struct {
		triggers []tiled.Object
		x, y     int
		want     string
	}{
		{[]tiled.Object{a}, 189, 110, "A"}, // A's right edge pixel is 149
		{[]tiled.Object{a}, 190, 110, ""},
		{[]tiled.Object{a}, 120, 110, "A"},
		{[]tiled.Object{a, b}, 260, 105, "B"}, // 111 from A, 40 from B
		{[]tiled.Object{a, b}, 285, 105, "B"},
		{[]tiled.Object{a, c}, 160, 105, "C"}, // 11 from A, 10 from C
		{[]tiled.Object{a, c}, 158, 105, "A"}, // 9 from A, 12 from C
		{[]tiled.Object{p}, 189, 110, ""},
		{[]tiled.Object{p, a}, 189, 110, "A"},
	} {
		if got := nearestTrigger(tc.triggers, tc.x, tc.y); got != tc.want {
			t.Errorf("(%d, %d) near %d triggers: %q, want %q", tc.x, tc.y, len(tc.triggers), got, tc.want)
		}
	}
}

func TestObjectLayersRefusesOtherPalette(t *testing.T) {
	g := romtest.Game()
	g.ROM[int(rom.Clean.Figures.Records)+6], g.ROM[int(rom.Clean.Figures.Records)+7] = 0x40, 0x00
	_, _, err := ObjectLayers(g, romtest.City, 5, 1, 10)
	if err == nil || !strings.Contains(err.Error(), "record 0") || !strings.Contains(err.Error(), "palette line 2") {
		t.Fatalf("a figure on another palette line: %v", err)
	}
}
