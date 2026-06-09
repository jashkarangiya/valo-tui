package data

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite" // pure-Go SQLite driver, no CGO
)

// regionKeywords classify an event/match into one of the four leagues.
var regionKeywords = map[string][]string{
	"Americas": {"americas", "north america", "latam", "brazil", "united states", "n.a"},
	"EMEA":     {"emea", "europe", "middle east", "türkiye", "turkey", "mena"},
	"Pacific":  {"pacific", "korea", "japan", "asia", "oceania", "south asia"},
	"China":    {"china", "chinese", "中国", " cn "},
}

var international = []string{
	"masters", "champions", "valorant champions", "vct international",
	"esports world cup", "ewc",
}

// ClassifyRegion buckets an event/match into a league, or "" if none match.
func ClassifyRegion(parts ...string) string {
	text := strings.ToLower(strings.Join(parts, " "))
	for _, region := range REGIONS {
		for _, kw := range regionKeywords[region] {
			if strings.Contains(text, kw) {
				return region
			}
		}
	}
	return ""
}

// IsInternational reports whether an event name is an international (non-regional) one.
func IsInternational(name string) bool {
	text := strings.ToLower(name)
	if strings.Contains(text, "challengers") { // regional tier, not international
		return false
	}
	for _, kw := range international {
		if strings.Contains(text, kw) {
			return true
		}
	}
	return false
}

// dbPath resolves the cache DB path, matching config.db_path() in Python:
// $VALO_TUI_DB, else ~/.cache/valo-tui/cache.db.
func dbPath() string {
	if raw := os.Getenv("VALO_TUI_DB"); raw != "" {
		return raw
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cache", "valo-tui", "cache.db")
}

// open returns a read-only connection to the kv cache. Read-only so the TUI
// never contends with the worker's writes.
func open() (*sql.DB, error) {
	dsn := "file:" + dbPath() + "?mode=ro&_busy_timeout=5000"
	return sql.Open("sqlite", dsn)
}

// getRaw returns the JSON value and updated_at for a kv key.
func getRaw(key string) (json.RawMessage, string) {
	db, err := open()
	if err != nil {
		return nil, ""
	}
	defer db.Close()
	var value, updatedAt string
	err = db.QueryRow("SELECT value, updated_at FROM kv WHERE key = ?", key).
		Scan(&value, &updatedAt)
	if err != nil {
		return nil, ""
	}
	return json.RawMessage(value), updatedAt
}

// getList decodes a kv value known to be a JSON array of objects.
func getList(key string) []map[string]any {
	raw, _ := getRaw(key)
	if raw == nil {
		return nil
	}
	var out []map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil
	}
	return out
}

func cards(key string) []MatchCard {
	rows := getList(key)
	out := make([]MatchCard, 0, len(rows))
	for _, r := range rows {
		out = append(out, matchFromRaw(r))
	}
	return out
}

// LiveMatches / UpcomingMatches / CompletedMatches read the worker-populated
// polling lists.
func LiveMatches() []MatchCard      { return cards("matches:live") }
func UpcomingMatches() []MatchCard  { return cards("matches:upcoming") }
func CompletedMatches() []MatchCard { return cards("matches:completed") }

// ActiveEvents reads the currently-tracked tournaments.
func ActiveEvents() []EventCard {
	rows := getList("events:active")
	out := make([]EventCard, 0, len(rows))
	for _, r := range rows {
		out = append(out, eventFromRaw(r))
	}
	return out
}

// EventByID finds one active event, or false.
func EventByID(id int) (EventCard, bool) {
	for _, e := range ActiveEvents() {
		if e.ID == id {
			return e, true
		}
	}
	return EventCard{}, false
}

// lastUpdatedTime is the most recent write across the polling keys (zero if none).
func lastUpdatedTime() time.Time {
	var latest time.Time
	for _, key := range []string{"matches:live", "matches:upcoming", "events:active"} {
		_, ts := getRaw(key)
		if ts == "" {
			continue
		}
		if t, err := time.Parse(time.RFC3339, ts); err == nil && t.After(latest) {
			latest = t
		}
	}
	return latest
}

// LastUpdated is the most recent write across the polling keys, as HH:MM:SS.
func LastUpdated() string {
	latest := lastUpdatedTime()
	if latest.IsZero() {
		return ""
	}
	return latest.UTC().Format("15:04:05")
}

