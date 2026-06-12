package screens

import (
	"fmt"
	"sort"

	tea "charm.land/bubbletea/v2"

	"github.com/jashkarangiya/valo-tui/internal/data"
	"github.com/jashkarangiya/valo-tui/internal/widgets"
)

var resultOrder = map[string]int{"live": 0, "completed": 1}

// Results is the [r] event sub-page: completed & live series. Enter drills into
// the per-map scoreboards.
type Results struct {
	w, h   int
	table  widgets.Table
	hasEvt bool
}

func NewResults(w, h int) Results {
	return Results{w: w, h: h, table: widgets.NewTable(
		widgets.Column{Title: "status", Width: 6},
		widgets.Column{Title: "team 1", Width: 18},
		widgets.Column{Title: "score", Width: 5},
		widgets.Column{Title: "team 2", Width: 18},
		widgets.Column{Title: "phase", Width: 15},
		widgets.Column{Title: "when", Width: 12},
	)}
}

func (s *Results) SetSize(w, h int) {
	s.w, s.h = w, h
	s.table.SetHeight(h - 4)
}

func (s *Results) Load(eventID int, eventName string) {
	s.hasEvt = eventID != 0
	if !s.hasEvt {
		s.table.SetRows(nil)
		return
	}
	matches := []data.MatchCard{}
	for _, m := range data.EventMatchCards(eventID, eventName) {
		if _, ok := resultOrder[m.Status]; ok {
			matches = append(matches, m)
		}
	}
	sort.SliceStable(matches, func(a, b int) bool {
		return resultOrder[matches[a].Status] < resultOrder[matches[b].Status]
	})
	rows := make([]widgets.Row, 0, len(matches))
	for _, m := range matches {
		status := widgets.Cell{Text: "✓ done", Style: mutedSt}
		scoreStyle := mutedSt
		if m.IsLive() {
			status = widgets.Cell{Text: "● live", Style: liveSt}
			scoreStyle = liveSt
		}
		s1, s2 := "–", "–"
		if m.Team1.Score != nil {
			s1 = fmt.Sprint(*m.Team1.Score)
		}
		if m.Team2.Score != nil {
			s2 = fmt.Sprint(*m.Team2.Score)
		}
		when := m.Time
		if when == "" {
			when = m.Date
		}
		// Dim the losing side of a completed match so the winner reads at a glance.
		t1Style, t2Style := textSt, textSt
		if m.Status == "completed" {
			if a, b := derefOr0(m.Team1.Score), derefOr0(m.Team2.Score); a > b {
				t2Style = mutedSt
			} else if b > a {
				t1Style = mutedSt
			}
		}
		rows = append(rows, widgets.Row{
			Key: fmt.Sprint(m.MatchID),
			Cells: []widgets.Cell{
				status,
				{Text: m.Team1.Name, Style: t1Style},
				{Text: s1 + "–" + s2, Style: scoreStyle},
				{Text: m.Team2.Name, Style: t2Style},
				{Text: m.Phase, Style: mutedSt},
				{Text: when, Style: mutedSt},
			},
		})
	}
	s.table.SetRows(rows)
}

func (s *Results) Focus()                           { s.table.Focus() }
func (s *Results) Blur()                            { s.table.Blur() }
func (s Results) Selected() string                  { return s.table.SelectedKey() }
func (s *Results) ClickVisual(i int) (string, bool) { return s.table.ClickVisual(i) }
func (s *Results) MoveCursor(d int)                 { s.table.MoveCursor(d) }

func (s Results) Update(msg tea.Msg) (Results, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok {
		tableMove(&s.table, k.String())
	}
	return s, nil
}

func (s Results) View() string {
	header := title("results") + "\n" + hint("j/k move · enter → scoreboards") + "\n\n"
	if !s.hasEvt {
		return header + muted("select an event first")
	}
	if s.table.Len() == 0 {
		return header + muted("no completed matches yet")
	}
	return header + s.table.View()
}
