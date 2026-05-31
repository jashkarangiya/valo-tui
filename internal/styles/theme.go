// Package styles holds the valo-tui colour palette and reusable lipgloss
// styles. It is the Go analog of valo_tui/styles.tcss — same four colours,
// same framed/Bento treatment.
package styles

import "charm.land/lipgloss/v2"

// Palette — VALORANT brand colours. This is the single source of colour for
// the whole app; every other file references these vars, so the theme can be
// changed here alone.
//
//	bg #0F1923 ("Rage") · text #ECE8E1 ("Nitro") · accent #FF4655 ("Tilt") · teal #18E2C4
var (
	BG      = lipgloss.Color("#0F1923") // Valorant dark navy
	Surface = lipgloss.Color("#1B2733") // elevated panels
	Text    = lipgloss.Color("#ECE8E1") // off-white
	Accent  = lipgloss.Color("#FF4655") // Valorant red — the signature colour
	Live    = lipgloss.Color("#FF4655")
	Muted   = lipgloss.Color("#7B8B97") // steel gray
	Rule    = lipgloss.Color("#2B3A45")
	Border  = lipgloss.Color("#2B3A45")
	Blue    = lipgloss.Color("#18E2C4") // mint/teal secondary

	// Match colours: team 1 (left) is the brand red, team 2 (right) the teal.
	Team1    = Accent
	Team2    = Blue
	SelBg    = lipgloss.Color("#2A3A47") // selected-row / cursor background
	BarTrack = lipgloss.Color("#243441") // empty bar track

	// Agent-role accents, tuned to the palette.
	RoleDuelist    = lipgloss.Color("#FF4655") // aggressive red
	RoleController = lipgloss.Color("#8B7BE8") // smoke purple
	RoleInitiator  = lipgloss.Color("#F0B43C") // recon amber
	RoleSentinel   = lipgloss.Color("#18E2C4") // anchor teal
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