// staleAfter is how old the newest live write may get before we treat the
// cache as stale — a generous multiple of the fastest fetch cadence, so a
// briefly-slow fetcher doesn't trip a false alarm but a dead one does.
const staleAfter = 3 * time.Minute

// FreshnessState returns the freshness label plus whether the cache is stale
// (newest live write older than staleAfter), i.e. the fetcher is likely down.
func FreshnessState() (label string, stale bool) {
	latest := lastUpdatedTime()
	if latest.IsZero() {
		return "", false
	}
	return Freshness(), time.Since(latest) > staleAfter
}

// FetchError returns the fetcher's last-recorded error for the live polling
// path (empty when the last run succeeded), and whether it is recent enough to
// still be worth showing.
func FetchError() (msg string, recent bool) {
	obj := getObject("meta:fetch")
	if obj == nil {
		return "", false
	}
	msg = s(obj["error"])
	if msg == "" {
		return "", false
	}
	if ts, err := time.Parse(time.RFC3339, s(obj["updated_at"])); err == nil {
		recent = time.Since(ts) < 10*time.Minute
	}
	return msg, recent
}

// Freshness is the age of the newest cached data as a short relative string
// ("42s ago" / "3m ago" / "2h ago"), or "" when the cache is empty.
func Freshness() string {
	latest := lastUpdatedTime()
	if latest.IsZero() {
		return ""
	}
	d := time.Since(latest)
	switch {
	case d < 0:
		return "just now"
	case d < time.Minute:
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	default:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
}

// getObject decodes a kv value known to be a JSON object.
func getObject(key string) map[string]any {
	raw, _ := getRaw(key)
	if raw == nil {
		return nil
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil
	}
	return out
}

// EventMatches returns all cached matches for an event as raw dicts. Read-only:
// unlike v1 there is no network read-through yet (the Go fetcher is pending),
// so a cache miss yields an empty list.
func EventMatches(eventID int) []map[string]any {
	return getList(fmt.Sprintf("event:matches:%d", eventID))
}

// EventMatchCards returns an event's matches as typed cards for the
// results/fixtures/standings sub-pages.
func EventMatchCards(eventID int, eventName string) []MatchCard {
	rows := EventMatches(eventID)
	out := make([]MatchCard, 0, len(rows))
	for _, r := range rows {
		out = append(out, matchFromEventRaw(r, eventName))
	}
	return out
}

// SeriesDetailFor returns per-map scoreboards + vetoes for a match from cache.
func SeriesDetailFor(matchID int) (SeriesDetail, bool) {
	obj := getObject(fmt.Sprintf("series:%d", matchID))
	if obj == nil {
		return SeriesDetail{}, false
	}
	info := asMap(obj["info"])
	if info == nil {
		return SeriesDetail{}, false
	}
	return seriesFromRaw(info, asList(obj["maps"])), true
}

// BracketFor reconstructs an event's double-elim bracket from its matches.
func BracketFor(eventID int) Bracket {
	return BuildBracket(EventMatches(eventID))
}

// RegionSlots holds one league's three time-buckets for the global dashboard.
type RegionSlots struct {
	Live   []MatchCard
	Next   []MatchCard
	Recent []MatchCard
}

// GlobalLive builds the (regions, international) structure for the dashboard,
// mirroring cache.global_live().
func GlobalLive() (map[string]*RegionSlots, []MatchCard) {
	buckets := make(map[string]*RegionSlots, len(REGIONS))
	for _, r := range REGIONS {
		buckets[r] = &RegionSlots{}
	}
	var intl []MatchCard

	groups := []struct {
		slot    string
		matches []MatchCard
	}{
		{"live", LiveMatches()},
		{"next", UpcomingMatches()},
		{"recent", CompletedMatches()},
	}
	for _, g := range groups {
		for _, m := range g.matches {
			if IsInternational(m.Event) {
				if g.slot != "recent" || len(intl) < 6 {
					intl = append(intl, m)
				}
				continue
			}
			region := ClassifyRegion(m.Event, m.Phase)
			if region == "" {
				continue
			}
			b := buckets[region]
			switch g.slot {
			case "live":
				b.Live = append(b.Live, m)
			case "next":
				b.Next = append(b.Next, m)
			case "recent":
				b.Recent = append(b.Recent, m)
			}
		}
	}
	return buckets, intl
}
