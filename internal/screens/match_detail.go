package screens

import (
	"fmt"
	"image/color"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/jashkarangiya/valo-tui/internal/data"
	"github.com/jashkarangiya/valo-tui/internal/styles"
)

var (
	team1Col = styles.Team1 // left team (Valorant red)
	team2Col = styles.Team2 // right team (teal)
)

var bracketPhases = []string{
	"playoff", "bracket", "upper", "lower", "grand final",
	"quarterfinal", "semifinal", "final",
}
var roleOrder = []string{"duelist", "initiator", "controller", "sentinel", "flex"}

// MatchDetail is the tactical broadcast overlay: hero score, series momentum,
// per-map scoreboards. Mirrors screens/match_detail.py.
type MatchDetail struct {
	matchID int
	w, h    int
	scroll  int
	detail  data.SeriesDetail
	ok      bool
}

func NewMatchDetail(matchID, w, h int) MatchDetail {
	m := MatchDetail{matchID: matchID, w: w, h: h}
	m.detail, m.ok = data.SeriesDetailFor(matchID)
	return m
}

func (m *MatchDetail) SetSize(w, h int) { m.w, m.h = w, h }

// TeamAt returns the team name for a click at overlay-local (x, y), so the
// header names open a roster. Only the unscrolled header band is hot; the
// left/right half of the width picks the team (names sit either side of "vs").
func (m MatchDetail) TeamAt(x, y int) (string, bool) {
	if !m.ok || m.scroll != 0 || y > 5 {
		return "", false
	}
	if x < m.w/2 {
		return m.detail.Team1.Name, true
	}
	return m.detail.Team2.Name, true
}

func (m MatchDetail) hasBracket() bool {
	if !m.ok {
		return false
	}
	p := strings.ToLower(m.detail.Phase)
	for _, k := range bracketPhases {
		if strings.Contains(p, k) {
			return true
		}
	}
	return false
}

