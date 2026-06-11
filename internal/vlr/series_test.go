package vlr

import (
	"os"
	"testing"
)

func TestClassifyNotes(t *testing.T) {
	cases := []struct {
		notes                     []string
		bestOf, status, remaining string
	}{
		{[]string{"final", "Bo3"}, "Bo3", "final", ""},   // completed
		{[]string{"Bo3"}, "Bo3", "", ""},                 // upcoming, no timer
		{[]string{"18h 0m", "Bo3"}, "Bo3", "", "18h 0m"}, // upcoming w/ countdown
		{[]string{"Bo3", "1d 9h"}, "Bo3", "", "1d 9h"},   // countdown after best-of
		{[]string{"live", "Bo5"}, "Bo5", "live", ""},     // live
		{[]string{"Bo1"}, "Bo1", "", ""},                 // Bo1
		{nil, "", "", ""},                                // nothing
	}
	for _, c := range cases {
		bo, st, rem := classifyNotes(c.notes)
		if bo != c.bestOf || st != c.status || rem != c.remaining {
			t.Errorf("classifyNotes(%v) = (%q,%q,%q), want (%q,%q,%q)",
				c.notes, bo, st, rem, c.bestOf, c.status, c.remaining)
		}
	}
}

func TestParseSeries(t *testing.T) {
	f, err := os.Open("testdata/match.html")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	s, err := parseSeries(f, 684615)
	if err != nil {
		t.Fatal(err)
	}

	if len(s.Info.Teams) != 2 {
		t.Fatalf("expected 2 teams, got %d: %+v", len(s.Info.Teams), s.Info.Teams)
	}
	if s.Info.Teams[0].Name == "" || s.Info.Teams[1].Name == "" {
		t.Errorf("team names missing: %+v", s.Info.Teams)
	}
	if s.Info.Teams[0].Short == "" || s.Info.Teams[1].Short == "" {
		t.Errorf("team shorts not backfilled: %+v", s.Info.Teams)
	}
	if s.Info.Teams[0].ID == 0 || s.Info.Teams[1].ID == 0 {
		t.Errorf("team ids not captured (needed for roster lookup): %+v", s.Info.Teams)
	}
	if len(s.Info.Score) != 2 {
		t.Errorf("expected a 2-element series score, got %v", s.Info.Score)
	}
	if s.Info.BestOf == "" || s.Info.StatusNote == "" {
		t.Errorf("missing best_of/status: bo=%q status=%q", s.Info.BestOf, s.Info.StatusNote)
	}
	if s.Info.Event == "" {
		t.Errorf("missing event name")
	}
	if len(s.Info.MapActions) == 0 {
		t.Errorf("expected parsed veto actions")
	}

	if len(s.Maps) == 0 {
		t.Fatal("no maps parsed")
	}
	for _, m := range s.Maps {
		if m.MapName == "" || m.MapName == "all" {
			t.Errorf("bad map name %q", m.MapName)
		}
		if len(m.Teams) != 2 || m.Teams[0].Short == "" {
			t.Errorf("map %s: bad teams %+v", m.MapName, m.Teams)
		}
		if len(m.Players) != 10 {
			t.Errorf("map %s: expected 10 players, got %d", m.MapName, len(m.Players))
		}
		for _, p := range m.Players {
			if p.Name == "" || p.TeamShort == "" || len(p.Agents) == 0 {
				t.Errorf("map %s: incomplete player %+v", m.MapName, p)
			}
			if p.ACS == nil || p.K == nil || p.D == nil || p.A == nil {
				t.Errorf("map %s: %s missing core stats", m.MapName, p.Name)
			}
		}
		if len(m.Rounds) == 0 {
			t.Errorf("map %s: no rounds parsed", m.MapName)
		}
		for _, r := range m.Rounds {
			if r.WinnerSide != "Attacker" && r.WinnerSide != "Defender" {
				t.Errorf("map %s round %d: bad side %q", m.MapName, r.Number, r.WinnerSide)
			}
			if r.WinnerTeamShort == "" {
				t.Errorf("map %s round %d: no winner short", m.MapName, r.Number)
			}
		}
	}
	t.Logf("parsed %d maps, teams %s/%s score %v", len(s.Maps),
		s.Info.Teams[0].Short, s.Info.Teams[1].Short, s.Info.Score)
}
