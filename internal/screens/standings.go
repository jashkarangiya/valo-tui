package screens

import (
	"fmt"

	tea "charm.land/bubbletea/v2"

	"github.com/jashkarangiya/valo-tui/internal/data"
	"github.com/jashkarangiya/valo-tui/internal/widgets"
)

// Standings is the [t] event sub-page: a W-L / map-record table derived from
// the event's completed matches.
type Standings struct {
	w, h   int
	table  widgets.Table
	hasEvt bool
}

func NewStandings(w, h int) Standings {
	return Standings{w: w, h: h, table: widgets.NewTable(
		widgets.Column{Title: "#", Width: 4},
		widgets.Column{Title: "team", Width: 22},
		widgets.Column{Title: "W-L", Width: 7},
		widgets.Column{Title: "maps", Width: 8},
		widgets.Column{Title: "diff", Width: 6},
	)}
}

func (s *Standings) SetSize(w, h int) {
	s.w, s.h = w, h
	s.table.SetHeight(h - 4)
}

func (s *Standings) Load(eventID int, eventName string) {
	s.hasEvt = eventID != 0
	if !s.hasEvt {
		s.table.SetRows(nil)
		return
	}
	records := data.TeamRecords(data.EventMatchCards(eventID, eventName))
	rows := make([]widgets.Row, 0, len(records))
	for i, r := range records {
		if i >= 24 {
			break
		}
		diffStyle := mutedSt
		if r.MapDiff() > 0 {
			diffStyle = accentSt
		}
		rows = append(rows, widgets.Row{
			Key: r.Team,
			Cells: []widgets.Cell{
				{Text: fmt.Sprint(i + 1), Style: mutedSt},
				{Text: r.Team, Style: textSt},
				{Text: fmt.Sprintf("%d-%d", r.Wins, r.Losses), Style: textSt},
				{Text: fmt.Sprintf("%d-%d", r.MapsWon, r.MapsLost), Style: mutedSt},
				{Text: fmt.Sprintf("%+d", r.MapDiff()), Style: diffStyle},
			},
		})
	}
	s.table.SetRows(rows)
}

func (s *Standings) Focus()                           { s.table.Focus() }
func (s *Standings) Blur()                            { s.table.Blur() }
func (s *Standings) ClickVisual(i int) (string, bool) { return s.table.ClickVisual(i) }

// SelectedTeam returns the team name of the cursor row (for Enter → roster).
func (s Standings) SelectedTeam() string { return s.table.SelectedKey() }

func (s Standings) Update(msg tea.Msg) (Standings, tea.Cmd) {
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

func (s Standings) View() string {
	header := title("standings") + "\n" + hint("derived from this event's completed matches") + "\n\n"
	if !s.hasEvt {
		return header + muted("select an event first")
	}
	if s.table.Len() == 0 {
		return header + muted("no completed matches for this event yet")
	}
	return header + s.table.View()
}
