package data

import "strings"

// PlayerLine is one row of a map scoreboard. Mirrors models.PlayerLine.
type PlayerLine struct {
	Name      string
	Agents    []string
	ACS       *int
	K, D, A   *int
	ADR       *float64
	HSPct     *float64
	FK, FD    *int
	TeamShort string
}

func playerFromRaw(d map[string]any) PlayerLine {
	name := s(d["name"])
	if name == "" {
		name = "?"
	}
	return PlayerLine{
		Name:      name,
		Agents:    strList(d["agents"]),
		ACS:       i(d["acs"]),
		K:         i(d["k"]),
		D:         i(d["d"]),
		A:         i(d["a"]),
		ADR:       f(d["adr"]),
		HSPct:     f(d["hs_pct"]),
		FK:        i(d["fk"]),
		FD:        i(d["fd"]),
		TeamShort: s(d["team_short"]),
	}
}

// Agent returns the player's first agent, or "" if none.
func (p PlayerLine) Agent() string {
	if len(p.Agents) > 0 {
		return p.Agents[0]
	}
	return ""
}

// RoundLine is one round's outcome. Mirrors models.RoundLine.
type RoundLine struct {
	Number      int
	Side        string // "Attacker" | "Defender"
	WinnerShort string
}

func (r RoundLine) IsAttack() bool {
	return strings.HasPrefix(strings.ToLower(r.Side), "attack")
}

func roundFromRaw(d map[string]any) RoundLine {
	return RoundLine{
		Number:      deref(i(d["number"])),
		Side:        s(d["winner_side"]),
		WinnerShort: s(d["winner_team_short"]),
	}
}

// MapScore is a single map's scoreboard + round momentum. Mirrors models.MapScore.
type MapScore struct {
	Name       string
	Players    []PlayerLine
	Team1Short string
	Team1Score *int
	Team2Short string
	Team2Score *int
	Rounds     []RoundLine
}

func (m MapScore) IsAggregate() bool { return strings.ToLower(m.Name) == "all" }

// HasScore reports a non-zero combined score (upcoming maps come back 0–0).
func (m MapScore) HasScore() bool {
	return deref(m.Team1Score)+deref(m.Team2Score) > 0
}

// State is "completed" | "live" | "pending" for rendering decisions.
func (m MapScore) State() string {
	if len(m.Rounds) > 0 {
		return "completed"
	}
	if m.HasScore() {
		return "live"
	}
	return "pending"
}

func mapFromRaw(d map[string]any) MapScore {
	teams := asList(d["teams"])
	var t1, t2 map[string]any
	if len(teams) > 0 {
		t1 = asMap(teams[0])
	}
	if len(teams) > 1 {
		t2 = asMap(teams[1])
	}
	name := s(d["map_name"])
	if name == "" {
		name = "?"
	}
	players := []PlayerLine{}
	for _, p := range asList(d["players"]) {
		players = append(players, playerFromRaw(asMap(p)))
	}
	rounds := []RoundLine{}
	for _, r := range asList(d["rounds"]) {
		rounds = append(rounds, roundFromRaw(asMap(r)))
	}
	t1short := s(t1["short"])
	if t1short == "" {
		t1short = s(t1["name"])
	}
	t2short := s(t2["short"])
	if t2short == "" {
		t2short = s(t2["name"])
	}
	return MapScore{
		Name:       name,
		Players:    players,
		Team1Short: t1short,
		Team1Score: i(t1["score"]),
		Team2Short: t2short,
		Team2Score: i(t2["score"]),
		Rounds:     rounds,
	}
}

// Veto is one map-veto action.
type Veto struct {
	Action string // pick | ban | remaining
	Team   string
	Map    string
}

// SeriesDetail is the full per-match view: vetoes + per-map scoreboards.
type SeriesDetail struct {
	MatchID    int
	Team1      TeamSide
	Team2      TeamSide
	Event      string
	Phase      string
	BestOf     string
	StatusNote string
	Remaining  string
	Patch      string
	Vetoes     []Veto
	Maps       []MapScore
}

func (d SeriesDetail) IsLive() bool {
	return strings.Contains(strings.ToLower(d.StatusNote), "live")
}

func (d SeriesDetail) IsCompleted() bool {
	if d.IsLive() {
		return false
	}
	return deref(d.Team1.Score) > 0 || deref(d.Team2.Score) > 0
}

// PickLabel returns which team picked a given map (or "decider").
func (d SeriesDetail) PickLabel(mapName string) string {
	for _, v := range d.Vetoes {
		if !strings.EqualFold(v.Map, mapName) {
			continue
		}
		switch strings.ToLower(v.Action) {
		case "pick":
			return v.Team + " pick"
		case "remaining", "decider":
			return "decider"
		}
	}
	return ""
}

func seriesFromRaw(info map[string]any, maps []any) SeriesDetail {
	teams := asList(info["teams"])
	var t1raw, t2raw map[string]any
	if len(teams) > 0 {
		t1raw = asMap(teams[0])
	}
	if len(teams) > 1 {
		t2raw = asMap(teams[1])
	}
	t1 := teamFromRaw(t1raw)
	t2 := teamFromRaw(t2raw)
	score := asList(info["score"])
	if len(score) > 0 {
		t1.Score = i(score[0])
	}
	if len(score) > 1 {
		t2.Score = i(score[1])
	}

	vetoes := []Veto{}
	for _, v := range asList(info["map_actions"]) {
		m := asMap(v)
		vetoes = append(vetoes, Veto{Action: s(m["action"]), Team: s(m["team"]), Map: s(m["map"])})
	}

	mapScores := []MapScore{}
	for _, m := range maps {
		mapScores = append(mapScores, mapFromRaw(asMap(m)))
	}

	return SeriesDetail{
		MatchID:    deref(i(info["match_id"])),
		Team1:      t1,
		Team2:      t2,
		Event:      s(info["event"]),
		Phase:      s(info["event_phase"]),
		BestOf:     s(info["best_of"]),
		StatusNote: s(info["status_note"]),
		Remaining:  s(info["remaining"]),
		Patch:      s(info["patch"]),
		Vetoes:     vetoes,
		Maps:       mapScores,
	}
}
