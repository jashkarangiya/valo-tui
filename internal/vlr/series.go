package vlr

import (
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// Series is the full per-match detail: header info, vetoes and per-map
// scoreboards. Marshals to the {info, maps} shape data.SeriesDetailFor reads
// from the series:{id} cache key.
type Series struct {
	Info SeriesInfo  `json:"info"`
	Maps []SeriesMap `json:"maps"`
}

// SeriesInfo is the match header: teams, series score, veto, status.
type SeriesInfo struct {
	MatchID    int          `json:"match_id"`
	Event      string       `json:"event"`
	EventPhase string       `json:"event_phase"`
	BestOf     string       `json:"best_of"`
	StatusNote string       `json:"status_note"`
	Remaining  string       `json:"remaining"`
	Teams      []SeriesTeam `json:"teams"`
	Score      []int        `json:"score"`
	MapActions []MapAction  `json:"map_actions"`
}

type SeriesTeam struct {
	Name  string `json:"name"`
	Short string `json:"short"`
}

// MapAction is one veto step: action ∈ {pick, ban, remaining}.
type MapAction struct {
	Action string `json:"action"`
	Team   string `json:"team"`
	Map    string `json:"map"`
}

type SeriesMap struct {
	MapName string         `json:"map_name"`
	Teams   []MapTeam      `json:"teams"`
	Players []SeriesPlayer `json:"players"`
	Rounds  []SeriesRound  `json:"rounds"`
}

type MapTeam struct {
	Short string `json:"short"`
	Score *int   `json:"score"`
}

type SeriesPlayer struct {
	Name      string   `json:"name"`
	Agents    []string `json:"agents"`
	TeamShort string   `json:"team_short"`
	ACS       *int     `json:"acs"`
	K         *int     `json:"k"`
	D         *int     `json:"d"`
	A         *int     `json:"a"`
	FK        *int     `json:"fk"`
	FD        *int     `json:"fd"`
	ADR       *float64 `json:"adr"`
	HSPct     *float64 `json:"hs_pct"`
}

type SeriesRound struct {
	Number          int    `json:"number"`
	WinnerSide      string `json:"winner_side"` // "Attacker" | "Defender"
	WinnerTeamShort string `json:"winner_team_short"`
}

// SeriesDetail scrapes a single match page (/{id}) into the per-map scoreboards
// the match-detail overlay renders.
func (c *Client) SeriesDetail(matchID int) (Series, error) {
	body, err := c.get(fmt.Sprintf("/%d", matchID))
	if err != nil {
		return Series{}, err
	}
	defer body.Close()
	return parseSeries(body, matchID)
}

// statColumn maps the vm-stats overview table's mod-stat columns (after the
// player + agent cells) to the fields we keep. Column order on vlr.gg is
// R · ACS · K · D · A · +/- · KAST · ADR · HS% · FK · FD · +/-.
const (
	colACS = 1
	colK   = 2
	colD   = 3
	colA   = 4
	colADR = 7
	colHS  = 8
	colFK  = 9
	colFD  = 10
)

func parseSeries(r io.Reader, matchID int) (Series, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return Series{}, err
	}

	info := SeriesInfo{MatchID: matchID}

	// Team names (the two header links).
	doc.Find(".match-header-link-name .wf-title-med").EachWithBreak(func(_ int, s *goquery.Selection) bool {
		info.Teams = append(info.Teams, SeriesTeam{Name: norm(s.Text())})
		return len(info.Teams) < 2
	})

	// Series score: the two spoiler spans, in team order.
	doc.Find(".match-header-vs-score .js-spoiler span").Each(func(_ int, s *goquery.Selection) {
		if n, err := strconv.Atoi(norm(s.Text())); err == nil {
			info.Score = append(info.Score, n)
		}
	})

	// The vs-notes are not reliably ordered: a completed match shows
	// ["final", "Bo3"], an upcoming one just ["Bo3"] (or a "18h 0m" countdown),
	// a live one a "live" badge. Classify by content rather than position.
	var noteTexts []string
	doc.Find(".match-header-vs-note").Each(func(_ int, n *goquery.Selection) {
		noteTexts = append(noteTexts, norm(n.Text()))
	})
	info.BestOf, info.StatusNote, info.Remaining = classifyNotes(noteTexts)

	// Event name + series phase.
	ev := doc.Find(".match-header-event").First()
	info.EventPhase = norm(ev.Find(".match-header-event-series").Text())
	full := norm(ev.Text())
	info.Event = strings.TrimSpace(strings.Replace(full, info.EventPhase, "", 1))

	info.MapActions = parseVeto(norm(doc.Find(".match-header-note").First().Text()))

	var maps []SeriesMap
	doc.Find(".vm-stats-game").Each(func(_ int, g *goquery.Selection) {
		id, _ := g.Attr("data-game-id")
		if id == "" || id == "all" { // skip the aggregate "all" tab
			return
		}
		if mp, ok := parseGame(g); ok {
			maps = append(maps, mp)
		}
	})

	// Backfill header team shorts from the first map's scoreboard.
	if len(maps) > 0 && len(maps[0].Teams) == 2 {
		for i := range info.Teams {
			if i < len(maps[0].Teams) {
				info.Teams[i].Short = maps[0].Teams[i].Short
			}
		}
	}

	return Series{Info: info, Maps: maps}, nil
}

