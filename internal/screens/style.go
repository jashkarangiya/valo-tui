package screens

import (
	"charm.land/lipgloss/v2"

	"github.com/jashkarangiya/valo-tui/internal/styles"
)

// Concise inline-colour helpers so screen bodies read close to the v1 Rich
// markup they were ported from.

var (
	mutedSt  = lipgloss.NewStyle().Foreground(styles.Muted)
	textSt   = lipgloss.NewStyle().Foreground(styles.Text)
	accentSt = lipgloss.NewStyle().Foreground(styles.Accent)
	liveSt   = lipgloss.NewStyle().Foreground(styles.Live)
	blueSt   = lipgloss.NewStyle().Foreground(styles.Blue)
)

func muted(s string) string   { return mutedSt.Render(s) }
func text(s string) string    { return textSt.Render(s) }
func accent(s string) string  { return accentSt.Render(s) }
func live(s string) string    { return liveSt.Render(s) }
func mutedB(s string) string  { return mutedSt.Bold(true).Render(s) }
func textB(s string) string   { return textSt.Bold(true).Render(s) }
func accentB(s string) string { return accentSt.Bold(true).Render(s) }
func liveB(s string) string   { return liveSt.Bold(true).Render(s) }

func title(s string) string { return styles.PageTitle.Render(s) }
func hint(s string) string  { return styles.Hint.Render(s) }
