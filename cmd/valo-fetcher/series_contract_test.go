package main

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/jashkarangiya/valo-tui/internal/data"
	"github.com/jashkarangiya/valo-tui/internal/vlr"
)

// TestSeriesContract drives the whole fetcher → cache → read-side seam: serve a
// real match page, run the actual writeSeries path, then decode it back through
// data.SeriesDetailFor and assert the broadcast view is fully populated.
func TestSeriesContract(t *testing.T) {
	const matchID = 684615
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "../../internal/vlr/testdata/match.html")
	}))
	defer srv.Close()
	client := vlr.NewWithBase(srv.URL)

	dbPath := filepath.Join(t.TempDir(), "cache.db")
	t.Setenv("VALO_TUI_DB", dbPath)
	db := openDB(dbPath)
	if !writeSeries(db, client, matchID) {
		db.Close()
		t.Fatal("writeSeries reported failure")
	}
	db.Close() // release the writer before the read-only reader opens it

	detail, ok := data.SeriesDetailFor(matchID)
	if !ok {
		t.Fatal("SeriesDetailFor returned not-ok for a freshly written series")
	}
	if detail.Team1.Name == "" || detail.Team2.Name == "" {
		t.Errorf("team names lost across the contract: %+v / %+v", detail.Team1, detail.Team2)
	}
	if detail.Team1.Short == "" || detail.Team2.Short == "" {
		t.Errorf("team shorts lost: %q / %q", detail.Team1.Short, detail.Team2.Short)
	}
	if len(detail.Maps) == 0 {
		t.Fatal("no maps decoded")
	}
	var players, rounds int
	for _, m := range detail.Maps {
		players += len(m.Players)
		rounds += len(m.Rounds)
		if !m.HasScore() {
			t.Errorf("map %s decoded without a score", m.Name)
		}
	}
	if players == 0 || rounds == 0 {
		t.Errorf("scoreboards empty after decode: players=%d rounds=%d", players, rounds)
	}
	if len(detail.Vetoes) == 0 {
		t.Error("veto actions lost across the contract")
	}
	t.Logf("contract ok · %d maps, %d players, %d rounds", len(detail.Maps), players, rounds)
}
