package main

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/jashkarangiya/valo-tui/internal/vlr"
)

func TestMigrateCacheWipesUnstamped(t *testing.T) {
	db := openDB(filepath.Join(t.TempDir(), "cache.db"))
	defer db.Close()

	// Simulate a cache written before versioning existed: rows, but no stamp.
	if err := writeKey(db, "matches:live", []byte(`[{"match_id":1}]`)); err != nil {
		t.Fatal(err)
	}

	migrateCache(db)

	// The stale row is gone, replaced only by the version stamp.
	var live string
	if err := db.QueryRow("SELECT value FROM kv WHERE key='matches:live'").Scan(&live); err == nil {
		t.Errorf("stale matches:live survived migration: %q", live)
	}
	var stamp string
	if err := db.QueryRow("SELECT value FROM kv WHERE key=?", cacheVersionKey).Scan(&stamp); err != nil {
		t.Fatalf("version not stamped after wipe: %v", err)
	}
	if stamp != strconv.Itoa(vlr.CacheVersion) {
		t.Errorf("stamp = %q, want %d", stamp, vlr.CacheVersion)
	}
}

func TestMigrateCachePreservesCurrentVersion(t *testing.T) {
	db := openDB(filepath.Join(t.TempDir(), "cache.db"))
	defer db.Close()

	if err := writeKey(db, cacheVersionKey, []byte(strconv.Itoa(vlr.CacheVersion))); err != nil {
		t.Fatal(err)
	}
	if err := writeKey(db, "matches:live", []byte(`[{"match_id":1}]`)); err != nil {
		t.Fatal(err)
	}

	migrateCache(db)

	var live string
	if err := db.QueryRow("SELECT value FROM kv WHERE key='matches:live'").Scan(&live); err != nil {
		t.Errorf("current-version cache was wiped: %v", err)
	}
}

// TestBackfillEventSeriesCoversCompletedOnly verifies the niche-tournament path:
// every completed match in an event's list gets detail cached, while upcoming
// ones are skipped (caching their empty scoreboard would stick permanently).
func TestBackfillEventSeriesCoversCompletedOnly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "../../internal/vlr/testdata/match.html")
	}))
	defer srv.Close()
	client := vlr.NewWithBase(srv.URL)

	db := openDB(filepath.Join(t.TempDir(), "cache.db"))
	defer db.Close()

	const eventID = 2500
	const doneMatch, upcomingMatch = 684615, 999001
	if err := writeKey(db, "events:active", []byte(`[{"id":2500,"name":"Challengers 2026"}]`)); err != nil {
		t.Fatal(err)
	}
	if err := writeKey(db, "event:matches:2500", []byte(
		`[{"match_id":684615,"status":"completed"},{"match_id":999001,"status":"upcoming"}]`,
	)); err != nil {
		t.Fatal(err)
	}

	backfillEventSeries(db, client)

	if !hasKey(db, "series:684615") {
		t.Errorf("completed event match %d got no detail backfilled", doneMatch)
	}
	if hasKey(db, "series:999001") {
		t.Errorf("upcoming event match %d should not be fetched", upcomingMatch)
	}
}

func TestMigrateCacheStampsEmptyDB(t *testing.T) {
	db := openDB(filepath.Join(t.TempDir(), "cache.db"))
	defer db.Close()

	migrateCache(db)

	var stamp string
	if err := db.QueryRow("SELECT value FROM kv WHERE key=?", cacheVersionKey).Scan(&stamp); err != nil {
		t.Fatalf("empty DB not stamped: %v", err)
	}
	if stamp != strconv.Itoa(vlr.CacheVersion) {
		t.Errorf("stamp = %q, want %d", stamp, vlr.CacheVersion)
	}
}
