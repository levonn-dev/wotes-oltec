package convert

import (
	"testing"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/romtest"
)

func TestBorrowedRuns(t *testing.T) {
	set := map[tileKey]bool{
		{Region: "swamp", Index: 5}: true, {Region: "swamp", Index: 1}: true, {Region: "swamp", Index: 2}: true,
		{Region: "valley", Index: 7}: true,
	}
	chunks, err := borrowedRuns(set, romtest.Regions, 10)
	if err != nil {
		t.Fatal(err)
	}
	// Regions in list order, indices ascending: valley 7 alone, swamp 1 and 2 as one run, the gap before swamp 5 splits it.
	want := []Chunk{
		{Kind: "borrowed", Region: "valley", First: 7, Count: 1, Source: romtest.Region.TileSource + 32*7, VRAM: 10},
		{Kind: "borrowed", Region: "swamp", First: 1, Count: 2, Source: romtest.Other.TileSource + 32, VRAM: 11},
		{Kind: "borrowed", Region: "swamp", First: 5, Count: 1, Source: romtest.Other.TileSource + 32*5, VRAM: 13},
	}
	if len(chunks) != len(want) {
		t.Fatalf("chunks %+v", chunks)
	}
	for i := range want {
		if chunks[i] != want[i] {
			t.Fatalf("chunk %d = %+v, want %+v", i, chunks[i], want[i])
		}
	}
}
