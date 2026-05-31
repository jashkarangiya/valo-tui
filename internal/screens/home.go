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
	if len(liveMatches) > 0 {
		b.WriteString(liveB("● live now") + "\n")
		for i, m := range liveMatches {
			if i >= 6 {
				break
			}
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