// parseGame parses one per-map vm-stats-game block.
func parseGame(g *goquery.Selection) (SeriesMap, bool) {
	header := g.Find(".vm-stats-game-header").First()

	nameSel := header.Find(".map").First().Clone()
	nameSel.Find(".picked").Remove()
	nameSel.Find(".map-duration").Remove()
	name := norm(nameSel.Text())
	if name == "" {
		return SeriesMap{}, false
	}

	var teams []MapTeam
	header.Find(".score").EachWithBreak(func(_ int, s *goquery.Selection) bool {
		teams = append(teams, MapTeam{Score: atoiPtr(s.Text())})
		return len(teams) < 2
	})
	if len(teams) != 2 {
		return SeriesMap{}, false
	}

	// Round momentum + the per-map team shorts (from the rounds header).
	shorts, rounds := parseRounds(g)
	for i := range teams {
		if i < len(shorts) {
			teams[i].Short = shorts[i]
		}
	}

	var players []SeriesPlayer
	g.Find("table.wf-table-inset.mod-overview tbody tr").Each(func(_ int, tr *goquery.Selection) {
		if p, ok := parsePlayer(tr); ok {
			players = append(players, p)
		}
	})

	return SeriesMap{MapName: name, Teams: teams, Players: players, Rounds: rounds}, true
}

// parsePlayer reads one scoreboard row.
func parsePlayer(tr *goquery.Selection) (SeriesPlayer, bool) {
	name := norm(tr.Find("td.mod-player .text-of").First().Text())
	if name == "" {
		return SeriesPlayer{}, false
	}
	p := SeriesPlayer{
		Name:      name,
		TeamShort: norm(tr.Find("td.mod-player .ge-text-light").First().Text()),
	}
	tr.Find("td.mod-agents img").Each(func(_ int, img *goquery.Selection) {
		if alt, ok := img.Attr("alt"); ok && alt != "" {
			p.Agents = append(p.Agents, strings.ToLower(norm(alt)))
		}
	})

	tr.Find("td.mod-stat").Each(func(i int, td *goquery.Selection) {
		v := norm(td.Find(".mod-both").First().Text())
		switch i {
		case colACS:
			p.ACS = atoiPtr(v)
		case colK:
			p.K = atoiPtr(v)
		case colD:
			p.D = atoiPtr(v)
		case colA:
			p.A = atoiPtr(v)
		case colADR:
			p.ADR = floatPtr(v)
		case colHS:
			p.HSPct = floatPtr(strings.TrimSuffix(v, "%"))
		case colFK:
			p.FK = atoiPtr(v)
		case colFD:
			p.FD = atoiPtr(v)
		}
	})
	return p, true
}

