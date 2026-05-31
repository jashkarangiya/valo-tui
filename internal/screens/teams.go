package screens

import (
	"fmt"
	"sort"

	tea "charm.land/bubbletea/v2"

	"github.com/jashkarangiya/valo-tui/internal/data"
	"github.com/jashkarangiya/valo-tui/internal/widgets"
)

// Teams is the [m] event sub-page: rosters in the event with their running
// series record, derived from the event's own matches.
type Teams struct {
	w, h   int
	table  widgets.Table
	hasEvt bool
}

func NewTeams(w, h int) Teams {
	return Teams{w: w, h: h, table: widgets.NewTable(
		widgets.Column{Title: "team", Width: 24},
		widgets.Column{Title: "W-L", Width: 8},
		widgets.Column{Title: "maps", Width: 8},
		widgets.Column{Title: "diff", Width: 6},
	)}
}

func (s *Teams) SetSize(w, h int) {
	s.w, s.h = w, h
	s.table.SetHeight(h - 4)
}

func (s *Teams) Load(eventID int, eventName string) {
	s.hasEvt = eventID != 0
	if !s.hasEvt {
		s.table.SetRows(nil)
		return
	}
	matches := data.EventMatchCards(eventID, eventName)

	// Every team that appears, even without a decided match yet.
	var names []string
	seen := map[string]bool{}
	for _, m := range matches {
		for _, name := range []string{m.Team1.Name, m.Team2.Name} {
			if name != "" && name != "TBD" && !seen[name] {
				seen[name] = true
				names = append(names, name)
			}
		}
	}

	records := map[string]data.TeamRecord{}
	var ranked []string
	for _, r := range data.TeamRecords(matches) {
		records[r.Team] = r
		ranked = append(ranked, r.Team)
	}
	var rest []string
	for _, n := range names {
		if _, ok := records[n]; !ok {
			rest = append(rest, n)
		}
	}
	sort.Strings(rest)

	rows := []widgets.Row{}
	for _, name := range append(ranked, rest...) {
		wl, maps, diff, diffStyle := "0-0", "0-0", "—", mutedSt
		if r, ok := records[name]; ok {
			wl = fmt.Sprintf("%d-%d", r.Wins, r.Losses)
			maps = fmt.Sprintf("%d-%d", r.MapsWon, r.MapsLost)
			diff = fmt.Sprintf("%+d", r.MapDiff())
			if r.MapDiff() > 0 {
				diffStyle = accentSt
			}
		}
		rows = append(rows, widgets.Row{Cells: []widgets.Cell{
			{Text: name, Style: textSt},
			{Text: wl, Style: textSt},
			{Text: maps, Style: mutedSt},
			{Text: diff, Style: diffStyle},
		}})
	}
	s.table.SetRows(rows)
}

func (s *Teams) Focus()                           { s.table.Focus() }
func (s *Teams) Blur()                            { s.table.Blur() }
func (s *Teams) ClickVisual(i int) (string, bool) { return s.table.ClickVisual(i) }

func (s Teams) Update(msg tea.Msg) (Teams, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok {
		switch k.String() {
		case "j", "down":
			s.table.MoveCursor(1)
		case "k", "up":
			s.table.MoveCursor(-1)
		}
	}
	return s, nil
}

func (s Teams) View() string {
	header := title("teams") + "\n" + hint("rosters in this event") + "\n\n"
	if !s.hasEvt {
		return header + muted("select an event first")
	}
	if s.table.Len() == 0 {
		return header + muted("no teams cached for this event")
	}
	return header + s.table.View()
}
