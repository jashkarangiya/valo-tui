// Package app is the root Bubble Tea model: the framed shell with a
// context-aware sidebar and a content switcher. It mirrors valo_tui/app.py's
// event-first information architecture — global routes are always available,
// event routes only once a tournament is in focus.
package app

import (
	"strings"
	"time"

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

	// contentScroll is the vertical line offset for the prose screens (home,
	// live, about, overview, fixtures); table screens scroll their own cursor.
	contentScroll int

	overlay *screens.MatchDetail  // non-nil ⇒ match-detail overlay is open
	roster  *screens.RosterDetail // non-nil ⇒ roster overlay is open (stacks on top)

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
	// Reserve one row at the bottom of the shell for the watermark footer.
	ch := h - 7
	if ch < 6 {
		ch = 6
	}
	return cw, ch
}

// refreshInterval is how often the visible screen is re-read from the cache so
// fetcher updates appear without manual navigation. It also keeps the "↻ Ns
// ago" freshness indicator ticking.
const refreshInterval = 15 * time.Second

type refreshTickMsg struct{}

func refreshTick() tea.Cmd {
	return tea.Tick(refreshInterval, func(time.Time) tea.Msg { return refreshTickMsg{} })
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.splash.Init(), refreshTick())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.resize(msg.Width, msg.Height)
		return m, nil

	case refreshTickMsg:
		// Re-read the visible screen from the cache, then re-arm the ticker.
		if m.route != "splash" && m.overlay == nil && m.roster == nil {
			m.loadRoute(m.route)
		}
		return m, refreshTick()

	case tea.KeyPressMsg:
		return m.handleKey(msg)

	case tea.MouseClickMsg:
		return m.handleClick(msg.Mouse())

	case tea.MouseWheelMsg:
		return m.handleWheel(msg.Mouse())

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
		// Close the topmost layer: roster sits above the match-detail overlay.
		if m.roster != nil {
			m.roster = nil
		} else {
			m.overlay = nil
		}
		return m, nil

	case screens.OpenRosterMsg:
		m.openRoster(msg.TeamName)
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
	if m.roster != nil {
		m.roster.SetSize(w-10, ch)
	}
}

