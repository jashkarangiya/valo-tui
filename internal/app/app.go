// Package app is the root Bubble Tea model: the framed shell with a
// context-aware sidebar and a content switcher. It mirrors valo_tui/app.py's
// event-first information architecture — global routes are always available,
// event routes only once a tournament is in focus.
package app

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/jashkarangiya/valo-tui/internal/data"
	"github.com/jashkarangiya/valo-tui/internal/screens"
	"github.com/jashkarangiya/valo-tui/internal/styles"
	"github.com/jashkarangiya/valo-tui/internal/widgets"
)

var (
	globalRoutes = []string{"home", "events", "live", "about"}
	eventRoutes  = []string{"overview", "results", "fixtures", "standings", "bracket", "teams"}
)

func isEventRoute(r string) bool {
	for _, e := range eventRoutes {
		if e == r {
			return true
		}
	}
	return false
}

// letterRoute maps a hotkey to the route it jumps to.
var letterRoute = map[string]string{
	"h": "home", "e": "events", "l": "live", "a": "about",
	"o": "overview", "r": "results", "f": "fixtures",
	"t": "standings", "b": "bracket", "m": "teams",
}

// Model is the application shell.
type Model struct {
	w, h int

	route     string // current content route, or "splash" before entry
	navFocus  bool   // whether the sidebar rail holds focus
	eventID   int    // 0 ⇒ global scope
	eventName string

	overlay *screens.MatchDetail // non-nil ⇒ match-detail overlay is open

	splash    screens.Splash
	home      screens.Home
	events    screens.Events
	live      screens.GlobalLive
	about     screens.About
	overview  screens.EventOverview
	results   screens.Results
	fixtures  screens.Fixtures
	standings screens.Standings
	bracket   screens.Bracket
	teams     screens.Teams
}

// New builds the shell at the given size.
func New(w, h int) Model {
	cw, ch := contentSize(w, h)
	return Model{
		w: w, h: h, route: "splash",
		splash:    screens.NewSplash(w, h),
		home:      screens.NewHome(cw, ch),
		events:    screens.NewEvents(cw, ch),
		live:      screens.NewGlobalLive(cw, ch),
		about:     screens.NewAbout(cw, ch),
		overview:  screens.NewEventOverview(cw, ch),
		results:   screens.NewResults(cw, ch),
		fixtures:  screens.NewFixtures(cw, ch),
		standings: screens.NewStandings(cw, ch),
		bracket:   screens.NewBracket(cw, ch),
		teams:     screens.NewTeams(cw, ch),
	}
}

// Layout budget. The shell is sidebar | content inside the rounded frame.
// Fixed sizes keep the frame the SAME on every screen so the border never
// jumps as you navigate. In this lipgloss, Width/Height are TOTAL sizes
// (padding + border are absorbed), so the text area is the total minus chrome.
const sidebarTotal = 26 // total sidebar column width

// contentSize is the text area available to a screen after all chrome:
// frame (margin 4 + border 2) + sidebar + content padding (4) horizontally;
// frame (margin 2 + border 2) + content padding (2) vertically.
func contentSize(w, h int) (int, int) {
	cw := w - 6 - sidebarTotal - 4
	if cw < 24 {
		cw = 24
	}
	ch := h - 6
	if ch < 6 {
		ch = 6
	}
	return cw, ch
}

func (m Model) Init() tea.Cmd { return m.splash.Init() }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.resize(msg.Width, msg.Height)
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKey(msg)

	case tea.MouseClickMsg:
		return m.handleClick(msg.Mouse())

	case screens.EnterAppMsg:
		m.route = "live"
		m.navFocus = true
		m.live.Load()
		return m, m.live.Init()

	case screens.SwitchRouteMsg:
		m.show(msg.To)
		return m, nil

	case screens.SelectEventMsg:
		m.selectEvent(msg.ID, msg.Tab)
		return m, nil

	case screens.CloseOverlayMsg:
		m.overlay = nil
		return m, nil

	case screens.OpenBracketMsg:
		m.overlay = nil
		if id, ok := eventIDByName(msg.EventName); ok {
			m.selectEvent(id, "bracket")
		}
		return m, nil

	default:
		// Async messages (global-live fetch + ticks) keep the dashboard fresh
		// regardless of which screen is focused.
		var cmd tea.Cmd
		m.live, cmd = m.live.Update(msg)
		return m, cmd
	}
}

func (m *Model) resize(w, h int) {
	m.w, m.h = w, h
	cw, ch := contentSize(w, h)
	m.splash.SetSize(w, h)
	m.home.SetSize(cw, ch)
	m.events.SetSize(cw, ch)
	m.live.SetSize(cw, ch)
	m.about.SetSize(cw, ch)
	m.overview.SetSize(cw, ch)
	m.results.SetSize(cw, ch)
	m.fixtures.SetSize(cw, ch)
	m.standings.SetSize(cw, ch)
	m.bracket.SetSize(cw, ch)
	m.teams.SetSize(cw, ch)
	if m.overlay != nil {
		m.overlay.SetSize(w-10, ch)
	}
}

