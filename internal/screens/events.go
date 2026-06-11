package screens

import (
	"fmt"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/jashkarangiya/valo-tui/internal/data"
	"github.com/jashkarangiya/valo-tui/internal/widgets"
)

var eventOrder = map[string]int{"ongoing": 0, "upcoming": 1, "completed": 2}

// Events is the [e] hub: browse tournaments and pick one to focus the app on.
type Events struct {
	w, h  int
	table widgets.Table
}

func NewEvents(w, h int) Events {
	e := Events{w: w, h: h, table: widgets.NewTable(
		widgets.Column{Title: "status", Width: 8},
		widgets.Column{Title: "event", Width: 40},
		widgets.Column{Title: "region", Width: 12},
		widgets.Column{Title: "when", Width: 20},
	)}
	e.Load()
	return e
}

func (s *Events) SetSize(w, h int) {
	s.w, s.h = w, h
	s.table.SetHeight(h - 4)
}

func (s *Events) Load() {
	events := data.ActiveEvents()
	sort.SliceStable(events, func(a, b int) bool {
		return eventOrder[strings.ToLower(events[a].Status)] < eventOrder[strings.ToLower(events[b].Status)]
	})
	rows := make([]widgets.Row, 0, len(events))
	for _, e := range events {
		when := ""
		switch {
		case e.Start != "" && e.End != "":
			when = e.Start + " – " + e.End
		case e.Start != "":
			when = e.Start
		}
		region := e.Region
		regionStyle := accentSt
		if region == "" {
			region = "—"
			regionStyle = mutedSt
		}
		rows = append(rows, widgets.Row{
			Key: fmt.Sprint(e.ID),
			Cells: []widgets.Cell{
				statusCell(e.Status),
				{Text: e.Name, Style: textSt},
				{Text: region, Style: regionStyle},
				{Text: when, Style: mutedSt},
			},
		})
	}
	s.table.SetRows(rows)
}

func statusCell(status string) widgets.Cell {
	s := strings.ToLower(status)
	switch {
	case strings.HasPrefix(s, "ongo"):
		return widgets.Cell{Text: "● live", Style: liveSt}
	case strings.HasPrefix(s, "upcom"):
		return widgets.Cell{Text: "○ soon", Style: mutedSt}
	default:
		return widgets.Cell{Text: "✓ done", Style: mutedSt}
	}
}

func (s *Events) Focus()                           { s.table.Focus() }
func (s *Events) Blur()                            { s.table.Blur() }
func (s Events) Selected() string                  { return s.table.SelectedKey() }
func (s *Events) ClickVisual(i int) (string, bool) { return s.table.ClickVisual(i) }

func (s Events) Update(msg tea.Msg) (Events, tea.Cmd) {
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

func (s Events) View() string {
	header := title("events") + "\n" + hint("enter → open event · j/k move") + "\n\n"
	if s.table.Len() == 0 {
		return header + muted("no events in cache — seed it (see README)")
	}
	return header + s.table.View()
}
