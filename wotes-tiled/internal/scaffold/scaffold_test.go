package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/convert"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/extract"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/romtest"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/tiled"
)

func TestNew(t *testing.T) {
	dir := t.TempDir()
	r := romtest.New()
	if err := extract.Run(r, romtest.Regions, dir); err != nil {
		t.Fatal(err)
	}
	if err := New(dir, "probe", romtest.Region); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"probe.tmj", "probe-metatiles.tmj", "gen/probe-metatiles.tsj", "gen/probe-metatiles.png"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("%s not written: %v", name, err)
		}
	}
	m, err := tiled.ReadMap(filepath.Join(dir, "probe.tmj"))
	if err != nil {
		t.Fatal(err)
	}
	// The synthetic valley has 3 metatiles: flags start at 4 (two tiles), the project tileset at 6, last.
	if len(m.Tilesets) != 3 || m.Tilesets[1].FirstGID != 4 || m.Tilesets[2].FirstGID != 6 || m.Tilesets[2].Source != "gen/probe-metatiles.tsj" || tiled.StringProp(m.Properties, extract.PropBase) != "valley" {
		t.Fatalf("tilesets %+v", m.Tilesets)
	}
	if m.Tilesets[0].FirstGID != 1 || m.Tilesets[0].Source != "rom/valley-metatiles.tsj" {
		t.Fatalf("the base tileset is %+v", m.Tilesets[0])
	}
	var layers []string
	for _, l := range m.Layers {
		layers = append(layers, l.Name)
	}
	if strings.Join(layers, " ") != "ground front solid places" {
		t.Fatalf("layers %v", layers)
	}
	ts, err := tiled.ReadTileset(filepath.Join(dir, "gen/probe-metatiles.tsj"))
	if err != nil {
		t.Fatal(err)
	}
	if kind, defs := tiled.StringProp(ts.Properties, extract.PropKind), tiled.StringProp(ts.Properties, extract.PropDefinitions); kind != "project-metatiles" || defs != "../probe-metatiles.tmj" {
		t.Fatalf("project tileset kind %q, definitions %q", kind, defs)
	}
	// A fresh map has an empty ground, which convert must refuse at its first cell; the tilesets-only pass succeeds.
	_, err = convert.Convert(convert.Options{ROM: r, Regions: romtest.Regions, MapPath: filepath.Join(dir, "probe.tmj")})
	if err == nil || !strings.Contains(err.Error(), "(x 0, y 0): empty ground cell") {
		t.Fatalf("a fresh map must fail on its empty ground, got %v", err)
	}
	if _, err := convert.Convert(convert.Options{ROM: r, Regions: romtest.Regions, MapPath: filepath.Join(dir, "probe.tmj"), TilesetsOnly: true}); err != nil {
		t.Fatalf("tilesets-only on a fresh map: %v", err)
	}
	if err := New(dir, "probe", romtest.Region); err == nil || !strings.Contains(err.Error(), "exists") {
		t.Fatalf("a second New must refuse to overwrite, got %v", err)
	}
	if err := New(dir, "my-land", romtest.Region); err == nil || !strings.Contains(err.Error(), "letters, digits and underscores") {
		t.Fatalf("a name that is not a symbol must be refused, got %v", err)
	}
}

func TestNewRemovesAPartialScaffold(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores the read-only folder")
	}
	dir := t.TempDir()
	gen := filepath.Join(dir, "gen")
	if err := os.Mkdir(gen, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(gen, 0o755) })
	if err := New(dir, "probe", romtest.Region); err == nil {
		t.Fatal("New must fail when gen is read-only")
	}
	if _, err := os.Stat(filepath.Join(dir, "probe-metatiles.tmj")); !os.IsNotExist(err) {
		t.Fatalf("the definitions map written before the failure must be removed, stat: %v", err)
	}
	if err := os.Chmod(gen, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := New(dir, "probe", romtest.Region); err != nil {
		t.Fatalf("New after a failed one: %v", err)
	}
}
