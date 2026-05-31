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
	panel := func(region string) string {
		card := styles.Card
		if g.regionHasLive(region) {
			card = styles.CardLive
		}
		return card.Width(half).Render(g.renderRegion(region))
	}
	top := lipgloss.JoinHorizontal(lipgloss.Top, panel("Americas"), panel("EMEA"))
	bot := lipgloss.JoinHorizontal(lipgloss.Top, panel("Pacific"), panel("China"))

	return lipgloss.JoinVertical(lipgloss.Left, title, intl, top, bot)
}

func (g GlobalLive) renderIntl() string {
	muted := lipgloss.NewStyle().Foreground(styles.Muted)
	if len(g.intl) == 0 {
		return styles.IntlBar.Width(g.w - 2).Render(
			muted.Render("★ international · no active international events"))
	}
	lines := []string{lipgloss.NewStyle().Foreground(styles.Live).Bold(true).Render("★ international")}
	for i, m := range g.intl {
		if i >= 5 {
			break
		}
		lines = append(lines, widgets.MatchLine(m))
	}
	return styles.IntlBar.Width(g.w - 2).Render(strings.Join(lines, "\n"))
}

func (g GlobalLive) renderRegion(region string) string {
	slots := g.regions[region]
	muted := lipgloss.NewStyle().Foreground(styles.Muted)
	live := lipgloss.NewStyle().Foreground(styles.Live)

	lines := []string{lipgloss.NewStyle().Foreground(styles.Text).Bold(true).Render(region)}
	if slots == nil || (len(slots.Live)+len(slots.Next)+len(slots.Recent) == 0) {
		lines = append(lines, muted.Render("no matches tracked"))
		return strings.Join(lines, "\n")
	}
	add := func(header string, hs lipgloss.Style, ms []data.MatchCard) {
		if len(ms) == 0 {
			return
		}
		lines = append(lines, hs.Render(header))
		for i, m := range ms {
			if i >= maxPerSlot {
				break
			}
			lines = append(lines, widgets.MatchLine(m))
		}
	}
	add("── live ──", live, slots.Live)
	add("── next ──", muted, slots.Next)
	add("── recent ──", muted, slots.Recent)
	return strings.Join(lines, "\n")
}

// IsLive reports whether the dashboard currently has any live match — used to
// give a region panel the accent border (RegionPanel.live in the .tcss).
func (g GlobalLive) regionHasLive(region string) bool {
	s := g.regions[region]
	return s != nil && len(s.Live) > 0
}
