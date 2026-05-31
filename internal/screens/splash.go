package screens

import (
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/jashkarangiya/valo-tui/internal/data"
	"github.com/jashkarangiya/valo-tui/internal/styles"
)

// version is shown on the splash. Kept here until a build-time ldflags value
// replaces it.
const version = "0.1.0"

// figlet "valo-tui" in a slant-ish font, mirroring the Python splash.
const logoArt = `                  __                 __        _
  _   ______ _   / /___        ___  / /  __ __(_)
 | | / / __ ` + "`" + `/  / // _ \      / _ \/ /  / // / /
 | |/ / /_/ /  / // // /     / // / /  / // / / /
 |___/\__,_/  /_//_//_/      \___/_/   \_,_/_/_/   `

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
