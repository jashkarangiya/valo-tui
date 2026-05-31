// Package data is the read-side access to the SQLite cache the Python worker
// writes. It is the Go port of valo_tui/data — same kv schema, same model
// shapes, so the Go TUI renders from exactly the same source of truth.
package data

import (
	"strconv"
)

// REGIONS are the four Tier-1 regional leagues used to bucket global-live.
var REGIONS = [...]string{"Americas", "EMEA", "Pacific", "China"}

// TeamSide is one side of a match. Mirrors models.TeamSide.
type TeamSide struct {
	Name    string
	Score   *int
	Country string
	Short   string
}

func teamFromRaw(d map[string]any) TeamSide {
	if d == nil {
		d = map[string]any{}
	}
	short := s(d["short"])
	if short == "" {
		short = s(d["tag"])
	}
	name := s(d["name"])
	if name == "" {
		name = "TBD"
	}
	return TeamSide{
		Name:    name,
		Score:   i(d["score"]),
		Country: s(d["country"]),
		Short:   short,
	}
}

// MatchCard is a single match row shared by live/upcoming/completed views.
type MatchCard struct {
	MatchID int
	Team1   TeamSide
	Team2   TeamSide
	Event   string
	Phase   string
	Status  string // "upcoming" | "live" | "completed"
	Time    string
	Date    string
}

func (m MatchCard) IsLive() bool { return m.Status == "live" }

func matchFromRaw(d map[string]any) MatchCard {
	phase := s(d["event_phase"])
	if phase == "" {
		phase = s(d["phase"])
	}
	status := s(d["status"])
	if status == "" {
		status = "upcoming"
	}
	id := i(d["match_id"])
	return MatchCard{
		MatchID: deref(id),
		Team1:   teamFromRaw(asMap(d["team1"])),
		Team2:   teamFromRaw(asMap(d["team2"])),
		Event:   s(d["event"]),
		Phase:   phase,
		Status:  status,
		Time:    s(d["time"]),
		Date:    s(d["date"]),
	}
}

// EventCard is a tournament row for the events list.
type EventCard struct {
	ID     int
	Name   string
	Status string // "upcoming" | "ongoing" | "completed"
	Region string
	Prize  string
	Start  string
	End    string
}

func eventFromRaw(d map[string]any) EventCard {
	status := s(d["status"])
	if status == "" {
		status = "ongoing"
	}
	start := s(d["start_text"])
	if start == "" {
		start = s(d["start_date"])
	}
	end := s(d["end_text"])
	if end == "" {
		end = s(d["end_date"])
	}
	return EventCard{
		ID:     deref(i(d["id"])),
		Name:   s(d["name"]),
		Status: status,
		Region: s(d["region"]),
		Prize:  s(d["prize"]),
		Start:  start,
		End:    end,
	}
}

// ── coercion helpers (mirror models._i and the `or ""` fallbacks) ──────────

// s coerces an arbitrary JSON value into a string, defaulting to "".
func s(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case nil:
		return ""
	default:
		return ""
	}
}

// i coerces an arbitrary JSON value into *int, like Python's _i. JSON numbers
// decode as float64; the upstream also sometimes stores numbers as strings.
func i(v any) *int {
	switch t := v.(type) {
	case float64:
		n := int(t)
		return &n
	case int:
		return &t
	case string:
		if n, err := strconv.Atoi(t); err == nil {
			return &n
		}
	}
	return nil
}

func deref(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

func asMap(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return nil
}
