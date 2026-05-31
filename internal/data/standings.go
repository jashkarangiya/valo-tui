package data

import "sort"

// TeamRecord is a team's W-L / map record. Mirrors standings.TeamRecord.
type TeamRecord struct {
	Team     string
	Played   int
	Wins     int
	Losses   int
	MapsWon  int
	MapsLost int
}

func (r TeamRecord) MapDiff() int { return r.MapsWon - r.MapsLost }

func (r TeamRecord) Pct() float64 {
	if r.Played == 0 {
		return 0
	}
	return float64(r.Wins) / float64(r.Played) * 100
}

// TeamRecords builds a sorted standings table from completed matches, mirroring
// standings.team_records. A match's per-team score is the series score (maps
// won), giving both a W-L record and a map differential.
func TeamRecords(matches []MatchCard) []TeamRecord {
	table := map[string]*TeamRecord{}
	order := []string{} // preserve first-seen for stable sorting of ties
	get := func(name string) *TeamRecord {
		r, ok := table[name]
		if !ok {
			r = &TeamRecord{Team: name}
			table[name] = r
			order = append(order, name)
		}
		return r
	}

	for _, m := range matches {
		if m.Status != "completed" {
			continue
		}
		if m.Team1.Score == nil || m.Team2.Score == nil {
			continue
		}
		s1, s2 := *m.Team1.Score, *m.Team2.Score
		if s1 == s2 {
			continue
		}
		if m.Team1.Name == "TBD" || m.Team1.Name == "" || m.Team2.Name == "TBD" || m.Team2.Name == "" {
			continue
		}
		r1, r2 := get(m.Team1.Name), get(m.Team2.Name)
		r1.Played++
		r2.Played++
		r1.MapsWon += s1
		r1.MapsLost += s2
		r2.MapsWon += s2
		r2.MapsLost += s1
		if s1 > s2 {
			r1.Wins++
			r2.Losses++
		} else {
			r2.Wins++
			r1.Losses++
		}
	}

	out := make([]TeamRecord, 0, len(order))
	for _, name := range order {
		out = append(out, *table[name])
	}
	sort.SliceStable(out, func(a, b int) bool {
		ra, rb := out[a], out[b]
		if ra.Wins != rb.Wins {
			return ra.Wins > rb.Wins
		}
		if ra.MapDiff() != rb.MapDiff() {
			return ra.MapDiff() > rb.MapDiff()
		}
		return ra.Pct() > rb.Pct()
	})
	return out
}
