// Command valo-fetcher scrapes vlr.gg and writes the SQLite cache the TUI
// reads. Slice 1 populates matches:live.
//
//	valo-fetcher --once
//	valo-fetcher --watch --interval 30s
package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	_ "modernc.org/sqlite"

	"github.com/jashkarangiya/valo-tui/internal/vlr"
)

func main() {
	_ = flag.Bool("once", true, "fetch once and exit (default)")
	watch := flag.Bool("watch", false, "fetch repeatedly on --interval instead of once")
	interval := flag.Duration("interval", 30*time.Second, "live-match refresh interval for --watch")
	// Per-key cadences default to a sensible floor; override per deployment.
	resultsInterval := flag.Duration("results-interval", 0, "completed-match refresh interval (0 = max(5m, interval))")
	eventsInterval := flag.Duration("events-interval", 0, "events + event-match-list refresh interval (0 = max(15m, interval))")
	seriesInterval := flag.Duration("series-interval", 0, "live match-detail refresh interval (0 = max(1m, interval))")
	dbFlag := flag.String("db", "", "cache db path (default $VALO_TUI_DB or ~/.cache/valo-tui/cache.db)")
	flag.Parse()

	db := openDB(resolveDB(*dbFlag))
	defer db.Close()
	client := vlr.New()

	if !*watch {
		fetchMatches(db, client)
		fetchResults(db, client)
		fetchEvents(db, client)
		fetchLiveSeries(db, client)
		backfillSeries(db, client)
		return
	}

	// Per-key cadences: live scores and the broadcast view change fast; results
	// are slower; events and their (~40-page) match lists change slowly. Flags
	// override; otherwise each key falls back to a polite floor. This keeps live
	// data fresh without hammering vlr.gg.
	matchEvery := *interval
	resultsEvery := orDefault(*resultsInterval, maxDur(5*time.Minute, *interval))
	eventsEvery := orDefault(*eventsInterval, maxDur(15*time.Minute, *interval))
	seriesEvery := orDefault(*seriesInterval, maxDur(1*time.Minute, *interval))
	log.Printf("watching · matches %s · series %s · results %s · events %s",
		matchEvery, seriesEvery, resultsEvery, eventsEvery)

	fetchMatches(db, client)
	fetchResults(db, client)
	fetchEvents(db, client)
	fetchLiveSeries(db, client)
	backfillSeries(db, client)

	matchesT := time.NewTicker(matchEvery)
	seriesT := time.NewTicker(seriesEvery)
	resultsT := time.NewTicker(resultsEvery)
	eventsT := time.NewTicker(eventsEvery)
	defer matchesT.Stop()
	defer seriesT.Stop()
	defer resultsT.Stop()
	defer eventsT.Stop()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	for {
		select {
		case <-matchesT.C:
			fetchMatches(db, client)
		case <-seriesT.C:
			fetchLiveSeries(db, client)
		case <-resultsT.C:
			fetchResults(db, client)
			backfillSeries(db, client)
		case <-eventsT.C:
			fetchEvents(db, client)
		case <-stop:
			log.Println("stopping")
			return
		}
	}
}

// fetchMatches updates matches:live + matches:upcoming from one /matches
// request, and records a heartbeat so the TUI can surface a dead/erroring
// fetcher (the live path is the primary freshness signal).
func fetchMatches(db *sql.DB, client *vlr.Client) {
	all, err := client.Matches()
	recordFetch(db, err)
	if err != nil {
		log.Printf("matches: %v", err)
		return
	}
	writeMatches(db, "matches:live", vlr.FilterStatus(all, "live"))
	writeMatches(db, "matches:upcoming", vlr.FilterStatus(all, "upcoming"))
}

// recordFetch writes the live-path heartbeat: the attempt time and the error
// (empty on success) the TUI reads via data.FetchError.
func recordFetch(db *sql.DB, err error) {
	msg := ""
	if err != nil {
		msg = err.Error()
	}
	payload, mErr := json.Marshal(map[string]string{
		"error":      msg,
		"updated_at": time.Now().UTC().Format(time.RFC3339),
	})
	if mErr != nil {
		return
	}
	if wErr := writeKey(db, "meta:fetch", payload); wErr != nil {
		log.Printf("write meta:fetch: %v", wErr)
	}
}

// fetchResults updates matches:completed from /matches/results.
func fetchResults(db *sql.DB, client *vlr.Client) {
	done, err := client.Results()
	if err != nil {
		log.Printf("results: %v", err)
		return
	}
	writeMatches(db, "matches:completed", vlr.FilterStatus(done, "completed"))
}

// fetchEvents updates events:active and each active event's match list.
func fetchEvents(db *sql.DB, client *vlr.Client) {
	events, err := client.Events()
	if err != nil {
		log.Printf("events: %v", err)
		return
	}
	if payload, err := json.Marshal(events); err != nil {
		log.Printf("marshal events: %v", err)
	} else if err := writeKey(db, "events:active", payload); err != nil {
		log.Printf("write events:active: %v", err)
	} else {
		log.Printf("events:active · %d", len(events))
	}
	fetchEventMatches(db, client, events)
}

func maxDur(a, b time.Duration) time.Duration {
	if a > b {
		return a
	}
	return b
}

