package app

import (
	"database/sql"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	_ "modernc.org/sqlite"

	"github.com/jashkarangiya/valo-tui/internal/screens"
)

// seed writes a representative cache into a temp DB and points VALO_TUI_DB at it.
func seed(t *testing.T) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "cache.db")
	db, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE kv (key TEXT PRIMARY KEY, value JSON NOT NULL, updated_at TEXT NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	put := func(k string, v any) {
		b, _ := json.Marshal(v)
		if _, err := db.Exec(`INSERT INTO kv VALUES (?,?,?)`, k, string(b), now); err != nil {
			t.Fatal(err)
		}
	}
	type o = map[string]any
	type a = []any

	put("events:active", a{o{"id": 1, "name": "VCT 2026 Americas Stage 1",
		"status": "ongoing", "region": "Americas", "prize": "$250,000",
		"start_text": "Apr 5", "end_text": "May 31"}})

	em := func(id int, phase, status string, an string, asc int, aw bool, bn string, bsc int, bw bool) o {
		return o{"match_id": id, "phase": phase, "status": status,
			"teams": a{
				o{"name": an, "short": an[:3], "score": asc, "is_winner": aw},
				o{"name": bn, "short": bn[:3], "score": bsc, "is_winner": bw},
			}}
	}
	put("event:matches:1", a{
		em(101, "Upper Semifinals", "completed", "Sentinels", 2, true, "G2 Esports", 1, false),
		em(102, "Upper Final", "completed", "Sentinels", 2, true, "NRG Esports", 1, false),
		em(103, "Grand Final", "live", "Sentinels", 1, false, "NRG Esports", 1, false),
		em(104, "Group Stage", "upcoming", "LOUD Gaming", 0, false, "Evil Geniuses", 0, false),
	})
	put("series:101", o{
		"info": o{"match_id": 101, "event": "VCT 2026 Americas Stage 1",
			"event_phase": "Upper Semifinals", "best_of": "Bo3", "status_note": "Completed",
			"teams": a{o{"name": "Sentinels", "short": "SEN"}, o{"name": "G2 Esports", "short": "G2"}},
			"score": a{2, 1}},
		"maps": a{o{"map_name": "Lotus",
			"teams":   a{o{"short": "SEN", "score": 13}, o{"short": "G2", "score": 9}},
			"players": a{o{"name": "zekken", "agents": a{"raze"}, "team_short": "SEN", "acs": 271, "k": 21, "d": 14, "a": 5}},
			"rounds":  a{o{"number": 1, "winner_side": "Attacker", "winner_team_short": "SEN"}}}},
	})
	put("matches:live", a{o{"match_id": 9001, "team1": o{"name": "Sentinels", "score": 1},
		"team2": o{"name": "NRG", "score": 1}, "event": "VCT 2026 Americas Stage 1", "status": "live"}})
	put("matches:upcoming", a{})
	put("matches:completed", a{})

	t.Setenv("VALO_TUI_DB", path)
}

func key(s string) tea.KeyPressMsg {
	switch s {
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	default:
		return tea.KeyPressMsg{Code: []rune(s)[0], Text: s}
	}
}

// run applies a message and returns the updated model, executing any command
// once so emitted follow-up messages (e.g. CloseOverlay) are delivered.
func run(t *testing.T, m tea.Model, msg tea.Msg) tea.Model {
	t.Helper()
	next, cmd := m.Update(msg)
	if cmd != nil {
		if follow := cmd(); follow != nil {
			// Only re-deliver routing messages the shell acts on synchronously.
			switch follow.(type) {
			case screens.CloseOverlayMsg, screens.OpenBracketMsg, screens.EnterAppMsg:
				next, _ = next.Update(follow)
			}
		}
	}
	return next
}

func contentOf(m tea.Model) string { return m.View().Content }

func TestFullNavigationRendersEveryScreen(t *testing.T) {
	seed(t)
	var m tea.Model = New(140, 44)

	// Enter the app from the splash.
	m = run(t, m, screens.EnterAppMsg{})
	if !strings.Contains(contentOf(m), "global live") {
		t.Fatalf("expected global live after entry, got:\n%s", contentOf(m))
	}

	// Global routes.
	for _, tc := range []struct{ key, want string }{
		{"h", "home"},
		{"a", "about"},
		{"e", "events"},
	} {
		m = run(t, m, key(tc.key))
		if !strings.Contains(contentOf(m), tc.want) {
			t.Errorf("route %q: expected %q in view", tc.key, tc.want)
		}
	}
	if !strings.Contains(contentOf(m), "VCT 2026 Americas") {
		t.Errorf("events list should show the seeded event")
	}

	// Drill into the event (Enter on the events table).
	m = run(t, m, key("enter"))
	if !strings.Contains(contentOf(m), "overview") {
		t.Fatalf("Enter on events should open the overview, got:\n%s", contentOf(m))
	}

	// Every event sub-page must render.
	for _, tc := range []struct{ key, want string }{
		{"r", "results"},
		{"f", "fixtures"},
		{"t", "standings"},
		{"m", "teams"},
		{"b", "bracket"},
	} {
		m = run(t, m, key(tc.key))
		if !strings.Contains(contentOf(m), tc.want) {
			t.Errorf("event route %q: expected %q in view, got:\n%s", tc.key, tc.want, contentOf(m))
		}
	}
	// Standings should derive a real record from the completed matches.
	m = run(t, m, key("t"))
	if !strings.Contains(contentOf(m), "Sentinels") {
		t.Errorf("standings should list Sentinels")
	}

	// Bracket → Enter opens the match-detail overlay for the selected match.
	m = run(t, m, key("b"))
	m = run(t, m, key("enter"))
	overlay := contentOf(m)
	if !strings.Contains(overlay, "Sentinels") || !strings.Contains(overlay, "Lotus") {
		t.Errorf("overlay should show the series detail (teams + map), got:\n%s", overlay)
	}

	// Esc closes the overlay and returns to the bracket.
	m = run(t, m, key("esc"))
	if !strings.Contains(contentOf(m), "bracket") {
		t.Errorf("esc should return from overlay to the bracket, got:\n%s", contentOf(m))
	}
}

func click(x, y int) tea.MouseClickMsg {
	return tea.MouseClickMsg{Button: tea.MouseLeft, X: x, Y: y}
}

func TestMouseClickNavigatesRailAndRows(t *testing.T) {
	seed(t)
	var m tea.Model = New(140, 40)
	m = run(t, m, screens.EnterAppMsg{})

	// Click "events" in the rail (text line index 4 ⇒ y=7).
	m = run(t, m, click(6, 7))
	if !strings.Contains(contentOf(m), "events") {
		t.Fatalf("clicking the rail should open events, got:\n%s", contentOf(m))
	}

	// Click the first event row (content table first row ⇒ y=7) to drill in.
	m = run(t, m, click(40, 7))
	if !strings.Contains(contentOf(m), "overview") {
		t.Fatalf("clicking an event row should open its overview, got:\n%s", contentOf(m))
	}

	// The event rail is now present; click "results" (text line index 11 ⇒ y=14).
	m = run(t, m, click(6, 14))
	if !strings.Contains(contentOf(m), "results") {
		t.Fatalf("clicking the event rail should open results, got:\n%s", contentOf(m))
	}
}
