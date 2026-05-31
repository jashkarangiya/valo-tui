// Package styles holds the valo-tui colour palette and reusable lipgloss
// styles. It is the Go analog of valo_tui/styles.tcss — same four colours,
// same framed/Bento treatment.
package styles

import "charm.land/lipgloss/v2"

// Palette — kept byte-for-byte identical to styles.tcss so the Go build is
// visually continuous with the shipped Python TUI.
//
//	bg #0a1822 · text #c8d8e8 · accent #e8674e · muted #4a708b
var (
	BG     = lipgloss.Color("#0a1822")
	Text   = lipgloss.Color("#c8d8e8")
	Accent = lipgloss.Color("#e8674e") // also "LIVE"
	Live   = lipgloss.Color("#e8674e")
	Muted  = lipgloss.Color("#4a708b")
	Rule   = lipgloss.Color("#1c3a52")
	Border = lipgloss.Color("#1c3a52")
	Blue   = lipgloss.Color("#5d9ce8")
)

var (
	// Frame is the outer rounded border (#frame in the .tcss).
	Frame = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Border)

	// PageTitle is the bold accent screen heading (.page-title).
	PageTitle = lipgloss.NewStyle().
			Foreground(Accent).
			Bold(true).
			MarginBottom(1)

	// Hint is muted helper text (.hint).
	Hint = lipgloss.NewStyle().Foreground(Muted)

	// Card is a Bento compartment (.card / RegionPanel).
	Card = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Border).
		Padding(0, 1)

	// CardLive is a compartment with a live match (RegionPanel.live).
	CardLive = Card.BorderForeground(Accent)

	// IntlBar is the pinned international banner (#intl-bar).
	IntlBar = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Accent).
		Padding(0, 1)
)
