package styles

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/jashkarangiya/valo-tui/internal/data"
)

// Block-character bar + sparkline primitives, ported from style/bars.py.
// Colours come from the central palette (Team1 / Team2 / BarTrack).

const (
	barFull  = "█"
	barEmpty = "░"
)

// WinBar is a proportional round-share bar; the left (accent) is team A's share.
func WinBar(a, b *int, width int) string {
	av, bv := derefInt(a), derefInt(b)
	total := av + bv
	if total == 0 {
		return lipgloss.NewStyle().Foreground(BarTrack).Render(strings.Repeat(barEmpty, width))
	}
	fill := int(float64(av)/float64(total)*float64(width) + 0.5)
	if fill < 0 {
		fill = 0
	}
	if fill > width {
		fill = width
	}
	full := lipgloss.NewStyle().Foreground(Team1).Render(strings.Repeat(barFull, fill))
	empty := lipgloss.NewStyle().Foreground(BarTrack).Render(strings.Repeat(barEmpty, width-fill))
	return full + empty
}

// Momentum renders round-by-round momentum: ▲ attacker win, ▼ defender win,
// coloured by which team won.
func Momentum(rounds []data.RoundLine, team1Short string) string {
	if len(rounds) == 0 {
		return ""
	}
	var b strings.Builder
	for _, r := range rounds {
		glyph := "▼"
		if r.IsAttack() {
			glyph = "▲"
		}
		colour := Team2
		if r.WinnerShort == team1Short {
			colour = Team1
		}
		b.WriteString(lipgloss.NewStyle().Foreground(colour).Render(glyph))
	}
	return b.String()
}

func derefInt(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}
