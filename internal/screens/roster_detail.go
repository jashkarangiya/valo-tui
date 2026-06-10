package screens

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/jashkarangiya/valo-tui/internal/data"
)

// RosterDetail is the team-roster overlay: the active lineup plus staff, opened
// by clicking any team name. Stacks above the match-detail overlay when reached
// from there.
type RosterDetail struct {
	teamName string
	w, h     int
	scroll   int
	roster   data.Roster
	ok       bool
}

// NewRosterDetail builds the overlay for a team name, loading its cached roster.
func NewRosterDetail(teamName string, w, h int) RosterDetail {
	r := RosterDetail{teamName: teamName, w: w, h: h}
	r.roster, r.ok = data.RosterByTeamName(teamName)
	return r
}

func (m *RosterDetail) SetSize(w, h int) { m.w, m.h = w, h }

func (m RosterDetail) Update(msg tea.Msg) (RosterDetail, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok {
		switch k.String() {
		case "esc", "q":
			return m, func() tea.Msg { return CloseOverlayMsg{} }
		case "j", "down":
			m.scroll++
		case "k", "up":
			if m.scroll > 0 {
				m.scroll--
			}
		}
	}
	return m, nil
}

func (m RosterDetail) View() string {
	body := m.render()
	lines := strings.Split(body, "\n")
	visible := m.h - 1
	if visible < 1 {
		visible = 1
	}
	if m.scroll > len(lines)-visible {
		m.scroll = max(0, len(lines)-visible)
	}
	end := min(len(lines), m.scroll+visible)
	view := strings.Join(lines[m.scroll:end], "\n")
	return view + "\n" + muted("esc back")
}

func (m RosterDetail) render() string {
	name := m.teamName
	if m.roster.Team != "" {
		name = m.roster.Team
	}
	center := func(s string) string { return lipgloss.PlaceHorizontal(m.w, lipgloss.Center, s) }

	var b strings.Builder
	b.WriteString(center(accentB(name)) + "\n")

	if !m.ok {
		b.WriteString("\n" + center(muted("roster not cached yet")) + "\n")
		b.WriteString(center(muted("(the fetcher loads rosters on its slow cadence)")) + "\n")
		return b.String()
	}

	players, staff := m.roster.Players(), m.roster.Staff()
	b.WriteString(center(muted(fmt.Sprintf("%d players", len(players)))) + "\n\n")

	hdr := fmt.Sprintf("  %-14s %-4s %s", "player", "cc", "name")
	b.WriteString(mutedSt.Render(hdr) + "\n")
	for _, p := range players {
		b.WriteString(m.memberLine(p) + "\n")
	}
	if len(staff) > 0 {
		b.WriteString("\n" + accent("staff") + "\n")
		for _, s := range staff {
			b.WriteString(m.memberLine(s) + "\n")
		}
	}
	return b.String()
}

// memberLine renders one roster row: captain star, alias, country, real name +
// (for staff) the role.
func (m RosterDetail) memberLine(p data.RosterMember) string {
	star := "  "
	if p.Captain {
		star = accent("★ ")
	}
	alias := clipRunes(p.Alias, 14)
	if alias == "" {
		alias = "—"
	}
	cc := strings.ToUpper(p.Country)
	if cc == "" {
		cc = "··"
	}
	line := star + textSt.Render(fmt.Sprintf("%-14s", alias)) + " " +
		mutedSt.Render(fmt.Sprintf("%-4s", cc)) + text(p.Name)
	if p.Role != "" {
		line += "  " + muted("· "+p.Role)
	}
	return line
}
