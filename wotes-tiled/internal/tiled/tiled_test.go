package tiled

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// must fails the test immediately if err is not nil.
func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestGID(t *testing.T) {
	g := MakeGID(42, true, false)
	if g != 0x80000000|42 {
		t.Fatalf("MakeGID = %#x", g)
	}
	id, h, v, err := SplitGID(g)
	if err != nil || id != 42 || !h || v {
		t.Fatalf("SplitGID = %d %v %v %v", id, h, v, err)
	}
	g = MakeGID(42, false, true)
	if g != 0x40000000|42 {
		t.Fatalf("MakeGID = %#x", g)
	}
	id, h, v, err = SplitGID(g)
	if err != nil || id != 42 || h || !v {
		t.Fatalf("SplitGID = %d %v %v %v", id, h, v, err)
	}
	g = MakeGID(42, true, true)
	if g != 0x80000000|0x40000000|42 {
		t.Fatalf("MakeGID = %#x", g)
	}
	id, h, v, err = SplitGID(g)
	if err != nil || id != 42 || !h || !v {
		t.Fatalf("SplitGID = %d %v %v %v", id, h, v, err)
	}
	if _, _, _, err := SplitGID(0x20000000 | 7); err == nil || !strings.Contains(err.Error(), "diagonal") {
		t.Fatalf("a diagonal flip must be refused, got %v", err)
	}
	if _, _, _, err := SplitGID(0x10000000 | 7); err == nil {
		t.Fatal("a hexagonal rotation must be refused")
	}
}

func TestLayerDataCSVOnly(t *testing.T) {
	const csvMsg = `layer "ground": tile data must use the CSV encoding (Map > Map Properties > Tile Layer Format in Tiled)`
	l := Layer{Type: "tilelayer", Name: "ground", Width: 2, Height: 2, Encoding: "base64", Data: json.RawMessage(`"AQAAAAIAAAADAAAABAAAAA=="`)}
	if _, err := l.GIDs(); err == nil || err.Error() != csvMsg {
		t.Fatalf("base64 data must ask for CSV, got %v", err)
	}
	l = Layer{Type: "tilelayer", Name: "ground", Width: 2, Height: 2, Data: json.RawMessage(`"gAAA"`)}
	if _, err := l.GIDs(); err == nil || err.Error() != csvMsg {
		t.Fatalf("a JSON string with no encoding key must ask for CSV, got %v", err)
	}
	l = Layer{Type: "tilelayer", Name: "ground", Width: 2, Height: 2}
	l.SetData([]uint32{1, 2, 3, 4})
	g, err := l.GIDs()
	if err != nil || len(g) != 4 || g[3] != 4 {
		t.Fatalf("GIDs = %v, %v", g, err)
	}
	l.SetData([]uint32{1, 2, 3})
	if _, err := l.GIDs(); err == nil {
		t.Fatal("a short data array must fail")
	}
}

func TestMapRoundTripAndRowFormatting(t *testing.T) {
	dir := t.TempDir()
	m := &Map{Type: "map", Version: FormatVersion, TiledVersion: TiledVersion, Orientation: "orthogonal", RenderOrder: "right-down",
		Width: 3, Height: 2, TileWidth: 24, TileHeight: 24, NextLayerID: 3, NextObjectID: 4,
		Properties: []Property{Str("wotes.base", "valley")},
		Tilesets:   []TilesetRef{{FirstGID: 1, Source: "rom/valley-metatiles.tsj"}},
		Layers: []Layer{
			{Type: "tilelayer", ID: 1, Name: "ground", Width: 3, Height: 2, Opacity: 1, Visible: true},
			{Type: "objectgroup", ID: 2, Name: "places", Opacity: 1, Visible: true, DrawOrder: "topdown",
				Objects: []Object{
					{ID: 1, Name: "cave", X: 100, Y: 200, Point: true, Visible: true},
					{ID: 2, Name: "pond", X: 10, Y: 20, Width: 30, Height: 40, Ellipse: true, Visible: true},
					{ID: 3, Name: "trail", X: 0, Y: 0, Visible: true, Polygon: []Point{{X: 0, Y: 0}, {X: 10, Y: 10}}},
				}},
		}}
	m.Layers[0].SetData([]uint32{1, 2, 3, 4, 5, 6})
	path := filepath.Join(dir, "m.tmj")
	must(t, WriteMap(path, m))
	text, err := os.ReadFile(path)
	must(t, err)
	if !strings.Contains(string(text), "\n   1,2,3,\n   4,5,6\n") {
		t.Fatalf("rows must be one per line:\n%s", text)
	}
	back, err := ReadMap(path)
	if err != nil {
		t.Fatal(err)
	}
	g, err := back.Layers[0].GIDs()
	if err != nil || g[5] != 6 || StringProp(back.Properties, "wotes.base") != "valley" || !back.Layers[1].Objects[0].Point {
		t.Fatalf("read back: %v %v %+v", g, err, back.Layers[1].Objects)
	}
	pond := back.Layers[1].Objects[1]
	if !pond.Ellipse || pond.Width != 30 || pond.Height != 40 {
		t.Fatalf("ellipse object did not survive the round trip: %+v", pond)
	}
	trail := back.Layers[1].Objects[2]
	if len(trail.Polygon) != 2 || trail.Polygon[1].X != 10 || trail.Polygon[1].Y != 10 {
		t.Fatalf("polygon object did not survive the round trip: %+v", trail)
	}
	if g2, _ := m.Layers[0].GIDs(); len(g2) != 6 {
		t.Fatal("WriteMap must leave the map's data intact")
	}
	bad := strings.Replace(string(text), `"version": "1.10"`, `"version": "1.11"`, 1)
	must(t, os.WriteFile(path, []byte(bad), 0o644))
	if _, err := ReadMap(path); err == nil || !strings.Contains(err.Error(), "1.11") {
		t.Fatalf("a newer format version must be refused by name, got %v", err)
	}
}

