package data

import (
	"database/sql"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// seedDB writes the kv rows a worker would, into a temp DB, and points
// VALO_TUI_DB at it for the duration of the test.
func seedDB(t *testing.T, rows map[string]any) {
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
	for k, v := range rows {
		b, _ := json.Marshal(v)
		if _, err := db.Exec(`INSERT INTO kv VALUES (?, ?, ?)`, k, string(b), now); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("VALO_TUI_DB", path)
}

func TestGlobalLiveBucketsByRegionAndInternational(t *testing.T) {
	seedDB(t, map[string]any{
		"matches:live": []map[string]any{
			{"match_id": 1, "team1": map[string]any{"name": "Sentinels", "score": 1},
				"team2": map[string]any{"name": "NRG", "score": 0},
				"event": "VCT Americas Stage 1", "status": "live"},
		},
		"matches:upcoming": []map[string]any{
			{"match_id": 2, "team1": map[string]any{"name": "Fnatic"},
				"team2": map[string]any{"name": "Team Heretics"},
				"event": "VCT EMEA Stage 1", "status": "upcoming", "time": "21:00"},
		},
		"matches:completed": []map[string]any{
			{"match_id": 3, "team1": map[string]any{"name": "Gen.G", "score": 2},
				"team2": map[string]any{"name": "Paper Rex", "score": 1},
				"event": "Masters Toronto", "status": "completed"},
		},
		"events:active": []map[string]any{},
	})

	regions, intl := GlobalLive()

	if got := len(regions["Americas"].Live); got != 1 {
		t.Errorf("Americas live: want 1, got %d", got)
	}
	if regions["Americas"].Live[0].Team1.Name != "Sentinels" {
		t.Errorf("unexpected Americas match: %+v", regions["Americas"].Live[0])
	}
	if got := *regions["Americas"].Live[0].Team1.Score; got != 1 {
		t.Errorf("score coercion failed: want 1, got %d", got)
	}
	if got := len(regions["EMEA"].Next); got != 1 {
		t.Errorf("EMEA next: want 1, got %d", got)
	}
	if len(intl) != 1 || intl[0].Event != "Masters Toronto" {
		t.Errorf("international bucket wrong: %+v", intl)
	}
	// Masters must not also land in a regional bucket.
	if len(regions["Pacific"].Recent) != 0 {
		t.Errorf("international leaked into Pacific: %+v", regions["Pacific"].Recent)
	}
}

func TestLastUpdatedEmptyWhenNoDB(t *testing.T) {
	t.Setenv("VALO_TUI_DB", filepath.Join(t.TempDir(), "missing.db"))
	if ts := LastUpdated(); ts != "" {
		t.Errorf("want empty freshness for missing DB, got %q", ts)
	}
}
