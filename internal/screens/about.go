package screens

import "strings"

// About is the [a] page: what valo-tui is, the data source, and the keys.
type About struct{ w, h int }

func NewAbout(w, h int) About { return About{w: w, h: h} }

func (s *About) SetSize(w, h int) { s.w, s.h = w, h }
func (s *About) Load()            {}

func (s About) View() string {
	var b strings.Builder
	b.WriteString(title("about") + "\n\n")
	b.WriteString(textB("valo-tui") + "  " + muted("v"+version) + "\n")
	b.WriteString(text("A terminal-native tracker for global Valorant esports.") + "\n\n")

	b.WriteString(accent("what you can do") + "\n")
	for _, l := range []string{
		"· browse events by region and stage",
		"· open an event for its overview, results, fixtures,",
		"  standings, bracket and teams",
		"· follow live matches with map scores across regions",
		"· drill into a match for veto, maps, agents, ACS,",
		"  K/D/A, ADR, HS%, FK and FD",
	} {
		b.WriteString(muted(l) + "\n")
	}
	b.WriteString("\n" + accent("data") + "\n")
	b.WriteString(muted("· source   ") + text("vlr.gg") + "\n")
	b.WriteString(muted("· cache    ") + text("SQLite, written by a background worker") + "\n")
	b.WriteString(muted("· the UI never blocks on the network") + "\n\n")

	b.WriteString(accent("global nav") + "\n")
	b.WriteString(muted("· h ") + text("home      ") + muted("landing dashboard") + "\n")
	b.WriteString(muted("· e ") + text("events    ") + muted("pick a tournament to focus on") + "\n")
	b.WriteString(muted("· l ") + text("live      ") + muted("global Bento dashboard") + "\n")
	b.WriteString(muted("· a ") + text("about     ") + muted("this page") + "\n\n")

	b.WriteString(accent("inside an event") + "\n")
	b.WriteString(muted("· o ") + text("overview  ") + muted("· ") + text("r ") + muted("results  ") + text("· f ") + muted("fixtures") + "\n")
	b.WriteString(muted("· t ") + text("standings ") + muted("· ") + text("b ") + muted("bracket  ") + text("· m ") + muted("teams") + "\n\n")

	b.WriteString(accent("keys") + "\n")
	b.WriteString(muted("· ↑↓ / j k move · enter open · esc back to nav") + "\n")
	b.WriteString(muted("· ctrl+r refresh · q quit"))
	return b.String()
}
