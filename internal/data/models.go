// Package data is the read-side access to the SQLite cache the Python worker
// writes. It is the Go port of valo_tui/data — same kv schema, same model
// shapes, so the Go TUI renders from exactly the same source of truth.
package data

import (
	"strconv"
	"strings"
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

// matchFromEventRaw maps a vlr.events.matches row (a `teams` list rather than
// team1/team2) into the shared card shape, mirroring MatchCard.from_event_raw.
func matchFromEventRaw(d map[string]any, eventName string) MatchCard {
	teams := asList(d["teams"])
	var t1, t2 map[string]any
	if len(teams) > 0 {
		t1 = asMap(teams[0])
	}
	if len(teams) > 1 {
		t2 = asMap(teams[1])
	}
	event := eventName
	if event == "" {
		event = s(d["event"])
	}
	phase := s(d["phase"])
	if phase == "" {
		phase = s(d["event_phase"])
	}
	return MatchCard{
		MatchID: deref(i(d["match_id"])),
		Team1:   teamFromRaw(t1),
		Team2:   teamFromRaw(t2),
		Event:   event,
		Phase:   phase,
		Status:  normStatus(d),
		Time:    s(d["time"]),
		Date:    s(d["date"]),
	}
}

// normStatus normalises an event-match status into upcoming|live|completed,
// inferring from a decided winner when no usable status string is present.
func normStatus(d map[string]any) string {
	raw := strings.ToLower(s(d["status"]))
	switch {
	case strings.Contains(raw, "live"):
		return "live"
	case strings.Contains(raw, "complet"), strings.Contains(raw, "final"):
		return "completed"
	case strings.Contains(raw, "upcom"), strings.Contains(raw, "tbd"),
		strings.Contains(raw, "soon"), strings.Contains(raw, "sched"):
		return "upcoming"
	}
	for _, t := range asList(d["teams"]) {
		if b, _ := asMap(t)["is_winner"].(bool); b {
			return "completed"
		}
	}
	return "upcoming"
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

func asList(v any) []any {
	if l, ok := v.([]any); ok {
		return l
	}
	return nil
}

// f coerces an arbitrary JSON value into *float64 (for adr, hs_pct).
func f(v any) *float64 {
	switch t := v.(type) {
	case float64:
		return &t
	case int:
		x := float64(t)
		return &x
	case string:
		if x, err := strconv.ParseFloat(t, 64); err == nil {
			return &x
		}
	}
	return nil
}

func strList(v any) []string {
	out := []string{}
	for _, e := range asList(v) {
		if str, ok := e.(string); ok {
			out = append(out, str)
		}
	}
	return out
}