// parseRounds reads the vlr-rounds strip: the two team shorts (from the row
// header) and one RoundLine per played round.
func parseRounds(g *goquery.Selection) (shorts []string, rounds []SeriesRound) {
	g.Find(".vlr-rounds-row-col").Each(func(_ int, col *goquery.Selection) {
		// Header column: carries the two team shorts, no round number.
		if teams := col.Find(".team"); teams.Length() == 2 {
			if len(shorts) == 0 {
				teams.Each(func(_ int, t *goquery.Selection) {
					shorts = append(shorts, norm(t.Text()))
				})
			}
			return
		}
		numTxt := norm(col.Find(".rnd-num").First().Text())
		num, err := strconv.Atoi(numTxt)
		if err != nil {
			return
		}
		// The winning side is the .rnd-sq carrying .mod-win; its index (0|1)
		// picks the team, mod-t/mod-ct gives attacker/defender.
		winIdx, side := -1, ""
		col.Find(".rnd-sq").Each(func(i int, sq *goquery.Selection) {
			if !sq.HasClass("mod-win") {
				return
			}
			winIdx = i
			if cls, _ := sq.Attr("class"); strings.Contains(cls, "mod-ct") {
				side = "Defender"
			} else {
				side = "Attacker"
			}
		})
		if winIdx < 0 {
			return
		}
		short := ""
		if winIdx < len(shorts) {
			short = shorts[winIdx]
		}
		rounds = append(rounds, SeriesRound{Number: num, WinnerSide: side, WinnerTeamShort: short})
	})
	return shorts, rounds
}

// parseVeto splits the header note ("LEV ban Breeze; NRG pick Ascent; Pearl
// remains") into structured map actions.
func parseVeto(note string) []MapAction {
	if note == "" {
		return nil
	}
	var out []MapAction
	for _, part := range strings.Split(note, ";") {
		f := strings.Fields(strings.TrimSpace(part))
		switch {
		case len(f) >= 3 && (f[1] == "pick" || f[1] == "ban"):
			out = append(out, MapAction{Action: f[1], Team: f[0], Map: strings.Join(f[2:], " ")})
		case len(f) >= 2 && f[len(f)-1] == "remains":
			out = append(out, MapAction{Action: "remaining", Map: strings.Join(f[:len(f)-1], " ")})
		}
	}
	return out
}

// bestOfRe matches a best-of label ("Bo1" / "Bo3" / "Bo5").
var bestOfRe = regexp.MustCompile(`(?i)^Bo\d+$`)

// classifyNotes sorts the header vs-notes by content into (bestOf, status,
// remaining). Order varies by match state, so we match on what each note is,
// not where it sits: a "BoN" label is the best-of, a final/live word is the
// status, and anything else (a "18h 0m" countdown) is the time remaining.
func classifyNotes(notes []string) (bestOf, status, remaining string) {
	for _, t := range notes {
		switch {
		case bestOfRe.MatchString(t):
			bestOf = t
		case isStatusWord(t):
			status = t
		case remaining == "" && t != "":
			remaining = t
		}
	}
	return bestOf, status, remaining
}

// isStatusWord reports whether a note is a match-state word rather than a
// countdown or best-of label.
func isStatusWord(t string) bool {
	switch l := strings.ToLower(t); {
	case strings.Contains(l, "final"),
		strings.Contains(l, "live"),
		strings.Contains(l, "complet"),
		strings.Contains(l, "forfeit"),
		strings.Contains(l, "cancel"):
		return true
	}
	return false
}

func atoiPtr(s string) *int {
	if n, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
		return &n
	}
	return nil
}

func floatPtr(s string) *float64 {
	if v, err := strconv.ParseFloat(strings.TrimSpace(s), 64); err == nil {
		return &v
	}
	return nil
}
