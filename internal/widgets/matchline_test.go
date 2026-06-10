package widgets

import (
	"testing"

	"github.com/jashkarangiya/valo-tui/internal/data"
)

func TestMatchLineHit(t *testing.T) {
	// Line: dot(2) + "NRG"(3) + " vs "(4) + "LEVIATÁN"(8) + …
	// ⇒ team1 cols [2,5), team2 cols [9,17).
	m := data.MatchCard{
		Team1: data.TeamSide{Name: "NRG"},
		Team2: data.TeamSide{Name: "LEVIATÁN"},
	}
	cases := []struct {
		col  int
		want string
		ok   bool
	}{
		{0, "", false},        // dot
		{1, "", false},        // dot space
		{2, "NRG", true},      // start of team1
		{4, "NRG", true},      // end of team1
		{5, "", false},        // space after team1
		{6, "", false},        // "vs"
		{8, "", false},        // space before team2
		{9, "LEVIATÁN", true}, // start of team2
		{16, "LEVIATÁN", true},
		{17, "", false}, // past team2
	}
	for _, c := range cases {
		got, ok := MatchLineHit(m, c.col)
		if got != c.want || ok != c.ok {
			t.Errorf("MatchLineHit(col=%d) = (%q,%v), want (%q,%v)", c.col, got, ok, c.want, c.ok)
		}
	}

	// A long name is clipped to 12 columns, so team2 shifts accordingly.
	long := data.MatchCard{
		Team1: data.TeamSide{Name: "Shopify Rebellion Gold"}, // clipped to 12
		Team2: data.TeamSide{Name: "G2"},
	}
	if name, ok := MatchLineHit(long, 2); !ok || name != "Shopify Rebellion Gold" {
		t.Errorf("clipped team1 hit failed: %q %v", name, ok)
	}
	// team2 starts at 2 + 12 + 4 = 18
	if name, ok := MatchLineHit(long, 18); !ok || name != "G2" {
		t.Errorf("team2 after clipped name failed: %q %v", name, ok)
	}
}
