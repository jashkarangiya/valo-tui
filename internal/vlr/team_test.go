package vlr

import (
	"os"
	"testing"
)

func TestParseTeam(t *testing.T) {
	f, err := os.Open("testdata/team.html")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	r, err := parseTeam(f)
	if err != nil {
		t.Fatal(err)
	}
	if r.Team == "" {
		t.Error("team name not parsed")
	}
	if len(r.Members) < 5 {
		t.Fatalf("expected a full roster, got %d members", len(r.Members))
	}

	var players, withCountry, captains int
	for _, m := range r.Members {
		if m.Alias == "" && m.Name == "" {
			t.Errorf("empty member: %+v", m)
		}
		if m.ID == 0 {
			t.Errorf("member %q missing id", m.Alias)
		}
		if m.Role == "" {
			players++
		}
		if m.Country != "" {
			withCountry++
		}
		if m.Captain {
			captains++
		}
	}
	if players < 5 {
		t.Errorf("expected >=5 active players, got %d", players)
	}
	if withCountry == 0 {
		t.Error("no country flags parsed")
	}
	t.Logf("%s: %d members (%d players), %d captains", r.Team, len(r.Members), players, captains)
}
