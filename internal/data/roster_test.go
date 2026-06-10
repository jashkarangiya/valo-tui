package data

import "testing"

func TestRosterByTeamName(t *testing.T) {
	seedDB(t, map[string]any{
		"teams:index": map[string]any{"NRG": 1034, "LEVIATÁN": 2359},
		"team:1034": map[string]any{
			"team_id": 1034, "team": "NRG",
			"members": []map[string]any{
				{"id": 1, "alias": "Ethan", "name": "Ethan Arnold", "country": "us", "role": "", "captain": true},
				{"id": 2, "alias": "crashies", "name": "Austin Roberts", "country": "us", "role": ""},
				{"id": 3, "alias": "s0m", "name": "Sam Oh", "country": "us", "role": "head coach"},
			},
		},
	})

	r, ok := RosterByTeamName("NRG")
	if !ok {
		t.Fatal("RosterByTeamName(NRG) not ok")
	}
	if r.Team != "NRG" || r.TeamID != 1034 {
		t.Errorf("wrong team: %+v", r)
	}
	if len(r.Players()) != 2 {
		t.Errorf("expected 2 active players, got %d", len(r.Players()))
	}
	if len(r.Staff()) != 1 || r.Staff()[0].Role != "head coach" {
		t.Errorf("expected one coach in staff, got %+v", r.Staff())
	}
	if !r.Players()[0].Captain {
		t.Error("captain flag lost")
	}

	// Case-insensitive resolution still finds the team.
	if _, ok := RosterByTeamName("nrg"); !ok {
		t.Error("case-insensitive lookup failed")
	}
	// Unknown team and TBD resolve to not-ok.
	if _, ok := RosterByTeamName("Unknown Team"); ok {
		t.Error("unknown team should not resolve")
	}
	if _, ok := RosterByTeamName("TBD"); ok {
		t.Error("TBD should not resolve")
	}
}
