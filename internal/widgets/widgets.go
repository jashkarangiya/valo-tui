// Package widgets holds reusable render helpers: the context-aware sidebar and
// the one-line match summary. Ported from valo_tui/screens/widgets.py.
package widgets

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/jashkarangiya/valo-tui/internal/data"
	"github.com/jashkarangiya/valo-tui/internal/styles"
)

const brand = "valo-tui · vct26"

// NavItem is one selectable rail entry: hotkey, route id, label.
type NavItem struct {
	Key, Route, Label string
}

// GlobalNav is always visible.
var GlobalNav = []NavItem{
	{"h", "home", "home"},
	{"e", "events", "events"},
	{"l", "live", "live"},
	{"a", "about", "about"},
}

// EventNav is revealed only while an event is in focus.
var EventNav = []NavItem{
	{"o", "overview", "overview"},
	{"r", "results", "results"},
	{"f", "fixtures", "fixtures"},
	{"t", "standings", "standings"},
	{"b", "bracket", "bracket"},
	{"m", "teams", "teams"},
}

var (
	mutedS  = lipgloss.NewStyle().Foreground(styles.Muted)
	textS   = lipgloss.NewStyle().Foreground(styles.Text)
	accentB = lipgloss.NewStyle().Foreground(styles.Accent).Bold(true)
	textB   = lipgloss.NewStyle().Foreground(styles.Text).Bold(true)
	ruleS   = lipgloss.NewStyle().Foreground(styles.Rule)
	liveS   = lipgloss.NewStyle().Foreground(styles.Live)
)

// Sidebar renders the nav rail for the current route. eventName is "" when no
// event is in focus; focused controls whether the active row shows the ›
// cursor (set when the rail itself has keyboard focus).
func Sidebar(active, eventName string, focused bool) string {
	row := func(it NavItem) string {
		if it.Route == active {
			marker := " "
			if focused {
				marker = lipgloss.NewStyle().Foreground(styles.Accent).Render("›")
			}
			return marker + mutedS.Render("["+it.Key+"]") + " " + accentB.Render(it.Label)
		}
		return " " + mutedS.Render("["+it.Key+"]") + " " + textS.Render(it.Label)
	}

	var b strings.Builder
	b.WriteString(textB.Render(brand) + "\n")
	b.WriteString(ruleS.Render(strings.Repeat("─", 20)) + "\n\n")
	for _, it := range GlobalNav {
		b.WriteString(row(it) + "\n")
	}

	if eventName != "" {
		b.WriteString("\n" + mutedS.Render("── event ──") + "\n")
		b.WriteString(accentB.Render(clip(eventName, 18)) + "\n\n")
		for _, it := range EventNav {
			b.WriteString(row(it) + "\n")
		}
	}

	b.WriteString("\n" + ruleS.Render(strings.Repeat("─", 20)) + "\n\n")
	b.WriteString(mutedS.Render("↑↓    navigate") + "\n")
	b.WriteString(mutedS.Render("enter open") + "\n")
	b.WriteString(mutedS.Render("esc   back here") + "\n")
	b.WriteString(mutedS.Render("q     quit"))
	return b.String()
}

// MatchLine is a one-line summary of a match for the dashboard panels.
func MatchLine(m data.MatchCard) string {
	var dot, score string
	switch {
	case m.IsLive():
		dot = liveS.Render("●") + " "
		score = liveS.Render(scoreOf(m))
	case m.Status == "completed":
		dot = mutedS.Render("·") + " "
		score = mutedS.Render(scoreOf(m))
	default:
		dot = mutedS.Render("○") + " "
		t := m.Time
		if t == "" {
			t = "soon"
		}
		score = mutedS.Render(t)
	}
	t1 := clip(m.Team1.Name, 12)
	t2 := clip(m.Team2.Name, 12)
	return dot + t1 + " " + mutedS.Render("vs") + " " + t2 + "  " + score
}

func scoreOf(m data.MatchCard) string {
	s1, s2 := "–", "–"
	if m.Team1.Score != nil {
		s1 = itoa(*m.Team1.Score)
	}
	if m.Team2.Score != nil {
		s2 = itoa(*m.Team2.Score)
	}
	return s1 + "–" + s2
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func clip(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}
