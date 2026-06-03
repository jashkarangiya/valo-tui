// Package vlr is a small vlr.gg scraper. Its JSON output is shaped to match
// exactly what internal/data's readers consume (the kv cache contract), so the
// fetcher can write straight into the cache and the TUI renders it unchanged.
//
// Selectors are ported from the Python vlrdevapi library (validated against a
// live testdata fixture).
package vlr

// Team is one side of a match. JSON tags match data.teamFromRaw.
type Team struct {
	Name  string `json:"name"`
	Score *int   `json:"score"`
}

// Match is a single match row. JSON tags match data.matchFromRaw, so a
// []Match marshals straight into the matches:live|upcoming|completed cache keys.
type Match struct {
	MatchID    int    `json:"match_id"`
	Team1      Team   `json:"team1"`
	Team2      Team   `json:"team2"`
	Event      string `json:"event"`
	EventPhase string `json:"event_phase"`
	Status     string `json:"status"` // upcoming | live | completed
	Time       string `json:"time,omitempty"`
	Date       string `json:"date,omitempty"`
}

// Event is a tournament summary. JSON tags match data.eventFromRaw, so a
// []Event marshals straight into the events:active cache key.
type Event struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Status    string `json:"status"` // ongoing | upcoming | completed
	Region    string `json:"region,omitempty"`
	Prize     string `json:"prize,omitempty"`
	StartText string `json:"start_text,omitempty"`
	EndText   string `json:"end_text,omitempty"`
}

// EventTeam is one side of an event match. JSON tags match data.teamFromRaw +
// the is_winner used by bracket/standings reconstruction.
type EventTeam struct {
	Name     string `json:"name"`
	Score    *int   `json:"score"`
	IsWinner bool   `json:"is_winner"`
}

// EventMatch is a match within an event (teams[] shape). JSON tags match
// data.matchFromEventRaw, so a []EventMatch marshals into event:matches:{id}.
type EventMatch struct {
	MatchID int         `json:"match_id"`
	Teams   []EventTeam `json:"teams"`
	Phase   string      `json:"phase"`
	Status  string      `json:"status"` // upcoming | live | completed
	Time    string      `json:"time,omitempty"`
}
