package data

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Bracket reconstruction from a flat list of phase-tagged event matches.
// Mirrors data/bracket.py: vlr.gg exposes no explicit bracket structure, only
// matches tagged with a phase like "Upper Quarterfinals" / "Lower Final".

var roundNumRe = regexp.MustCompile(`round\s*(\d+)`)

var roundRank = []struct {
	kw   string
	rank int
}{
	{"quarter", 1}, {"semi", 2},
	{"round 1", 1}, {"round 2", 2}, {"round 3", 3}, {"round 4", 4}, {"round 5", 5},
	{"final", 9},
}

func roundRankOf(phase string) int {
	p := strings.ToLower(phase)
	if m := roundNumRe.FindStringSubmatch(p); m != nil {
		n, _ := strconv.Atoi(m[1])
		return n
	}
	for _, r := range roundRank {
		if strings.Contains(p, r.kw) {
			return r.rank
		}
	}
	return 0
}

// sectionOf returns "upper" | "lower" | "final", or "" if not a bracket phase.
func sectionOf(phase string) string {
	p := strings.ToLower(phase)
	switch {
	case strings.Contains(p, "grand final"):
		return "final"
	case strings.Contains(p, "upper"):
		return "upper"
	case strings.Contains(p, "lower"):
		return "lower"
	}
	return ""
}

// BracketSlot is one side of a bracket match.
type BracketSlot struct {
	Name   string
	Score  *int
	Winner bool
}

// BracketMatch is a single node in the tree.
type BracketMatch struct {
	MatchID int
	Top     BracketSlot
	Bottom  BracketSlot
	Status  string
}

func (m BracketMatch) WinnerName() string {
	if m.Top.Winner {
		return m.Top.Name
	}
	if m.Bottom.Winner {
		return m.Bottom.Name
	}
	return ""
}

// BracketColumn is one round (left-to-right) within a section.
type BracketColumn struct {
	Title   string
	Matches []BracketMatch
}

// BracketSection groups columns into Upper / Lower / Final.
type BracketSection struct {
	Name    string
	Columns []BracketColumn
}

// Bracket is the whole reconstructed tree.
type Bracket struct {
	Sections []BracketSection
}

func (b Bracket) HasData() bool {
	for _, s := range b.Sections {
		for _, c := range s.Columns {
			if len(c.Matches) > 0 {
				return true
			}
		}
	}
	return false
}

func slotOf(team map[string]any) BracketSlot {
	name := s(team["name"])
	if name == "" {
		name = "TBD"
	}
	winner, _ := team["is_winner"].(bool)
	return BracketSlot{Name: name, Score: i(team["score"]), Winner: winner}
}

func toBracketMatch(raw map[string]any) (BracketMatch, bool) {
	teams := asList(raw["teams"])
	if len(teams) < 2 {
		return BracketMatch{}, false
	}
	return BracketMatch{
		MatchID: deref(i(raw["match_id"])),
		Top:     slotOf(asMap(teams[0])),
		Bottom:  slotOf(asMap(teams[1])),
		Status:  s(raw["status"]),
	}, true
}

var bracketTitles = map[string]string{
	"upper": "Upper Bracket", "lower": "Lower Bracket", "final": "Grand Final",
}
var bracketOrder = []string{"upper", "lower", "final"}

// BuildBracket groups/orders bracket matches into sections of columns.
func BuildBracket(rawMatches []map[string]any) Bracket {
	// section -> rank -> [(phase, raw)]
	type entry struct {
		phase string
		raw   map[string]any
	}
	buckets := map[string]map[int][]entry{}
	for _, raw := range rawMatches {
		phase := s(raw["phase"])
		section := sectionOf(phase)
		if section == "" {
			continue
		}
		rank := roundRankOf(phase)
		if buckets[section] == nil {
			buckets[section] = map[int][]entry{}
		}
		buckets[section][rank] = append(buckets[section][rank], entry{phase, raw})
	}

	var bracket Bracket
	for _, section := range bracketOrder {
		ranks := buckets[section]
		if ranks == nil {
			continue
		}
		sec := BracketSection{Name: bracketTitles[section]}
		keys := make([]int, 0, len(ranks))
		for k := range ranks {
			keys = append(keys, k)
		}
		sort.Ints(keys)
		for _, rank := range keys {
			entries := ranks[rank]
			col := BracketColumn{Title: entries[0].phase}
			for _, e := range entries {
				if m, ok := toBracketMatch(e.raw); ok {
					col.Matches = append(col.Matches, m)
				}
			}
			if len(col.Matches) > 0 {
				sec.Columns = append(sec.Columns, col)
			}
		}
		if len(sec.Columns) > 0 {
			bracket.Sections = append(bracket.Sections, sec)
		}
	}
	return bracket
}