func TestWriteMapFailsWithoutCorruptingEarlierLayers(t *testing.T) {
	dir := t.TempDir()
	m := &Map{Type: "map", Version: FormatVersion, TiledVersion: TiledVersion, Orientation: "orthogonal", RenderOrder: "right-down",
		Width: 2, Height: 1, TileWidth: 8, TileHeight: 8,
		Tilesets: []TilesetRef{{FirstGID: 1, Source: "x.tsj"}},
		Layers: []Layer{
			{Type: "tilelayer", ID: 1, Name: "ground", Width: 2, Height: 1, Opacity: 1, Visible: true},
			{Type: "tilelayer", ID: 2, Name: "front", Width: 2, Height: 1, Opacity: 1, Visible: true,
				Encoding: "base64", Data: json.RawMessage(`"AQAAAAIAAAA="`)},
		}}
	m.Layers[0].SetData([]uint32{1, 2})
	before := string(m.Layers[0].Data)
	path := filepath.Join(dir, "m.tmj")
	err := WriteMap(path, m)
	if err == nil {
		t.Fatal("expected an error from the second layer's base64 data")
	}
	if !strings.Contains(err.Error(), path) || !strings.Contains(err.Error(), "front") {
		t.Fatalf("error must name the file and the failing layer, got %v", err)
	}
	if string(m.Layers[0].Data) != before {
		t.Fatalf("WriteMap must leave an earlier layer's data unchanged on error, got %s, want %s", m.Layers[0].Data, before)
	}
}

func TestTilesetRoundTrip(t *testing.T) {
	dir := t.TempDir()
	ts := &Tileset{Type: "tileset", Version: FormatVersion, TiledVersion: TiledVersion, Name: "x", TileWidth: 8, TileHeight: 8,
		TileCount: 2, Columns: 2, Image: "x.png", ImageWidth: 16, ImageHeight: 8,
		Properties: []Property{Str("wotes.kind", "tiles"), Int("wotes.palette", 3), Bool("wotes.test", true)},
		Tiles:      []TileDef{{ID: 1, Properties: []Property{Bool("wotes.solid", true)}}}}
	path := filepath.Join(dir, "x.tsj")
	if err := WriteTileset(path, ts); err != nil {
		t.Fatal(err)
	}
	back, err := ReadTileset(path)
	if err != nil {
		t.Fatal(err)
	}
	pal, ok := IntProp(back.Properties, "wotes.palette")
	if !ok || pal != 3 || !BoolProp(back.Properties, "wotes.test") || StringProp(back.Properties, "wotes.kind") != "tiles" || !BoolProp(back.Tiles[0].Properties, "wotes.solid") {
		t.Fatalf("read back: %+v", back)
	}
	if _, err := ReadTileset(filepath.Join(dir, "missing.tsj")); err == nil || !strings.Contains(err.Error(), "missing.tsj") {
		t.Fatalf("a missing file must be named, got %v", err)
	}
}

func TestIntPropRejectsNonIntegerFloats(t *testing.T) {
	frac := []Property{{Name: "wotes.frac", Type: "float", Value: 3.7}}
	if _, ok := IntProp(frac, "wotes.frac"); ok {
		t.Fatal("IntProp must not truncate a non-integer float")
	}
	whole := []Property{{Name: "wotes.whole", Type: "float", Value: 4.0}}
	if v, ok := IntProp(whole, "wotes.whole"); !ok || v != 4 {
		t.Fatalf("IntProp = %d, %v", v, ok)
	}
}
