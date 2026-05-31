// Package app is the root Bubble Tea model: the framed shell with a
// context-aware sidebar and a content switcher. It mirrors valo_tui/app.py's
// event-first information architecture — global routes are always available,
// event routes only once a tournament is in focus.
package app

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/jashkarangiya/valo-tui/internal/data"
	"github.com/jashkarangiya/valo-tui/internal/screens"
	"github.com/jashkarangiya/valo-tui/internal/styles"
	"github.com/jashkarangiya/valo-tui/internal/widgets"
)

const sidebarWidth = 28

// globalRoutes are always reachable; eventRoutes need an event in focus.
var (
	globalRoutes = map[string]bool{"home": true, "events": true, "live": true, "about": true}
	eventRoutes  = map[string]bool{
		"overview": true, "results": true, "fixtures": true,
		"standings": true, "bracket": true, "teams": true,
	}
)

// Model is the application shell.
type Model struct {
	w, h int

	route     string // current content route, or "splash" before entry
	navFocus  bool   // whether the sidebar rail holds focus
	eventID   int    // 0 ⇒ global scope
	eventName string

	splash screens.Splash
	live   screens.GlobalLive
}

// New builds the shell at the given size.
func New(w, h int) Model {
	cw, ch := contentSize(w, h)
	return Model{
		w:      w,
		h:      h,
		route:  "splash",
		splash: screens.NewSplash(w, h),
		live:   screens.NewGlobalLive(cw, ch),
	}
}

// contentSize is the inner area available to a screen, after the frame border
// and the sidebar are subtracted.
func contentSize(w, h int) (int, int) {
	cw := w - sidebarWidth - 5
	if cw < 20 {
		cw = 20
	}
	ch := h - 4
	if ch < 5 {
		ch = 5
	}
	return cw, ch
}

func (m Model) Init() tea.Cmd {
	return m.splash.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		cw, ch := contentSize(m.w, m.h)
		m.splash.SetSize(m.w, m.h)
		m.live.SetSize(cw, ch)
		return m, nil

	case tea.KeyPressMsg:
		if m.route == "splash" {
			var cmd tea.Cmd
			m.splash, cmd = m.splash.Update(msg)
			return m, cmd
		}
		if model, cmd, handled := m.handleKey(msg); handled {
			return model, cmd
		}

	case screens.EnterAppMsg:
		m.route = "live"
		m.navFocus = true
		return m, m.live.Init()

	case screens.SwitchRouteMsg:
		return m.show(msg.To)

	case screens.SelectEventMsg:
		m.eventID = msg.ID
		if e, ok := data.EventByID(msg.ID); ok {
			m.eventName = e.Name
		}
		return m.show(msg.Tab)
	}

	// Delegate to the active screen.
	return m.delegate(msg)
}

// handleKey processes global hotkeys. The returned bool reports whether the key
// was consumed here (vs. delegated to the active screen).
func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd, bool) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit, true
	case "h":
		mm, cmd := m.show("home")
		return mm, cmd, true
	case "e":
		mm, cmd := m.show("events")
		return mm, cmd, true
	case "l":
		mm, cmd := m.show("live")
		return mm, cmd, true
	case "a":
		mm, cmd := m.show("about")
		return mm, cmd, true
	case "o", "r", "f", "t", "b", "m":
		mm, cmd := m.show(routeForKey(msg.String()))
		return mm, cmd, true
	case "esc":
		m.navFocus = true
		return m, nil, true
	}
	return m, nil, false
}

func routeForKey(k string) string {
	return map[string]string{
		"o": "overview", "r": "results", "f": "fixtures",
		"t": "standings", "b": "bracket", "m": "teams",
	}[k]
}

// show switches the visible route, refusing event routes without an event.
func (m Model) show(route string) (tea.Model, tea.Cmd) {
	if eventRoutes[route] && m.eventID == 0 {
		// "select an event first" — silently ignore for now.
		return m, nil
	}
	if !globalRoutes[route] && !eventRoutes[route] {
		return m, nil
	}
	m.route = route
	m.navFocus = false
	if route == "live" {
		return m, m.live.Init()
	}
	return m, nil
}

// delegate hands a message to the active screen's model.
func (m Model) delegate(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch m.route {
	case "splash":
		m.splash, cmd = m.splash.Update(msg)
	case "live":
		m.live, cmd = m.live.Update(msg)
	}
	return m, cmd
}

func (m Model) View() tea.View {
	if m.route == "splash" {
		return altScreen(m.splash.View())
	}

	sidebar := lipgloss.NewStyle().
		Width(sidebarWidth).
		Padding(1, 2).
		BorderRight(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(styles.Border).
		Render(widgets.Sidebar(m.activeNav(), m.eventName, m.navFocus))

	content := lipgloss.NewStyle().Padding(1, 2).Render(m.content())

	shell := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content)
	return altScreen(styles.Frame.Margin(1, 2).Render(shell))
}

// altScreen wraps content in a full-window view (v2 makes the alt screen a
// declarative property of the view rather than a program option).
func altScreen(content string) tea.View {
	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

// activeNav is the sidebar's highlighted route.
func (m Model) activeNav() string { return m.route }

// content renders the current screen's body. Screens beyond the proof-of-concept
// pair show a placeholder until ported.
func (m Model) content() string {
	switch m.route {
	case "live":
		return m.live.View()
	default:
		return styles.Hint.Render(m.route + " — not ported yet\n\npress l for live, q to quit")
	}
}
