package main

import (
	"bytes"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/extract"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/romtest"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/tiled"
)

// stdout runs f with os.Stdout captured.
func stdout(t *testing.T, f func() error) ([]byte, error) {
	t.Helper()
	old := os.Stdout
	rp, wp, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	// Draining while f runs keeps output above the pipe's buffer from blocking f.
	done := make(chan []byte)
	go func() {
		out, _ := io.ReadAll(rp)
		rp.Close()
		done <- out
	}()
	os.Stdout = wp
	ferr := f()
	wp.Close()
	os.Stdout = old
	return <-done, ferr
}

func TestLayoutCommand(t *testing.T) {
	out, err := stdout(t, func() error { return runLayout(nil) })
	if err != nil {
		t.Fatal(err)
	}
	l, err := rom.ParseLayout(bytes.NewReader(out))
	if err != nil || !reflect.DeepEqual(l, &rom.Clean) {
		t.Fatalf("layout prints %v: %s", err, out)
	}
	if err := runLayout([]string{"extra"}); err == nil {
		t.Fatal("an argument did not fail")
	}
}

func TestExtractWithLayout(t *testing.T) {
	dir := t.TempDir()
	romPath := filepath.Join(dir, "clean.md")
	if err := os.WriteFile(romPath, romtest.New(), 0o644); err != nil {
		t.Fatal(err)
	}
	layoutPath := filepath.Join(dir, "layout.json")
	b, _ := romtest.Layout().JSON()
	if err := os.WriteFile(layoutPath, b, 0o644); err != nil {
		t.Fatal(err)
	}
	maps := filepath.Join(dir, "maps")
	if _, err := stdout(t, func() error { return runExtract([]string{"--rom", romPath, "--out", maps, "--layout", layoutPath}) }); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"city.tmj", "valley.tmj", "jungle.tmj", "swamp.tmj"} {
		if _, err := os.Stat(filepath.Join(maps, "rom", name)); err != nil {
			t.Errorf("%s not written: %v", name, err)
		}
	}
	renamed := filepath.Join(dir, "renamed")
	if err := os.WriteFile(layoutPath, bytes.Replace(b, []byte(`"Gateway Passage"`), []byte(`"Renamed Passage"`), 1), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := stdout(t, func() error { return runExtract([]string{"--rom", romPath, "--out", renamed, "--layout", layoutPath}) }); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(renamed, "rom", "dungeons", "02-renamed-passage.tmj")); err != nil {
		t.Errorf("the renamed level was not written: %v", err)
	}
	if _, err := os.Stat(filepath.Join(renamed, "rom", "dungeons", "02-gateway-passage.tmj")); err == nil {
		t.Error("the layout's old name was written")
	}
	if err := os.WriteFile(layoutPath, []byte(`{"size": "0x100000", "bogus": 1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := stdout(t, func() error { return runExtract([]string{"--rom", romPath, "--out", maps, "--layout", layoutPath}) }); err == nil || !strings.Contains(err.Error(), "bogus") {
		t.Fatalf("a broken layout: %v", err)
	}
}

func TestNewDungeonCommand(t *testing.T) {
	dir := t.TempDir()
	out, err := stdout(t, func() error {
		return runNew([]string{"caves", "--dungeon", "--size", "4x3", "--look", "cave-06", "--dir", dir})
	})
	if err != nil || !strings.Contains(string(out), filepath.Join(dir, "dungeons", "caves.tmj")+" written") {
		t.Fatalf("new --dungeon: %q, %v", out, err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "dungeons", "caves.tmj"))
	if err != nil || !bytes.Contains(b, []byte(`"value": "cave-06"`)) || !bytes.Contains(b, []byte(`"width": 4,`)) {
		t.Fatalf("the map lacks its look or its width: %v", err)
	}
	for _, size := range []string{"", "4", "4x", "4x3x2", "4 x 3", "0x4x3"} {
		err := runNew([]string{"other", "--dungeon", "--size", size, "--look", "cave", "--dir", dir})
		if err == nil || !strings.Contains(err.Error(), "is not WIDTHxHEIGHT, such as 40x34") {
			t.Errorf("size %q: %v", size, err)
		}
	}
	if err := runNew([]string{"other"}); err == nil || !strings.Contains(err.Error(), "new NAME --dungeon --size WxH --look LOOK") {
		t.Errorf("no flags: %v", err)
	}
	// A flag of the other form is refused, not passed over.
	err = runNew([]string{"other", "--dungeon", "--size", "4x3", "--look", "cave", "--base", "valley", "--dir", dir})
	if err == nil || !strings.Contains(err.Error(), "--base and --layout are for a land map") {
		t.Errorf("--base on a dungeon map: %v", err)
	}
	err = runNew([]string{"other", "--base", "valley", "--size", "4x3", "--dir", dir})
	if err == nil || !strings.Contains(err.Error(), "--size and --look are for a dungeon map") {
		t.Errorf("--size on a land map: %v", err)
	}
}

func TestRenderArea(t *testing.T) {
	dir := t.TempDir()
	romPath := filepath.Join(dir, "clean.md")
	if err := os.WriteFile(romPath, romtest.New(), 0o644); err != nil {
		t.Fatal(err)
	}
	layoutPath := filepath.Join(dir, "layout.json")
	b, _ := romtest.Layout().JSON()
	if err := os.WriteFile(layoutPath, b, 0o644); err != nil {
		t.Fatal(err)
	}
	maps := filepath.Join(dir, "maps")
	if _, err := stdout(t, func() error { return runExtract([]string{"--rom", romPath, "--out", maps, "--layout", layoutPath}) }); err != nil {
		t.Fatal(err)
	}
	if _, err := stdout(t, func() error {
		return runNew([]string{"land", "--base", "valley", "--dir", maps, "--layout", layoutPath})
	}); err != nil {
		t.Fatal(err)
	}
	mapPath := filepath.Join(maps, "land.tmj")
	// A fresh scaffold has an empty ground layer, which convert refuses: paint it with the base's metatile 1.
	// The map names its area map before the file exists, as it does before the first render --area.
	m, err := tiled.ReadMap(mapPath)
	if err != nil {
		t.Fatal(err)
	}
	ground := make([]uint32, extract.MapSize*extract.MapSize)
	for i := range ground {
		ground[i] = 2
	}
	m.Layers[0].SetData(ground)
	m.Properties = append(m.Properties, tiled.File(extract.PropArea, "area/land.png"))
	if err := tiled.WriteMap(mapPath, m); err != nil {
		t.Fatal(err)
	}
	if _, err := stdout(t, func() error {
		return runRender([]string{mapPath, "--rom", romPath, "--layout", layoutPath, "--out", filepath.Join(dir, "land.png")})
	}); err != nil {
		t.Fatalf("a plain render reads the area map: %v", err)
	}
	out := filepath.Join(dir, "area.png")
	printed, err := stdout(t, func() error {
		return runRender([]string{mapPath, "--area", "--rom", romPath, "--layout", layoutPath, "--out", out})
	})
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(` pixels in \d+ of the 582 distinct tiles it may hold\n$`).Match(printed) {
		t.Fatalf("render printed %q", printed)
	}
	f, err := os.Open(out)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	if b := img.Bounds(); b.Dx() != 208 || b.Dy() != 208 {
		t.Fatalf("%d by %d", b.Dx(), b.Dy())
	}
	if _, err := stdout(t, func() error {
		return runRender([]string{mapPath, "--area", "--scale", "2", "--rom", romPath, "--out", out})
	}); err == nil || !strings.Contains(err.Error(), "--area") {
		t.Fatalf("scale with area: %v", err)
	}
}