func (m MatchDetail) Update(msg tea.Msg) (MatchDetail, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok {
		switch k.String() {
		case "esc", "q":
			return m, func() tea.Msg { return CloseOverlayMsg{} }
		case "b":
			if m.hasBracket() {
				name := m.detail.Event
				return m, func() tea.Msg { return OpenBracketMsg{EventName: name} }
			}
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

func (m MatchDetail) View() string {
	body := m.render()
	lines := strings.Split(body, "\n")
	// Reserve one row for the footer hint.
	visible := m.h - 1
	if visible < 1 {
		visible = 1
	}
	if m.scroll > len(lines)-visible {
		m.scroll = max(0, len(lines)-visible)
	}
	end := min(len(lines), m.scroll+visible)
	view := strings.Join(lines[m.scroll:end], "\n")
	footer := mutedSt.Render("j/k scroll · click team → roster · esc back")
	if m.hasBracket() {
		footer = mutedSt.Render("j/k scroll · b bracket · click team → roster · esc back")
	}
	return view + "\n" + footer
}

func (m MatchDetail) render() string {
	if !m.ok {
		return muted(fmt.Sprintf("no detail available for match %d", m.matchID))
	}
	d := m.detail
	var b strings.Builder
	b.WriteString(m.header(d))

	var maps []data.MapScore
	for _, mp := range d.Maps {
		if !mp.IsAggregate() {
			maps = append(maps, mp)
		}
	}
	b.WriteString(m.seriesMomentum(d, maps))
	if len(maps) == 0 {
		b.WriteString("\n" + muted("no map data yet"))
		return b.String()
	}
	for _, mp := range maps {
		b.WriteString(m.mapBlock(d, mp))
	}
	return b.String()
}

func (m MatchDetail) header(d data.SeriesDetail) string {
	s1, s2 := derefOr0(d.Team1.Score), derefOr0(d.Team2.Score)
	center := func(s string) string { return lipgloss.PlaceHorizontal(m.w, lipgloss.Center, s) }

	// One readable scoreline: bold team names flank a clear score, the leading
	// side's number brightened. No oversized figlet — the names stay legible.
	t1 := lipgloss.NewStyle().Foreground(team1Col).Bold(true).Render(d.Team1.Name)
	t2 := lipgloss.NewStyle().Foreground(team2Col).Bold(true).Render(d.Team2.Name)
	score := scoreNum(s1, s1 >= s2) + muted("  —  ") + scoreNum(s2, s2 >= s1)
	scoreline := t1 + "    " + score + "    " + t2

	var status string
	switch {
	case d.IsLive():
		status = lipgloss.NewStyle().Foreground(team1Col).Bold(true).Render("● live")
	case d.IsCompleted():
		status = muted("✓ final")
	default:
		r := d.Remaining
		if r == "" {
			r = "upcoming"
		}
		status = muted("○ " + r)
	}

	var parts []string
	if d.BestOf != "" {
		parts = append(parts, d.BestOf)
	}
	if d.Phase != "" {
		parts = append(parts, d.Phase)
	}
	meta := muted(strings.Join(parts, " · "))
	if meta != "" {
		meta += "   "
	}
	meta += status
	if m.hasBracket() {
		meta += "    " + lipgloss.NewStyle().Foreground(team1Col).Render("[b]") + " " + text("bracket")
	}

	var b strings.Builder
	b.WriteString(center(scoreline) + "\n")
	b.WriteString(center(meta) + "\n")
	b.WriteString(center(muted(d.Event)) + "\n")
	if intel := m.intel(d); intel != "" {
		b.WriteString(center(intel) + "\n")
	}
	b.WriteString("\n")
	return b.String()
}

// scoreNum renders a bold series-score number, dimmed when it isn't the leading
// side so the result reads at a glance.
func scoreNum(n int, leading bool) string {
	col := styles.Muted
	if leading {
		col = styles.Text
	}
	return lipgloss.NewStyle().Foreground(col).Bold(true).Render(fmt.Sprintf("%d", n))
}

func (m MatchDetail) intel(d data.SeriesDetail) string {
	var players []data.PlayerLine
	for _, mp := range d.Maps {
		players = append(players, mp.Players...)
	}
	if len(players) == 0 {
		return ""
	}
	top, fk := players[0], players[0]
	for _, p := range players {
		if derefOr0(p.ACS) > derefOr0(top.ACS) {
			top = p
		}
		if derefOr0(p.FK) > derefOr0(fk.FK) {
			fk = p
		}
	}
	return muted("intel:  top acs ") + text(fmt.Sprintf("%s %d", top.Name, derefOr0(top.ACS))) +
		muted("   ·   most FK ") + text(fmt.Sprintf("%s %d", fk.Name, derefOr0(fk.FK)))
}

func (m MatchDetail) seriesMomentum(d data.SeriesDetail, maps []data.MapScore) string {
	var played []data.MapScore
	for _, mp := range maps {
		if mp.HasScore() {
			played = append(played, mp)
		}
	}
	if len(played) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(accent("maps") + "\n")
	// Header columns are positioned to match the rows below: a 2-col map icon +
	// 8-col name, then score, momentum bar and winner.
	b.WriteString(mutedSt.Render(fmt.Sprintf("  %-10s%-7s%-13s%s", "map", "score", "momentum", "won by")) + "\n")
	for _, mp := range played {
		s1, s2 := derefOr0(mp.Team1Score), derefOr0(mp.Team2Score)
		won, wonCol := mp.Team1Short, team1Col
		if s2 > s1 {
			won, wonCol = mp.Team2Short, team2Col
		}
		bar := styles.WinBar(mp.Team1Score, mp.Team2Score, 12)
		line := "  " + styles.MapIcon(mp.Name) + " " + text(fmt.Sprintf("%-8s", clipRunes(mp.Name, 8))) +
			text(fmt.Sprintf("%-7s", fmt.Sprintf("%d–%d", s1, s2))) +
			bar + " " +
			lipgloss.NewStyle().Foreground(wonCol).Bold(true).Render(won)
		if pick := d.PickLabel(mp.Name); pick != "" {
			line += "  " + muted(pick)
		}
		b.WriteString(line + "\n")
	}
	b.WriteString("\n")
	return b.String()
}

func (m MatchDetail) mapBlock(d data.SeriesDetail, mp data.MapScore) string {
	var b strings.Builder
	b.WriteString(m.mapTitle(d, mp) + "\n")
	if mp.State() == "pending" {
		b.WriteString(muted("  (not played yet)") + "\n\n")
		return b.String()
	}
	if len(mp.Rounds) > 0 {
		b.WriteString("  rounds  " + styles.Momentum(mp.Rounds, mp.Team1Short) + "\n")
	}
	b.WriteString(m.scoreboard(mp))
	return b.String()
}

func (m MatchDetail) mapTitle(d data.SeriesDetail, mp data.MapScore) string {
	score := muted("TBD")
	if mp.HasScore() {
		score = text(fmt.Sprintf("%d–%d", derefOr0(mp.Team1Score), derefOr0(mp.Team2Score)))
	}
	pick := d.PickLabel(mp.Name)
	pickTxt := ""
	if pick != "" {
		pickTxt = "   " + muted(pick)
	}
	return "\n" + styles.MapIcon(mp.Name) + " " + textB(mp.Name) + "   " + score + pickTxt
}

func (m MatchDetail) scoreboard(mp data.MapScore) string {
	teams := []struct {
		short  string
		colour color.Color
	}{{mp.Team1Short, team1Col}, {mp.Team2Short, team2Col}}

	// Pad the label gutter to the player rows' name+agent prefix so the column
	// headers sit directly over their values: 4 indent + 12 name + 1 + glyph(1)
	// + 1 + 8 agent = 27, and the stats block leads with a space ⇒ 28.
	hdr := mutedSt.Render(fmt.Sprintf("%-28s%4s %3s %3s %3s %5s %4s", "", "acs", "k", "d", "a", "adr", "hs"))
	var b strings.Builder
	for _, t := range teams {
		var tp []data.PlayerLine
		for _, p := range mp.Players {
			if p.TeamShort == t.short {
				tp = append(tp, p)
			}
		}
		if len(tp) == 0 {
			continue
		}
		short := t.short
		if short == "" {
			short = "?"
		}
		b.WriteString(lipgloss.NewStyle().Foreground(t.colour).Bold(true).Render(short) + "\n")
		b.WriteString(hdr + "\n")

		byRole := map[string][]data.PlayerLine{}
		for _, p := range tp {
			role := styles.AgentRole(p.Agent())
			if role == "" {
				role = "flex"
			}
			byRole[role] = append(byRole[role], p)
		}
		for _, role := range roleOrder {
			ps := byRole[role]
			if len(ps) == 0 {
				continue
			}
			b.WriteString("  " + muted(role+"s") + "\n")
			sort.SliceStable(ps, func(a, bb int) bool { return derefOr0(ps[a].ACS) > derefOr0(ps[bb].ACS) })
			for _, p := range ps {
				b.WriteString(m.playerLine(p, t.colour) + "\n")
			}
		}
		b.WriteString("\n")
	}
	return b.String()
}

func (m MatchDetail) playerLine(p data.PlayerLine, colour color.Color) string {
	glyph, gcol := styles.AgentGlyph(p.Agent())
	agent := p.Agent()
	if agent == "" {
		agent = "—"
	}
	agent = clipRunes(agent, 8)
	name := clipRunes(p.Name, 12)
	nameCol := lipgloss.NewStyle().Foreground(colour).Render(fmt.Sprintf("%-12s", name))
	glyphCol := lipgloss.NewStyle().Foreground(gcol).Render(glyph)
	stats := fmt.Sprintf(" %4s %3s %3s %3s %5s %4s", n(p.ACS), n(p.K), n(p.D), n(p.A), fNum(p.ADR), pct(p.HSPct))
	return "    " + nameCol + " " + glyphCol + " " + mutedSt.Render(fmt.Sprintf("%-8s", agent)) + textSt.Render(stats)
}

// ── numeric formatting (mirror _n / _f / _pct) ──────────────

func n(v *int) string {
	if v == nil {
		return "–"
	}
	return fmt.Sprint(*v)
}

func fNum(v *float64) string {
	if v == nil {
		return "–"
	}
	return fmt.Sprintf("%.0f", *v)
}

func pct(v *float64) string {
	if v == nil {
		return "–"
	}
	return fmt.Sprintf("%.0f%%", *v)
}

func derefOr0(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

func clipRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}
