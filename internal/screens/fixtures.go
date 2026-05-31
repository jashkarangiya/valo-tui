package screens

import (
	"strings"

	"github.com/jashkarangiya/valo-tui/internal/data"
	"github.com/jashkarangiya/valo-tui/internal/styles"
)

// Fixtures is the [f] event sub-page: upcoming matches grouped by phase.
type Fixtures struct {
	w, h int
	body string
}

func NewFixtures(w, h int) Fixtures { return Fixtures{w: w, h: h} }

func (s *Fixtures) SetSize(w, h int) { s.w, s.h = w, h }

func (s *Fixtures) Load(eventID int, eventName string) {
	var b strings.Builder
	b.WriteString(title("fixtures") + "\n\n")
	if eventID == 0 {
		b.WriteString(hint("select an event first"))
		s.body = b.String()
		return
	}
	var upcoming []data.MatchCard
	for _, m := range data.EventMatchCards(eventID, eventName) {
		if m.Status == "upcoming" {
			upcoming = append(upcoming, m)
		}
	}
	if len(upcoming) == 0 {
		b.WriteString(hint("nothing scheduled for this event"))
		s.body = b.String()
		return
	}

	// Group by phase, preserving first-seen order.
	var phases []string
	seen := map[string]bool{}
	for _, m := range upcoming {
		ph := m.Phase
		if ph == "" {
			ph = "scheduled"
		}
		if !seen[ph] {
			seen[ph] = true
			phases = append(phases, ph)
		}
	}
	for _, ph := range phases {
		b.WriteString("\n" + accentB(ph) + "\n")
		for _, m := range upcoming {
			mp := m.Phase
			if mp == "" {
				mp = "scheduled"
			}
			if mp != ph {
				continue
			}
			when := m.Time
			if when == "" {
				when = m.Date
			}
			if when == "" {
				when = "soon"
			}
			card := textB(m.Team1.Name+"  vs  "+m.Team2.Name) + "   " + accent("·") + "  " + text(when)
			b.WriteString(styles.Card.Render(card) + "\n")
		}
	}
	s.body = strings.TrimRight(b.String(), "\n")
}

func (s Fixtures) View() string { return s.body }