func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// Splash swallows keys until entry.
	if m.route == "splash" {
		var cmd tea.Cmd
		m.splash, cmd = m.splash.Update(msg)
		return m, cmd
	}

	// Overlay takes precedence when open.
	if m.overlay != nil {
		var cmd tea.Cmd
		o, c := m.overlay.Update(msg)
		*m.overlay = o
		cmd = c
		return m, cmd
	}

	key := msg.String()
	switch key {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "ctrl+r":
		m.loadRoute(m.route)
		return m, nil
	case "esc":
		m.focusNav()
		return m, nil
	}

	if m.navFocus {
		return m.handleNavKey(key)
	}
	return m.handleContentKey(msg, key)
}

// handleNavKey drives the sidebar rail: up/down preview-switch routes, enter
// focuses the content.
func (m Model) handleNavKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "up", "k":
		m.moveNav(-1)
		return m, nil
	case "down", "j":
		m.moveNav(1)
		return m, nil
	case "enter", "right":
		m.focusContent()
		return m, m.maybeLiveInit()
	}
	// Letter keys still jump from the rail.
	if route, ok := letterRoute[key]; ok {
		m.show(route)
		return m, m.maybeLiveInit()
	}
	return m, nil
}

// handleContentKey routes movement/drill keys to the focused screen, and lets
// letter keys jump elsewhere.
func (m Model) handleContentKey(msg tea.KeyPressMsg, key string) (tea.Model, tea.Cmd) {
	move := key == "j" || key == "k" || key == "up" || key == "down"
	bracketMove := move || key == "h" || key == "l" || key == "left" || key == "right"

	switch m.route {
	case "bracket":
		if key == "enter" {
			m.openDetail(m.bracket.SelectedMatchID())
			return m, nil
		}
		if bracketMove {
			m.bracket, _ = m.bracket.Update(msg)
			return m, nil
		}
	case "events":
		if move {
			m.events, _ = m.events.Update(msg)
			return m, nil
		}
		if key == "enter" {
			if id, ok := atoi(m.events.Selected()); ok {
				m.selectEvent(id, "overview")
			}
			return m, nil
		}
	case "results":
		if move {
			m.results, _ = m.results.Update(msg)
			return m, nil
		}
		if key == "enter" {
			if id, ok := atoi(m.results.Selected()); ok {
				m.openDetail(id)
			}
			return m, nil
		}
	case "standings":
		if move {
			m.standings, _ = m.standings.Update(msg)
			return m, nil
		}
	case "teams":
		if move {
			m.teams, _ = m.teams.Update(msg)
			return m, nil
		}
	}

	// Anything else: treat as a jump.
	if route, ok := letterRoute[key]; ok {
		m.show(route)
		return m, m.maybeLiveInit()
	}
	return m, nil
}

// ── routing helpers ─────────────────────────────────────────

// navItems is the ordered rail: global routes, plus event routes when focused.
func (m Model) navItems() []string {
	if m.eventID != 0 {
		return append(append([]string{}, globalRoutes...), eventRoutes...)
	}
	return globalRoutes
}

func (m *Model) moveNav(delta int) {
	items := m.navItems()
	idx := 0
	for i, r := range items {
		if r == m.route {
			idx = i
			break
		}
	}
	next := (idx + delta + len(items)) % len(items)
	m.switchContent(items[next])
}

// switchContent shows a route without moving focus (rail preview).
func (m *Model) switchContent(route string) {
	if isEventRoute(route) && m.eventID == 0 {
		return
	}
	m.route = route
	m.loadRoute(route)
}

// show jumps to a route and focuses its content (letter keys).
func (m *Model) show(route string) {
	if isEventRoute(route) && m.eventID == 0 {
		return
	}
	m.switchContent(route)
	m.focusContent()
}

func (m *Model) selectEvent(id int, tab string) {
	m.eventID = id
	if e, ok := data.EventByID(id); ok {
		m.eventName = e.Name
	}
	m.switchContent(tab)
	m.focusContent()
}

func (m *Model) focusNav() {
	m.navFocus = true
	m.blurAll()
}

func (m *Model) focusContent() {
	m.navFocus = false
	m.blurAll()
	switch m.route {
	case "events":
		m.events.Focus()
	case "results":
		m.results.Focus()
	case "standings":
		m.standings.Focus()
	case "teams":
		m.teams.Focus()
	}
}

func (m *Model) blurAll() {
	m.events.Blur()
	m.results.Blur()
	m.standings.Blur()
	m.teams.Blur()
}

