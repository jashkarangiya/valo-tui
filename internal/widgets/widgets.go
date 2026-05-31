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
	accentS = lipgloss.NewStyle().Foreground(styles.Accent)
	accentB = lipgloss.NewStyle().Foreground(styles.Accent).Bold(true)
	textB   = lipgloss.NewStyle().Foreground(styles.Text).Bold(true)
	ruleS   = lipgloss.NewStyle().Foreground(styles.Rule)
	liveS   = lipgloss.NewStyle().Foreground(styles.Live)
	rowSel  = lipgloss.NewStyle().Foreground(styles.Accent).Background(styles.SelBg).Bold(true)
)

// navRow renders one rail entry. The active page ALWAYS shows the › cursor and
// a highlighted [key] + label so you can see where you are; when the rail holds
// keyboard focus the active row also gets a background highlight.
func navRow(it NavItem, active, focused bool) string {
	if !active {
		return "  " + mutedS.Render("["+it.Key+"]") + " " + textS.Render(it.Label)
	}
	if focused {
		return rowSel.Render("› [" + it.Key + "] " + it.Label)
	}
	return accentS.Render("› ["+it.Key+"]") + " " + accentB.Render(it.Label)
}

// buildSidebar is the single source of the rail layout: it returns the rendered
// lines and a map from line index → route for click hit-testing.
func buildSidebar(active, eventName string, focused bool) ([]string, map[int]string) {
	var lines []string
	routes := map[int]string{}
	add := func(s string) { lines = append(lines, s) }
	addNav := func(it NavItem) {
		routes[len(lines)] = it.Route
		add(navRow(it, it.Route == active, focused))
	}

	add(textB.Render(brand))
	add(ruleS.Render(strings.Repeat("─", 20)))
	add("")
	for _, it := range GlobalNav {
		addNav(it)
	}
	if eventName != "" {
		add("")
		add(mutedS.Render("── event ──"))
		add(accentB.Render(clip(eventName, 18)))
		add("")
		for _, it := range EventNav {
			addNav(it)
		}
	}
	add("")
	add(ruleS.Render(strings.Repeat("─", 20)))
	add("")
	add(mutedS.Render("↑↓    navigate"))
	add(mutedS.Render("enter  open"))
	add(mutedS.Render("esc    back to rail"))
	add(mutedS.Render("q      quit"))
	return lines, routes
}

// Sidebar renders the nav rail for the current route.
func Sidebar(active, eventName string, focused bool) string {
	lines, _ := buildSidebar(active, eventName, focused)
	return strings.Join(lines, "\n")
}

// SidebarRoutes maps a sidebar text-line index to the route on that line (for
// mouse clicks). Only depends on whether an event is in focus.
func SidebarRoutes(eventName string) map[int]string {
	_, routes := buildSidebar("", eventName, false)
	return routes
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
