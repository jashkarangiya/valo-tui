package screens

import (
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/jashkarangiya/valo-tui/internal/data"
	"github.com/jashkarangiya/valo-tui/internal/styles"
)

// version is shown on the splash and about page. Kept here until a build-time
// ldflags value replaces it.
const version = "0.2.0"

// "valo-tui" in figlet ansi_shadow (the same font family as the match-detail
// score hero), so the branding is consistent across the app.
const logoArt = `██╗   ██╗ █████╗ ██╗      ██████╗    ████████╗██╗   ██╗██╗
██║   ██║██╔══██╗██║     ██╔═══██╗   ╚══██╔══╝██║   ██║██║
██║   ██║███████║██║     ██║   ██║█████╗██║   ██║   ██║██║
╚██╗ ██╔╝██╔══██║██║     ██║   ██║╚════╝██║   ██║   ██║██║
 ╚████╔╝ ██║  ██║███████╗╚██████╔╝      ██║   ╚██████╔╝██║
  ╚═══╝  ╚═╝  ╚═╝╚══════╝ ╚═════╝       ╚═╝    ╚═════╝ ╚═╝`

// Splash is the landing page: logo, tag, cache freshness, enter hint.
type Splash struct {
	w, h  int
	fresh string
}

// NewSplash builds the splash, reading cache freshness up front.
func NewSplash(w, h int) Splash {
	ts := data.LastUpdated()
	fresh := "cache empty — start the worker"
	if ts != "" {
		fresh = "cache · " + ts + " UTC"
	}
	return Splash{w: w, h: h, fresh: "v" + version + "  ·  " + fresh}
}

func (s *Splash) SetSize(w, h int) { s.w, s.h = w, h }

func (s Splash) Init() tea.Cmd {
	// Auto-advance as a fallback so it never gets stuck.
	return tea.Tick(6*time.Second, func(time.Time) tea.Msg { return EnterAppMsg{} })
}

func (s Splash) Update(msg tea.Msg) (Splash, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "enter", " ", "space":
			return s, func() tea.Msg { return EnterAppMsg{} }
		}
	}
	return s, nil
}

func (s Splash) View() string {
	logo := lipgloss.NewStyle().Foreground(styles.Accent).Bold(true).Render(logoArt)
	tag := lipgloss.NewStyle().Foreground(styles.Muted).Render("valorant esports in your terminal")
	fresh := lipgloss.NewStyle().Foreground(styles.Muted).Render(s.fresh)
	hint := lipgloss.NewStyle().Foreground(styles.Text).Render("[ press enter ]")

	body := lipgloss.JoinVertical(lipgloss.Center, logo, "", tag, fresh, "", hint)
	return lipgloss.Place(s.w, s.h, lipgloss.Center, lipgloss.Center, body)
}
