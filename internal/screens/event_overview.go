package screens

import (
	"fmt"
	"strings"

	"github.com/jashkarangiya/valo-tui/internal/data"
	"github.com/jashkarangiya/valo-tui/internal/styles"
)

// EventOverview is the [o] event landing page: what the event is and where
// things stand, before drilling into the children.
type EventOverview struct {
	w, h int
	body string
}

func NewEventOverview(w, h int) EventOverview { return EventOverview{w: w, h: h} }

func (s *EventOverview) SetSize(w, h int) { s.w, s.h = w, h }

func (s *EventOverview) Load(eventID int, eventName string) {
	var b strings.Builder
	b.WriteString(title("overview") + "\n\n")
	if eventID == 0 {
		b.WriteString(muted("select an event from ") + text("e events"))
		s.body = b.String()
		return
	}
	event, _ := data.EventByID(eventID)
	name := event.Name
	if name == "" {
		name = eventName
	}
	if name == "" {
		name = "event"
	}
	matches := data.EventMatchCards(eventID, name)

	b.WriteString(s.banner(event, name) + "\n\n")
	b.WriteString(s.progress(matches) + "\n")
	b.WriteString(s.nav())
	s.body = b.String()
}

func (s EventOverview) banner(e data.EventCard, name string) string {
	region := e.Region
	if region == "" {
		region = "—"
	}
	status := e.Status
	if status == "" {
		status = "—"
	}
	statusTxt := text(status)
	if strings.HasPrefix(strings.ToLower(status), "ongo") {
		statusTxt = live(status)
	}
	lines := []string{
		textB(name),
		muted("region ") + text(region) + "   " + muted("status ") + statusTxt,
	}
	if e.Start != "" && e.End != "" {
		lines = append(lines, muted("dates  ")+text(e.Start+" – "+e.End))
	} else if e.Start != "" {
		lines = append(lines, muted("dates  ")+text(e.Start))
	}
	if e.Prize != "" {
		lines = append(lines, muted("prize  ")+text(e.Prize))
	}
	return styles.Card.Render(strings.Join(lines, "\n"))
}

func (s EventOverview) progress(matches []data.MatchCard) string {
	var liveN, done, soon int
	var phases []string
	seen := map[string]bool{}
	for _, m := range matches {
		switch m.Status {
		case "live":
			liveN++
		case "completed":
			done++
		case "upcoming":
			soon++
		}
		if m.Phase != "" && !seen[m.Phase] {
			seen[m.Phase] = true
			phases = append(phases, m.Phase)
		}
	}

	var b strings.Builder
	b.WriteString(accentB("progress") + "\n")
	b.WriteString(muted("matches  ") + text(fmt.Sprintf("%d done", done)) + muted(" · ") +
		live(fmt.Sprintf("%d live", liveN)) + muted(" · ") + text(fmt.Sprintf("%d upcoming", soon)))
	if len(phases) == 0 {
		b.WriteString("\n" + muted("no matches cached for this event yet"))
		return b.String()
	}
	b.WriteString("\n\n")
	for i, ph := range phases {
		if i >= 8 {
			break
		}
		var left, liveHere int
		for _, m := range matches {
			if m.Phase != ph {
				continue
			}
			if m.Status != "completed" {
				left++
			}
			if m.Status == "live" {
				liveHere++
			}
		}
		state := text(fmt.Sprintf("%d left", left))
		if liveHere > 0 {
			state = live("live")
		} else if left == 0 {
			state = muted("complete")
		}
		b.WriteString(text(fmt.Sprintf("%-22s", ph)) + " " + state + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func (s EventOverview) nav() string {
	return accentB("quick nav") + "\n" +
		muted("· ") + text("r") + muted("  results    completed & live series") + "\n" +
		muted("· ") + text("f") + muted("  fixtures   what's still to come") + "\n" +
		muted("· ") + text("t") + muted("  standings  group tables") + "\n" +
		muted("· ") + text("b") + muted("  bracket    playoff tree") + "\n" +
		muted("· ") + text("m") + muted("  teams      rosters in this event")
}

func (s EventOverview) View() string { return s.body }
