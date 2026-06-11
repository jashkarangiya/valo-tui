package data

import (
	"fmt"
	"strings"
)

// RosterMember is one lineup entry. Role is "" for active players, else a staff
// label ("head coach", "manager", …).
type RosterMember struct {
	ID      int
	Alias   string
	Name    string
	Country string
	Role    string
	Captain bool
}

// IsPlayer reports an active player (no staff role).
func (m RosterMember) IsPlayer() bool { return m.Role == "" }

// Roster is a team's full lineup, read from the team:{id} cache key.
type Roster struct {
	TeamID  int
	Team    string
	Members []RosterMember
}

// Players / Staff split the lineup for rendering.
func (r Roster) Players() []RosterMember { return r.filter(true) }
func (r Roster) Staff() []RosterMember   { return r.filter(false) }

func (r Roster) filter(players bool) []RosterMember {
	var out []RosterMember
	for _, m := range r.Members {
		if m.IsPlayer() == players {
			out = append(out, m)
		}
	}
	return out
}

func rosterFromRaw(o map[string]any) Roster {
	r := Roster{TeamID: deref(i(o["team_id"])), Team: s(o["team"])}
	for _, raw := range asList(o["members"]) {
		m := asMap(raw)
		if m == nil {
			continue
		}
		isCaptain, _ := m["captain"].(bool)
		r.Members = append(r.Members, RosterMember{
			ID:      deref(i(m["id"])),
			Alias:   s(m["alias"]),
			Name:    s(m["name"]),
			Country: s(m["country"]),
			Role:    s(m["role"]),
			Captain: isCaptain,
		})
	}
	return r
}

// teamIndex reads the teams:index name→id map the fetcher maintains.
func teamIndex() map[string]int {
	o := getObject("teams:index")
	if o == nil {
		return nil
	}
	out := make(map[string]int, len(o))
	for name, v := range o {
		if id := deref(i(v)); id != 0 {
			out[name] = id
		}
	}
	return out
}

// resolveTeamID maps a display name to its team id (exact, then case-insensitive).
func resolveTeamID(name string) (int, bool) {
	idx := teamIndex()
	if id, ok := idx[name]; ok {
		return id, true
	}
	want := strings.ToLower(strings.TrimSpace(name))
	for n, id := range idx {
		if strings.ToLower(strings.TrimSpace(n)) == want {
			return id, true
		}
	}
	return 0, false
}

// RosterByTeamName resolves a team's display name to its cached roster. ok is
// false when the team is unknown or its roster hasn't been fetched yet.
func RosterByTeamName(name string) (Roster, bool) {
	if name == "" || name == "TBD" {
		return Roster{}, false
	}
	id, ok := resolveTeamID(name)
	if !ok {
		return Roster{}, false
	}
	o := getObject(fmt.Sprintf("team:%d", id))
	if o == nil {
		return Roster{}, false
	}
	return rosterFromRaw(o), true
}
