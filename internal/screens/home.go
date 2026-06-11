package screens

import (
	"fmt"
	"strings"

	"github.com/jashkarangiya/valo-tui/internal/data"
	"github.com/jashkarangiya/valo-tui/internal/widgets"
)

// Home is the [h] landing dashboard: what's live, how many events, where to go.
type Home struct {
	w, h int
	body string
	// lineMatches maps a body row index to the match rendered there, so a click
	// on a team name can open its roster.
	lineMatches map[int]data.MatchCard
}

func NewHome(w, h int) Home {
	hm := Home{w: w, h: h}
	hm.Load()
	return hm
}

func (s *Home) SetSize(w, h int) { s.w, s.h = w, h }

func (s *Home) Load() {
	liveMatches := data.LiveMatches()
	events := data.ActiveEvents()
	ongoing := 0
	for _, e := range events {
		if strings.HasPrefix(strings.ToLower(e.Status), "ongo") {
			ongoing++
		}
	}
	ts := data.LastUpdated()

	var b strings.Builder
	b.WriteString(title("home") + "\n\n")

	// hero
	b.WriteString(textB("valorant esports, in your terminal") + "\n")
	b.WriteString(muted("tracking ") + text(fmt.Sprint(len(events))) + muted(" events · ") +
		text(fmt.Sprint(ongoing)) + muted(" ongoing · ") + live(fmt.Sprintf("%d live", len(liveMatches))) + "\n")
	if ts != "" {
		b.WriteString(muted("cache · "+ts+" UTC") + "\n")
	}
	b.WriteString("\n")

	// live now
	s.lineMatches = map[int]data.MatchCard{}
	if len(liveMatches) > 0 {
		b.WriteString(liveB("● live now") + "\n")
		for i, m := range liveMatches {
			if i >= 6 {
				break
			}
			s.lineMatches[strings.Count(b.String(), "\n")] = m
			b.WriteString(widgets.MatchLine(m) + "\n")
		}
	} else {
		b.WriteString(mutedB("live now") + "\n" + muted("nothing live right now") + "\n")
	}
	b.WriteString("\n")

	// where to
	b.WriteString(accentB("where to") + "\n")
	b.WriteString(muted("· ") + text("e") + muted("  events    ") + text("pick a tournament to open its results, standings & bracket") + "\n")
	b.WriteString(muted("· ") + text("l") + muted("  live      ") + text("all live matches across every region") + "\n")
	b.WriteString(muted("· ") + text("a") + muted("  about     ") + text("what this tool can do and the key bindings"))

	s.body = b.String()
}

func (s Home) View() string { return s.body }

// TeamAt returns the team name at a body-local (x, y) click, for opening a
// roster from a live-match line.
func (s Home) TeamAt(x, y int) (string, bool) {
	m, ok := s.lineMatches[y]
	if !ok {
		return "", false
	}
	return widgets.MatchLineHit(m, x)
}
