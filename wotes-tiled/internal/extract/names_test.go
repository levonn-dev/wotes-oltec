package extract

import "testing"

func TestDungeonNames(t *testing.T) {
	for m := 0; m < 38; m++ {
		if name, err := DungeonName(m); err != nil || name == "" {
			t.Errorf("DungeonName(%#x) = %q, %v", m, name, err)
		}
	}
	if name, _ := DungeonName(2); name != "Gateway Passage" {
		t.Fatalf("map 2 is %q", name)
	}
	if name, _ := DungeonName(0x0E); name != "Oltec Tradeway" {
		t.Fatalf("map 0x0E is %q", name)
	}
	if name, _ := DungeonName(0x07); name != "Azcan Temple" {
		t.Fatalf("map 0x07 is %q", name)
	}
	for _, m := range []int{-1, 38, 0x3F} {
		if _, err := DungeonName(m); err == nil {
			t.Errorf("DungeonName(%d) did not fail", m)
		}
	}
}
