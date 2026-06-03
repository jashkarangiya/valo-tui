package vlr

import (
	"os"
	"testing"
)

// TestParseMatches runs the parser against a saved /matches page so selector
// regressions (vlr.gg HTML drift) are caught without hitting the network.
func TestParseMatches(t *testing.T) {
	f, err := os.Open("testdata/matches.html")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	matches, err := parseMatches(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) < 10 {
		t.Fatalf("expected a full match list, got %d", len(matches))
	}

	var live int
	for _, m := range matches {
		// Every parsed row must have an id, two named teams, and a status.
		if m.MatchID == 0 {
			t.Errorf("match with no id: %+v", m)
		}
		if m.Team1.Name == "" || m.Team2.Name == "" {
			t.Errorf("match with empty team name: %+v", m)
		}
		switch m.Status {
		case "live", "upcoming", "completed":
		default:
			t.Errorf("unexpected status %q in %+v", m.Status, m)
		}
		// Event should not still contain the series sub-label.
		if m.EventPhase != "" && m.Event != "" &&
			len(m.Event) >= len(m.EventPhase) && m.Event[:len(m.EventPhase)] == m.EventPhase {
			t.Errorf("event still has series prefix: event=%q series=%q", m.Event, m.EventPhase)
		}
		if m.Status == "live" {
			live++
		}
	}
	t.Logf("parsed %d matches, %d live", len(matches), live)
}