// orDefault returns v when set (>0), else the fallback. Lets a flag of 0 mean
// "use the derived default".
func orDefault(v, fallback time.Duration) time.Duration {
	if v > 0 {
		return v
	}
	return fallback
}

// seriesBackfillCap bounds how many completed matches we fetch detail for per
// pass, so a cold cache doesn't fan out into hundreds of match-page requests.
const seriesBackfillCap = 15

// fetchLiveSeries refreshes series:{id} for every live match so the broadcast
// match-detail view stays current. Bounded by the handful of concurrent live
// matches.
func fetchLiveSeries(db *sql.DB, client *vlr.Client) {
	ids := matchIDs(db, "matches:live")
	var ok int
	for _, id := range ids {
		if writeSeries(db, client, id) {
			ok++
		}
	}
	if len(ids) > 0 {
		log.Printf("series:live · %d/%d", ok, len(ids))
	}
}

// backfillSeries fetches detail for completed matches we don't have cached yet,
// capped so this stays polite. Each completed match is fetched once and then
// skipped on later passes (its scoreboard no longer changes).
func backfillSeries(db *sql.DB, client *vlr.Client) {
	ids := matchIDs(db, "matches:completed")
	var ok, n int
	for _, id := range ids {
		if n >= seriesBackfillCap {
			break
		}
		if hasKey(db, fmt.Sprintf("series:%d", id)) {
			continue
		}
		n++
		if writeSeries(db, client, id) {
			ok++
		}
	}
	if n > 0 {
		log.Printf("series:backfill · %d/%d", ok, n)
	}
}

// writeSeries scrapes one match page and upserts series:{id}. Reports success.
func writeSeries(db *sql.DB, client *vlr.Client, id int) bool {
	if id == 0 {
		return false
	}
	s, err := client.SeriesDetail(id)
	if err != nil {
		log.Printf("series:%d: %v", id, err)
		return false
	}
	payload, err := json.Marshal(s)
	if err != nil {
		log.Printf("marshal series:%d: %v", id, err)
		return false
	}
	if err := writeKey(db, fmt.Sprintf("series:%d", id), payload); err != nil {
		log.Printf("write series:%d: %v", id, err)
		return false
	}
	return true
}

// matchIDs reads the match_id of every row under a matches:* key.
func matchIDs(db *sql.DB, key string) []int {
	var value string
	if err := db.QueryRow("SELECT value FROM kv WHERE key = ?", key).Scan(&value); err != nil {
		return nil
	}
	var rows []struct {
		MatchID int `json:"match_id"`
	}
	if err := json.Unmarshal([]byte(value), &rows); err != nil {
		return nil
	}
	ids := make([]int, 0, len(rows))
	for _, r := range rows {
		if r.MatchID != 0 {
			ids = append(ids, r.MatchID)
		}
	}
	return ids
}

// hasKey reports whether a kv key already exists.
func hasKey(db *sql.DB, key string) bool {
	var one int
	return db.QueryRow("SELECT 1 FROM kv WHERE key = ?", key).Scan(&one) == nil
}

// fetchEventMatches pre-fetches each active event's match list into
// event:matches:{id}. Bounded by the number of active events (~40), so it's a
// polite amount of work on a slow cadence.
func fetchEventMatches(db *sql.DB, client *vlr.Client, events []vlr.Event) {
	var ok, total int
	for _, e := range events {
		ms, err := client.EventMatches(e.ID)
		if err != nil {
			log.Printf("event:matches:%d: %v", e.ID, err)
			continue
		}
		total += len(ms)
		payload, err := json.Marshal(ms)
		if err != nil {
			continue
		}
		if err := writeKey(db, fmt.Sprintf("event:matches:%d", e.ID), payload); err == nil {
			ok++
		}
	}
	log.Printf("event:matches · %d events, %d matches", ok, total)
}

// writeMatches marshals a match list and upserts it under key.
func writeMatches(db *sql.DB, key string, matches []vlr.Match) {
	payload, err := json.Marshal(matches)
	if err != nil {
		log.Printf("marshal %s: %v", key, err)
		return
	}
	if err := writeKey(db, key, payload); err != nil {
		log.Printf("write %s: %v", key, err)
		return
	}
	log.Printf("%s · %d", key, len(matches))
}

// writeKey upserts one kv row inside a transaction so the TUI (a read-only
// connection) never sees a half-written value.
func writeKey(db *sql.DB, key string, value []byte) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(
		`INSERT INTO kv (key, value, updated_at) VALUES (?, json(?), ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
		key, string(value), time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func openDB(path string) *sql.DB {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		log.Fatalf("cache dir: %v", err)
	}
	db, err := sql.Open("sqlite", "file:"+path+"?_busy_timeout=5000")
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	// WAL lets the fetcher write while many TUI readers read concurrently.
	if _, err := db.Exec(`PRAGMA journal_mode=WAL`); err != nil {
		log.Fatalf("wal: %v", err)
	}
	if _, err := db.Exec(
		`CREATE TABLE IF NOT EXISTS kv (key TEXT PRIMARY KEY, value JSON NOT NULL, updated_at TEXT NOT NULL)`,
	); err != nil {
		log.Fatalf("schema: %v", err)
	}
	return db
}

func resolveDB(flagVal string) string {
	if flagVal != "" {
		return flagVal
	}
	if env := os.Getenv("VALO_TUI_DB"); env != "" {
		return env
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cache", "valo-tui", "cache.db")
}
