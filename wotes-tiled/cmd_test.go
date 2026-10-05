package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/romtest"
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