func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// Splash swallows keys until entry.
	if m.route == "splash" {
		var cmd tea.Cmd
		m.splash, cmd = m.splash.Update(msg)
		return m, cmd
	}

	// Roster overlay is the topmost layer when open.
	if m.roster != nil {
		var cmd tea.Cmd
		r, c := m.roster.Update(msg)
		*m.roster = r
		cmd = c
		return m, cmd
	}

	// Match-detail overlay takes precedence over the shell.
	if m.overlay != nil {
		var cmd tea.Cmd
		o, c := m.overlay.Update(msg)
		*m.overlay = o
		cmd = c
		return m, cmd
	}

	key := msg.String()
	switch key {
	case "ctrl+c":
		return m, tea.Quit
	case "q":
		// q only quits from the rail; in content it backs out to the rail first,
		// so an accidental press never drops you out of the app.
		if m.navFocus {
			return m, tea.Quit
		}
		m.focusNav()
		return m, nil
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
	// ← is the symmetric partner of →/enter: it returns focus to the rail. The
	// bracket is exempt — ←/→ are its grid axes, so it backs out with esc/q.
	if key == "left" && m.route != "bracket" {
		m.focusNav()
		return m, nil
	}
	// Prose screens have no cursor, so movement keys scroll the body instead.
	if isScrollRoute(m.route) {
		return m.handleScrollKey(key)
	}

	move := key == "j" || key == "k" || key == "up" || key == "down" ||
		key == "pgup" || key == "pgdown" || key == "g" || key == "G" || key == "shift+g" ||
		key == "home" || key == "end" || key == "ctrl+d" || key == "ctrl+u"
	bracketMove := key == "j" || key == "k" || key == "up" || key == "down" ||
		key == "h" || key == "l" || key == "left" || key == "right"

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
		if key == "enter" {
			m.openRoster(m.standings.SelectedTeam())
			return m, nil
		}
	case "teams":
		if move {
			m.teams, _ = m.teams.Update(msg)
			return m, nil
		}
		if key == "enter" {
			m.openRoster(m.teams.SelectedTeam())
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

// ── prose scrolling ─────────────────────────────────────────

// scrollRoutes are the screens rendered as a flat block of text (no cursor), so
// the shell scrolls their body by a line offset.
var scrollRoutes = map[string]bool{
	"home": true, "live": true, "about": true, "overview": true, "fixtures": true,
}

func isScrollRoute(r string) bool { return scrollRoutes[r] }

// handleScrollKey scrolls a prose screen, or jumps away on a letter key.
func (m Model) handleScrollKey(key string) (tea.Model, tea.Cmd) {
	_, vh := contentSize(m.w, m.h)
	switch key {
	case "down", "j":
		m.scrollBy(1)
	case "up", "k":
		m.scrollBy(-1)
	case "pgdown":
		m.scrollBy(vh - 1)
	case "pgup":
		m.scrollBy(-(vh - 1))
	case "ctrl+d":
		m.scrollBy(vh / 2)
	case "ctrl+u":
		m.scrollBy(-vh / 2)
	case "home", "g":
		m.contentScroll = 0
	case "end", "G", "shift+g":
		m.contentScroll = m.maxContentScroll()
	default:
		if route, ok := letterRoute[key]; ok {
			m.show(route)
			return m, m.maybeLiveInit()
		}
	}
	return m, nil
}

// scrollBy adjusts the prose offset, clamped to [0, maxContentScroll].
func (m *Model) scrollBy(delta int) {
	m.contentScroll += delta
	if max := m.maxContentScroll(); m.contentScroll > max {
		m.contentScroll = max
	}
	if m.contentScroll < 0 {
		m.contentScroll = 0
	}
}

// maxContentScroll is how far the current prose body can scroll before its last
// line reaches the bottom of the viewport (0 when it already fits).
func (m Model) maxContentScroll() int {
	if !isScrollRoute(m.route) {
		return 0
	}
	_, vh := contentSize(m.w, m.h)
	lines := strings.Count(m.rawContent(), "\n") + 1
	if lines <= vh {
		return 0
	}
	return lines - vh
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
	if route != m.route {
		m.contentScroll = 0 // start each screen at the top
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

// openRoster opens the roster overlay for a team name (stacking above any
// match-detail overlay it was launched from).
func (m *Model) openRoster(name string) {
	if name == "" || name == "TBD" {
		return
	}
	_, ch := contentSize(m.w, m.h)
	rd := screens.NewRosterDetail(name, m.w-10, ch)
	m.roster = &rd
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
	innerH := m.h - 4   // shell height inside the frame (margin 2 + border 2)
	bodyH := innerH - 1 // reserve the last row for the watermark footer

	// Overlays fill the same framed area as the shell. Roster sits on top.
	if m.roster != nil {
		return m.frame(m.overlayBox(m.roster.View(), bodyH))
	}
	if m.overlay != nil {
		return m.frame(m.overlayBox(m.overlay.View(), bodyH))
	}

	sidebar := lipgloss.NewStyle().
		Width(sidebarTotal).Height(bodyH).
		Padding(1, 2).
		BorderRight(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(styles.Border).
		Render(widgets.Sidebar(m.route, m.eventName, m.navFocus, cacheHealth()))

	content := lipgloss.NewStyle().
		Width(m.w-6-sidebarTotal).Height(bodyH).MaxHeight(bodyH).
		Padding(1, 2).
		Render(m.content())

	shell := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content)
	return m.frame(shell)
}

// watermark is the small maker's mark in the bottom-right of every framed view.
const watermark = "blackpantha"

// keyHints is the always-on cheat sheet shown at the foot of the shell, so the
// global navigation keys are discoverable without opening About.
const keyHints = "←/→ focus · ↑/↓ move · enter open · esc back · q quit"

// frame wraps a shell body in the rounded frame, with the key hints left-
// aligned and the watermark right-aligned on a shared footer row just inside
// the bottom border.
func (m Model) frame(body string) tea.View {
	w := m.w - 6
	hint := lipgloss.NewStyle().Foreground(styles.Muted).Render(keyHints)
	mark := lipgloss.NewStyle().Foreground(styles.Muted).Render(watermark)
	// Drop the hints first when the frame is too narrow to fit both.
	footer := mark
	if gap := w - lipgloss.Width(hint) - lipgloss.Width(mark); gap >= 1 {
		footer = hint + strings.Repeat(" ", gap) + mark
	} else {
		footer = lipgloss.NewStyle().Width(w).Align(lipgloss.Right).Render(mark)
	}
	shell := lipgloss.JoinVertical(lipgloss.Left, body, footer)
	return altScreen(styles.Frame.Margin(1, 2).Render(shell))
}

// cacheHealth gathers the freshness/fetcher state for the rail footer.
func cacheHealth() widgets.Health {
	fresh, stale := data.FreshnessState()
	errMsg, recent := data.FetchError()
	if !recent {
		errMsg = ""
	}
	return widgets.Health{Freshness: fresh, Stale: stale, FetchErr: errMsg}
}

// overlayBox renders overlay content into the same padded framed area as the
// shell, so match-detail and roster overlays share one layout (and one
// click-coordinate origin).
func (m Model) overlayBox(content string, innerH int) string {
	return lipgloss.NewStyle().
		Width(m.w-6).Height(innerH).MaxHeight(innerH).Padding(1, 2).
		Render(content)
}

// content is the body for the active route, with the prose screens sliced to
// the scroll window. Table screens scroll their own cursor, so they pass
// through unchanged.
func (m Model) content() string {
	body := m.rawContent()
	if isScrollRoute(m.route) {
		_, vh := contentSize(m.w, m.h)
		body = scrollView(body, m.contentScroll, vh)
	}
	return body
}

// scrollView returns the height-row window of s starting at off (clamped).
func scrollView(s string, off, height int) string {
	if height <= 0 {
		return s
	}
	lines := strings.Split(s, "\n")
	if off > len(lines)-height {
		off = len(lines) - height
	}
	if off < 0 {
		off = 0
	}
	end := off + height
	if end > len(lines) {
		end = len(lines)
	}
	return strings.Join(lines[off:end], "\n")
}

func (m Model) rawContent() string {
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
	v.MouseMode = tea.MouseModeCellMotion // report clicks so the rail/tables are clickable
	v.BackgroundColor = styles.BG         // force the Valorant dark navy for an immersive, on-brand look
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
	// Roster overlay (topmost) swallows clicks; any click dismisses nothing —
	// use esc.
	if m.roster != nil {
		return m, nil
	}
	// Match-detail overlay: a click on a header team name opens its roster.
	// Overlay text origin = frame margin 2 + border 1 + padding 2 = x5, y3.
	if m.overlay != nil {
		if name, ok := m.overlay.TeamAt(mo.X-5, mo.Y-3); ok {
			m.openRoster(name)
		}
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

	// Click in the content area.
	if mo.X >= sidebarRight {
		// Dashboards render team names mid-line; a click on one opens its roster.
		// Their View origin is the content text cell (sidebar + content padding).
		contentLeft := sidebarRight + 2
		contentY := mo.Y - textTop + m.contentScroll // add any prose scroll offset
		switch m.route {
		case "home":
			if name, ok := m.home.TeamAt(mo.X-contentLeft, contentY); ok {
				m.openRoster(name)
				return m, nil
			}
		case "live":
			if name, ok := m.live.TeamAt(mo.X-contentLeft, contentY); ok {
				m.openRoster(name)
				return m, nil
			}
		}

		// Otherwise: select that table row (and drill where it makes sense).
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
			if name, ok := m.standings.ClickVisual(visual); ok {
				m.openRoster(name)
			}
		case "teams":
			if name, ok := m.teams.ClickVisual(visual); ok {
				m.openRoster(name)
			}
		}
	}
	return m, nil
}

// wheelStep is how many rows/lines one wheel notch scrolls.
const wheelStep = 3

// handleWheel scrolls whatever sits under the pointer: the rail previews
// routes, a prose screen scrolls its body, a table moves its cursor.
func (m Model) handleWheel(mo tea.Mouse) (tea.Model, tea.Cmd) {
	if m.route == "splash" || m.overlay != nil || m.roster != nil {
		return m, nil
	}
	dir := wheelStep
	if mo.Button == tea.MouseWheelUp {
		dir = -wheelStep
	}

	sidebarRight := 2 + sidebarTotal + 1
	if mo.X >= 2 && mo.X < sidebarRight {
		m.moveNav(sign(dir))
		return m, nil
	}

	if isScrollRoute(m.route) {
		m.scrollBy(dir)
		return m, nil
	}

	// Table / bracket screens: drive the cursor. Focus follows the wheel so the
	// selection is visible.
	m.focusContent()
	switch m.route {
	case "events":
		m.events.MoveCursor(dir)
	case "results":
		m.results.MoveCursor(dir)
	case "standings":
		m.standings.MoveCursor(dir)
	case "teams":
		m.teams.MoveCursor(dir)
	case "bracket":
		m.bracket.MoveCursor(sign(dir))
	}
	return m, nil
}

// sign collapses a delta to -1 or +1 (wheel notch → one rail/grid step).
func sign(d int) int {
	if d < 0 {
		return -1
	}
	return 1
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
