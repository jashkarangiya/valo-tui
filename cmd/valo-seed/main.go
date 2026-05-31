// Command valo-seed populates the SQLite cache with realistic sample data so
// every screen renders without a live fetcher. It writes the same kv schema the
// (future) Go fetcher will: `go run ./cmd/valo-seed`.
package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type obj = map[string]any
type arr = []any

func main() {
	path := dbPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		fail(err)
	}
	db, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		fail(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS kv (key TEXT PRIMARY KEY, value JSON NOT NULL, updated_at TEXT NOT NULL)`); err != nil {
		fail(err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	put := func(key string, val any) {
		b, _ := json.Marshal(val)
		if _, err := db.Exec(`INSERT OR REPLACE INTO kv (key, value, updated_at) VALUES (?, json(?), ?)`, key, string(b), now); err != nil {
			fail(err)
		}
	}

	const eventID = 2280
	eventName := "VCT 2026 Americas Stage 1"

	// Events list (drives [e] events + overview banner).
	put("events:active", arr{
		obj{"id": eventID, "name": eventName, "status": "ongoing", "region": "Americas",
			"prize": "$250,000", "start_text": "Apr 5", "end_text": "May 31"},
		obj{"id": 2281, "name": "VCT 2026 EMEA Stage 1", "status": "ongoing", "region": "EMEA",
			"prize": "$250,000", "start_text": "Apr 5", "end_text": "May 31"},
		obj{"id": 2282, "name": "VCT 2026 Pacific Stage 1", "status": "upcoming", "region": "Pacific",
			"prize": "$250,000", "start_text": "Jun 7", "end_text": "Jul 26"},
		obj{"id": 2290, "name": "Masters Toronto", "status": "upcoming", "region": "International",
			"prize": "$1,000,000", "start_text": "Aug 2", "end_text": "Aug 17"},
	})

	// Event matches (drives results/fixtures/standings/teams + bracket).
	put(fmt.Sprintf("event:matches:%d", eventID), arr{
		// playoffs — upper bracket
		em(101, "Upper Semifinals", "completed", "", "Sentinels", "SEN", 2, true, "G2 Esports", "G2", 1, false),
		em(102, "Upper Semifinals", "completed", "", "NRG", "NRG", 2, true, "100 Thieves", "100T", 0, false),
		em(103, "Upper Final", "completed", "", "Sentinels", "SEN", 2, true, "NRG", "NRG", 1, false),
		// lower bracket
		em(104, "Lower Round 1", "completed", "", "LOUD", "LOUD", 2, true, "Evil Geniuses", "EG", 0, false),
		em(105, "Lower Round 1", "completed", "", "MIBR", "MIBR", 2, true, "KRÜ Esports", "KRU", 1, false),
		em(106, "Lower Round 2", "completed", "", "G2 Esports", "G2", 2, true, "LOUD", "LOUD", 1, false),
		em(107, "Lower Round 2", "completed", "", "100 Thieves", "100T", 2, true, "MIBR", "MIBR", 0, false),
		em(108, "Lower Final", "completed", "", "NRG", "NRG", 2, true, "G2 Esports", "G2", 1, false),
		em(109, "Grand Final", "live", "LIVE", "Sentinels", "SEN", 2, false, "NRG", "NRG", 2, false),
		// group stage upcoming (drives fixtures)
		em(110, "Group Stage", "upcoming", "Sat 18:00", "LOUD", "LOUD", 0, false, "Evil Geniuses", "EG", 0, false),
		em(111, "Group Stage", "upcoming", "Sat 21:00", "KRÜ Esports", "KRU", 0, false, "MIBR", "MIBR", 0, false),
	})

	// Series detail for the Upper Semifinal (drives [enter] match detail).
	put("series:101", obj{
		"info": obj{
			"match_id": 101, "event": eventName, "event_phase": "Upper Semifinals",
			"best_of": "Bo3", "status_note": "Completed",
			"teams": arr{obj{"name": "Sentinels", "short": "SEN"}, obj{"name": "G2 Esports", "short": "G2"}},
			"score": arr{2, 1},
			"map_actions": arr{
				obj{"action": "pick", "team": "SEN", "map": "Lotus"},
				obj{"action": "pick", "team": "G2", "map": "Ascent"},
				obj{"action": "remaining", "team": "", "map": "Haven"},
			},
		},
		"maps": arr{
			mapScore("Lotus", "SEN", 13, "G2", 9, sideSEN(), sideG2()),
			mapScore("Ascent", "SEN", 11, "G2", 13, sideSEN(), sideG2()),
			mapScore("Haven", "SEN", 13, "G2", 8, sideSEN(), sideG2()),
		},
	})

	// Polling lists (drive [l] global live + [h] home).
	put("matches:live", arr{
		live("Sentinels", 1, "NRG", 1, "VCT 2026 Americas Stage 1", "Grand Final"),
		live("DRX", 0, "Gen.G", 1, "VCT 2026 Pacific Stage 1", "Playoffs"),
	})
	put("matches:upcoming", arr{
		upcoming("FNATIC", "Team Heretics", "VCT 2026 EMEA Stage 1", "Playoffs", "21:00"),
		upcoming("EDG", "Bilibili Gaming", "VCT 2026 China Stage 1", "Group Stage", "Sat 12:00"),
	})
	put("matches:completed", arr{
		completed("G2 Esports", 2, "LOUD", 1, "VCT 2026 Americas Stage 1", "Lower Round 2"),
		completed("Team Liquid", 2, "Natus Vincere", 0, "VCT 2026 EMEA Stage 1", "Playoffs"),
		completed("Paper Rex", 3, "Gen.G", 2, "Masters Toronto", "Grand Final"),
	})

	fmt.Printf("seeded %s\n", path)
}

// em builds an event-match row (teams[] shape).
func em(id int, phase, status, t string, a, as string, asc int, aw bool, b, bs string, bsc int, bw bool) obj {
	return obj{
		"match_id": id, "phase": phase, "status": status, "time": t,
		"teams": arr{
			obj{"name": a, "short": as, "score": asc, "is_winner": aw},
			obj{"name": b, "short": bs, "score": bsc, "is_winner": bw},
		},
	}
}

func live(a string, asc int, b string, bsc int, event, phase string) obj {
	return obj{"match_id": 9001, "team1": obj{"name": a, "score": asc},
		"team2": obj{"name": b, "score": bsc}, "event": event, "event_phase": phase,
		"status": "live", "time": "LIVE"}
}

func upcoming(a, b, event, phase, when string) obj {
	return obj{"team1": obj{"name": a}, "team2": obj{"name": b},
		"event": event, "event_phase": phase, "status": "upcoming", "time": when}
}

func completed(a string, asc int, b string, bsc int, event, phase string) obj {
	return obj{"team1": obj{"name": a, "score": asc}, "team2": obj{"name": b, "score": bsc},
		"event": event, "event_phase": phase, "status": "completed"}
}

// mapScore builds a per-map block with scoreboards + a little round momentum.
func mapScore(name, t1 string, s1 int, t2 string, s2 int, p1, p2 arr) obj {
	rounds := arr{}
	for i := 0; i < s1+s2; i++ {
		side, short := "Attacker", t1
		if i%2 == 1 {
			side, short = "Defender", t2
		}
		rounds = append(rounds, obj{"number": i + 1, "winner_side": side, "winner_team_short": short})
	}
	return obj{
		"map_name": name,
		"teams":    arr{obj{"short": t1, "score": s1}, obj{"short": t2, "score": s2}},
		"players":  append(append(arr{}, p1...), p2...),
		"rounds":   rounds,
	}
}

func player(name, agent, short string, acs, k, d, a, fk, fd int, adr, hs float64) obj {
	return obj{"name": name, "agents": arr{agent}, "team_short": short,
		"acs": acs, "k": k, "d": d, "a": a, "fk": fk, "fd": fd, "adr": adr, "hs_pct": hs}
}

func sideSEN() arr {
	return arr{
		player("zekken", "raze", "SEN", 271, 21, 14, 5, 4, 2, 168, 28),
		player("johnqt", "jett", "SEN", 245, 19, 15, 3, 5, 3, 152, 31),
		player("Zellsis", "fade", "SEN", 198, 15, 13, 9, 1, 2, 141, 22),
		player("N4RRATE", "omen", "SEN", 176, 13, 14, 7, 1, 1, 128, 24),
		player("Sacy", "killjoy", "SEN", 162, 12, 13, 11, 2, 1, 119, 19),
	}
}

func sideG2() arr {
	return arr{
		player("valyn", "jett", "G2", 233, 18, 16, 4, 3, 4, 149, 26),
		player("trent", "raze", "G2", 219, 17, 17, 3, 2, 2, 144, 23),
		player("JonahP", "skye", "G2", 187, 14, 15, 8, 1, 1, 133, 21),
		player("leaf", "omen", "G2", 171, 13, 16, 6, 2, 2, 124, 25),
		player("jawgemo", "killjoy", "G2", 158, 11, 15, 9, 1, 1, 116, 18),
	}
}

func dbPath() string {
	if raw := os.Getenv("VALO_TUI_DB"); raw != "" {
		return raw
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cache", "valo-tui", "cache.db")
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "valo-seed:", err)
	os.Exit(1)
}
