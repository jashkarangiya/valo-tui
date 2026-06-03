package vlr

import (
	"os"
	"testing"
)

func TestParseEventMatches(t *testing.T) {
	f, err := os.Open("testdata/event_matches.html")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	matches, err := parseEventMatches(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) < 5 {
		t.Fatalf("expected event matches, got %d", len(matches))
	}
	var winners int
	for _, m := range matches {
		if m.MatchID == 0 || len(m.Teams) != 2 {
			t.Errorf("bad event match: %+v", m)
		}
		for _, tm := range m.Teams {
			if tm.Name == "" {
				t.Errorf("empty team name: %+v", m)
			}
			if tm.IsWinner {
				winners++
			}
		}
	}
	if winners == 0 {
		t.Error("expected some mod-winner teams to be detected")
	}
	t.Logf("parsed %d event matches, %d winners", len(matches), winners)
}