// loadRoute refreshes the data for a route from the cache.
func (m *Model) loadRoute(route string) {
	switch route {
	case "home":
		m.home.Load()
	case "events":
		m.events.Load()
	case "live":
		m.live.Load()
	case "overview":
		m.overview.Load(m.eventID, m.eventName)
	case "results":
		m.results.Load(m.eventID, m.eventName)
	case "fixtures":
		m.fixtures.Load(m.eventID, m.eventName)
	case "standings":
		m.standings.Load(m.eventID, m.eventName)
	case "bracket":
		m.bracket.Load(m.eventID)
	case "teams":
		m.teams.Load(m.eventID, m.eventName)
	}
}

func (m *Model) openDetail(matchID int) {
	if matchID == 0 {
		return
	}
	_, ch := contentSize(m.w, m.h)
	md := screens.NewMatchDetail(matchID, m.w-10, ch)
	m.overlay = &md
}

// maybeLiveInit (re)starts the dashboard refresh loop when entering live.
func (m Model) maybeLiveInit() tea.Cmd {
	if m.route == "live" {
		return m.live.Init()
	}
	return nil
}

// ── view ────────────────────────────────────────────────────

func (m Model) View() tea.View {
	if m.route == "splash" {
		return altScreen(m.splash.View())
	}
	innerH := m.h - 4 // shell height inside the frame (margin 2 + border 2)

	// Match-detail overlay fills the same framed area as the shell.
	if m.overlay != nil {
		inner := lipgloss.NewStyle().
			Width(m.w-6).Height(innerH).MaxHeight(innerH).Padding(1, 2).
			Render(m.overlay.View())
		return altScreen(styles.Frame.Margin(1, 2).Render(inner))
	}

	sidebar := lipgloss.NewStyle().
		Width(sidebarTotal).Height(innerH).
		Padding(1, 2).
		BorderRight(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(styles.Border).
		Render(widgets.Sidebar(m.route, m.eventName, m.navFocus, data.Freshness()))

	content := lipgloss.NewStyle().
		Width(m.w-6-sidebarTotal).Height(innerH).MaxHeight(innerH).
		Padding(1, 2).
		Render(m.content())

	shell := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content)
	return altScreen(styles.Frame.Margin(1, 2).Render(shell))
}

func (m Model) content() string {
	switch m.route {
	case "home":
		return m.home.View()
	case "events":
		return m.events.View()
	case "live":
		return m.live.View()
	case "about":
		return m.about.View()
	case "overview":
		return m.overview.View()
	case "results":
		return m.results.View()
	case "fixtures":
		return m.fixtures.View()
	case "standings":
		return m.standings.View()
	case "bracket":
		return m.bracket.View()
	case "teams":
		return m.teams.View()
	}
	return ""
}

func altScreen(content string) tea.View {
	v := tea.NewView(content)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion     // report clicks so the rail/tables are clickable
	v.BackgroundColor = styles.BG             // force the Valorant dark navy for an immersive, on-brand look
	return v
}

// handleClick maps a mouse click to a navigation action. The layout is fixed,
// so screen coordinates map deterministically to the sidebar rail and table
// rows. (Frame margin 1 + border 1 + content padding 1 ⇒ text starts at row 3;
// table data rows begin after title/hint/blank/header ⇒ row 7.)
func (m Model) handleClick(mo tea.Mouse) (tea.Model, tea.Cmd) {
	if mo.Button != tea.MouseLeft {
		return m, nil
	}
	if m.route == "splash" {
		return m, func() tea.Msg { return screens.EnterAppMsg{} }
	}
	if m.overlay != nil {
		return m, nil
	}

	const textTop = 3
	const tableTop = 7
	sidebarRight := 2 + sidebarTotal + 1 // exclusive x of the sidebar column

	// Click in the sidebar rail → jump to that page.
	if mo.X >= 2 && mo.X < sidebarRight {
		if route, ok := widgets.SidebarRoutes(m.eventName)[mo.Y-textTop]; ok {
			m.show(route)
			return m, m.maybeLiveInit()
		}
		return m, nil
	}

	// Click in a content table → select that row (and drill where it makes sense).
	if mo.X >= sidebarRight {
		m.focusContent()
		visual := mo.Y - tableTop
		switch m.route {
		case "events":
			if k, ok := m.events.ClickVisual(visual); ok {
				if id, ok := atoi(k); ok {
					m.selectEvent(id, "overview")
				}
			}
		case "results":
			if k, ok := m.results.ClickVisual(visual); ok {
				if id, ok := atoi(k); ok {
					m.openDetail(id)
				}
			}
		case "standings":
			m.standings.ClickVisual(visual)
		case "teams":
			m.teams.ClickVisual(visual)
		}
	}
	return m, nil
}

// ── small helpers ───────────────────────────────────────────

func atoi(s string) (int, bool) {
	if s == "" {
		return 0, false
	}
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, false
		}
		n = n*10 + int(r-'0')
	}
	return n, true
}

func eventIDByName(name string) (int, bool) {
	want := strings.TrimSpace(strings.ToLower(name))
	for _, e := range data.ActiveEvents() {
		if strings.TrimSpace(strings.ToLower(e.Name)) == want {
			return e.ID, true
		}
	}
	return 0, false
}
