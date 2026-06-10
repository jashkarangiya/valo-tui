package screens

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/jashkarangiya/valo-tui/internal/data"
	"github.com/jashkarangiya/valo-tui/internal/styles"
	"github.com/jashkarangiya/valo-tui/internal/widgets"
)

const maxPerSlot = 4

// globalLiveData is the result of one cache read, delivered as a tea.Msg.
type globalLiveData struct {
	regions map[string]*data.RegionSlots
	intl    []data.MatchCard
}

type globalLiveTick struct{}

// GlobalLive is the [l] global-live dashboard: a 2×2 Bento of the four leagues
// with international events pinned across the top.
type GlobalLive struct {
	w, h    int
	regions map[string]*data.RegionSlots
	intl    []data.MatchCard
}

// NewGlobalLive builds an empty dashboard; data arrives via Init's fetch.
func NewGlobalLive(w, h int) GlobalLive {
	return GlobalLive{w: w, h: h, regions: map[string]*data.RegionSlots{}}
}

func (g *GlobalLive) SetSize(w, h int) { g.w, g.h = w, h }

// Load synchronously refreshes the dashboard from the cache (used on entry and
// for ctrl+r); the periodic refresh still runs via Init/Update.
func (g *GlobalLive) Load() {
	g.regions, g.intl = data.GlobalLive()
}

func (g GlobalLive) Init() tea.Cmd {
	return tea.Batch(fetchGlobalLive(), tickGlobalLive())
}

func fetchGlobalLive() tea.Cmd {
	return func() tea.Msg {
		regions, intl := data.GlobalLive()
		return globalLiveData{regions: regions, intl: intl}
	}
}

func tickGlobalLive() tea.Cmd {
	return tea.Tick(30*time.Second, func(time.Time) tea.Msg { return globalLiveTick{} })
}

func (g GlobalLive) Update(msg tea.Msg) (GlobalLive, tea.Cmd) {
	switch msg := msg.(type) {
	case globalLiveData:
		g.regions, g.intl = msg.regions, msg.intl
		return g, nil
	case globalLiveTick:
		return g, tea.Batch(fetchGlobalLive(), tickGlobalLive())
	}
	return g, nil
}

func (g GlobalLive) View() string {
	title := styles.PageTitle.Render("global live")
	intl := g.renderIntl()

	half := g.w/2 - 1
	if half < 10 {
		half = 10
	}
	// Even 2×2 grid: each card the same width and height (mirrors v1's grid).
	rowH := (g.h - 5) / 2
	if rowH < 4 {
		rowH = 4
	}
	panel := func(region string) string {
		card := styles.Card
		if g.regionHasLive(region) {
			card = styles.CardLive
		}
		return card.Width(half).Height(rowH).Render(g.renderRegion(region))
	}
	top := lipgloss.JoinHorizontal(lipgloss.Top, panel("Americas"), panel("EMEA"))
	bot := lipgloss.JoinHorizontal(lipgloss.Top, panel("Pacific"), panel("China"))

	return lipgloss.JoinVertical(lipgloss.Left, title, intl, top, bot)
}

// liveLine is one rendered dashboard line plus the match it shows (nil for
// headers), so clicks can be mapped back to a team.
type liveLine struct {
	text  string
	match *data.MatchCard
}

// joinLines stacks the lines, truncating each to w columns so none wraps —
// wrapping would desync the visual rows from the logical lines that TeamAt's
// hit-testing relies on (and orphaned "PM" fragments look bad anyway).
func joinLines(ls []liveLine, w int) string {
	trunc := lipgloss.NewStyle().MaxWidth(w)
	texts := make([]string, len(ls))
	for i, l := range ls {
		texts[i] = trunc.Render(l.text)
	}
	return strings.Join(texts, "\n")
}

// regionContentW is the text width inside a region panel: half minus its border
// (2) and padding (2).
func (g GlobalLive) regionContentW() int {
	half := g.w/2 - 1
	if half < 10 {
		half = 10
	}
	if w := half - 4; w > 1 {
		return w
	}
	return 1
}

