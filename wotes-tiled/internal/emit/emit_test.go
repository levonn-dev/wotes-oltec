package emit

import "testing"

func TestDepfile(t *testing.T) {
	got := Depfile("build/gen/o.s", []string{"maps/o.tmj", "maps/my tiles/a.png"})
	if got != "build/gen/o.s: maps/o.tmj maps/my\\ tiles/a.png\n" {
		t.Fatalf("Depfile = %q", got)
	}
}
