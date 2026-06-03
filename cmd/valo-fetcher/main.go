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
	interval := flag.Duration("interval", 30*time.Second, "refresh interval for --watch")
	dbFlag := flag.String("db", "", "cache db path (default $VALO_TUI_DB or ~/.cache/valo-tui/cache.db)")
	flag.Parse()

	db := openDB(resolveDB(*dbFlag))
	defer db.Close()
	client := vlr.New()

	if !*watch {
		fetchOnce(db, client)
		return
	}

	log.Printf("watching · interval %s", *interval)
	fetchOnce(db, client)
	ticker := time.NewTicker(*interval)
	defer ticker.Stop()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	for {
		select {
		case <-ticker.C:
			fetchOnce(db, client)
		case <-stop:
			log.Println("stopping")
			return
		}
	}
}

// fetchOnce runs one full scrape cycle. Each step logs and continues on error,
// so a single failing endpoint never aborts the rest (or kills --watch).
func fetchOnce(db *sql.DB, client *vlr.Client) {
	// /matches carries both live and upcoming — one request, two keys.
	if all, err := client.Matches(); err != nil {
		log.Printf("matches: %v", err)
	} else {
		writeMatches(db, "matches:live", vlr.FilterStatus(all, "live"))
		writeMatches(db, "matches:upcoming", vlr.FilterStatus(all, "upcoming"))
	}

	// /matches/results carries completed matches.
	if done, err := client.Results(); err != nil {
		log.Printf("results: %v", err)
	} else {
		writeMatches(db, "matches:completed", vlr.FilterStatus(done, "completed"))
	}

	// /events → the active tournaments list, then each event's matches so the
	// drill-down screens (results/standings/bracket/teams) have data.
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