func (g GlobalLive) intlLines() []liveLine {
	if len(g.intl) == 0 {
		return []liveLine{{text: lipgloss.NewStyle().Foreground(styles.Muted).
			Render("★ international · no active international events")}}
	}
	out := []liveLine{{text: lipgloss.NewStyle().Foreground(styles.Live).Bold(true).Render("★ international")}}
	for i, m := range g.intl {
		if i >= 5 {
			break
		}
		mc := m
		out = append(out, liveLine{text: widgets.MatchLine(m), match: &mc})
	}
	return out
}

func (g GlobalLive) renderIntl() string {
	return styles.IntlBar.Width(g.w - 2).Render(joinLines(g.intlLines(), g.w-6))
}

func (g GlobalLive) regionLines(region string) []liveLine {
	slots := g.regions[region]
	muted := lipgloss.NewStyle().Foreground(styles.Muted)
	live := lipgloss.NewStyle().Foreground(styles.Live)

	out := []liveLine{{text: lipgloss.NewStyle().Foreground(styles.Text).Bold(true).Render(region)}}
	if slots == nil || (len(slots.Live)+len(slots.Next)+len(slots.Recent) == 0) {
		out = append(out, liveLine{text: muted.Render("no matches tracked")})
		return out
	}
	add := func(header string, hs lipgloss.Style, ms []data.MatchCard) {
		if len(ms) == 0 {
			return
		}
		out = append(out, liveLine{text: hs.Render(header)})
		for i, m := range ms {
			if i >= maxPerSlot {
				break
			}
			mc := m
			out = append(out, liveLine{text: widgets.MatchLine(m), match: &mc})
		}
	}
	add("── live ──", live, slots.Live)
	add("── next ──", muted, slots.Next)
	add("── recent ──", muted, slots.Recent)
	return out
}

func (g GlobalLive) renderRegion(region string) string {
	return joinLines(g.regionLines(region), g.regionContentW())
}

// boxContent is the (x, y) offset to a bordered box's first content cell:
// RoundedBorder (1) + Padding(0,1) ⇒ left 2, top 1. Card and IntlBar share it.
const (
	boxContentX = 2
	boxContentY = 1
)

// TeamAt maps a View-local (x, y) click to the team name under it, mirroring
// View()'s layout. Heights/widths are measured from the rendered boxes so this
// stays correct without re-deriving lipgloss's border/padding math.
func (g GlobalLive) TeamAt(x, y int) (string, bool) {
	titleH := lipgloss.Height(styles.PageTitle.Render("global live"))
	intlH := lipgloss.Height(g.renderIntl())

	// International bar.
	if y >= titleH && y < titleH+intlH {
		return hitLines(g.intlLines(), x-boxContentX, y-titleH-boxContentY)
	}

	half := g.w/2 - 1
	if half < 10 {
		half = 10
	}
	rowH := (g.h - 5) / 2
	if rowH < 4 {
		rowH = 4
	}
	panel := func(region string) string {
		card := styles.Card
		if g.regionHasLive(region) {
			card = styles.CardLive
		}
		return card.Width(half).Height(rowH).Render(g.renderRegion(region))
	}
	leftW := lipgloss.Width(panel("Americas"))
	panelH := lipgloss.Height(panel("Americas"))
	topY := titleH + intlH

	hitRow := func(left, right string, baseY int) (string, bool) {
		region, ox := left, 0
		if x >= leftW {
			region, ox = right, leftW
		}
		return hitLines(g.regionLines(region), x-ox-boxContentX, y-baseY-boxContentY)
	}
	switch {
	case y >= topY && y < topY+panelH:
		return hitRow("Americas", "EMEA", topY)
	case y >= topY+panelH && y < topY+2*panelH:
		return hitRow("Pacific", "China", topY+panelH)
	}
	return "", false
}

// hitLines returns the team name at content-local (col, row) within a list of
// rendered lines, or ok=false if the row isn't a match line / col misses a name.
func hitLines(lines []liveLine, col, row int) (string, bool) {
	if row < 0 || row >= len(lines) || lines[row].match == nil {
		return "", false
	}
	return widgets.MatchLineHit(*lines[row].match, col)
}

// IsLive reports whether the dashboard currently has any live match — used to
// give a region panel the accent border (RegionPanel.live in the .tcss).
func (g GlobalLive) regionHasLive(region string) bool {
	s := g.regions[region]
	return s != nil && len(s.Live) > 0
}
