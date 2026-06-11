package screens

import (
	"fmt"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/jashkarangiya/valo-tui/internal/data"
	"github.com/jashkarangiya/valo-tui/internal/styles"
)

// Grid geometry (matches screens/brackets.py).
const (
	boxW = 15 // team box: name(11) + score(4)
	conn = 7  // connector zone between columns
	colW = boxW + conn
)

// Cell style tags for the bracket grid.
type cellTag int

const (
	tagPlain cellTag = iota
	tagWinner
	tagMuted
	tagDim
	tagSelected
	tagSection
)

var bracketStyles = map[cellTag]lipgloss.Style{
	tagPlain:    lipgloss.NewStyle().Foreground(styles.Text),
	tagWinner:   lipgloss.NewStyle().Foreground(styles.Accent).Bold(true),
	tagMuted:    lipgloss.NewStyle().Foreground(styles.Muted),
	tagDim:      lipgloss.NewStyle().Foreground(styles.Rule),
	tagSelected: lipgloss.NewStyle().Foreground(styles.Text).Background(styles.SelBg).Bold(true),
	tagSection:  lipgloss.NewStyle().Foreground(styles.Accent).Bold(true),
}

// grid is a character grid with a parallel style-tag grid.
type grid struct {
	w, h int
	ch   [][]rune
	tag  [][]cellTag
}

func newGrid(w, h int) *grid {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	ch := make([][]rune, h)
	tg := make([][]cellTag, h)
	for y := 0; y < h; y++ {
		ch[y] = make([]rune, w)
		tg[y] = make([]cellTag, w)
		for x := 0; x < w; x++ {
			ch[y][x] = ' '
		}
	}
	return &grid{w: w, h: h, ch: ch, tag: tg}
}

func (g *grid) put(y, x int, s string, tag cellTag) {
	if y < 0 || y >= g.h {
		return
	}
	for i, c := range []rune(s) {
		if x+i >= 0 && x+i < g.w {
			g.ch[y][x+i] = c
			g.tag[y][x+i] = tag
		}
	}
}

func (g *grid) String() string {
	lines := make([]string, g.h)
	for y := 0; y < g.h; y++ {
		var b strings.Builder
		run, runTag := strings.Builder{}, g.tag[y][0]
		flush := func() {
			if run.Len() > 0 {
				b.WriteString(bracketStyles[runTag].Render(run.String()))
				run.Reset()
			}
		}
		for x := 0; x < g.w; x++ {
			t := g.tag[y][x]
			if t != runTag {
				flush()
				runTag = t
			}
			run.WriteRune(g.ch[y][x])
		}
		flush()
		lines[y] = strings.TrimRight(b.String(), " ")
	}
	return strings.Join(lines, "\n")
}

type rowPos struct{ top, mid, bot int }

// layoutSection assigns (top,mid,bot) rows to each match; returns positions and
// the section height. Ported from _layout_section.
func layoutSection(columns []data.BracketColumn) (map[int]rowPos, int) {
	pos := map[int]rowPos{}
	height := 0
	var prev *data.BracketColumn
	for ci := range columns {
		col := columns[ci]
		winnersPrev := map[string]data.BracketMatch{}
		if prev != nil {
			for _, pm := range prev.Matches {
				if w := pm.WinnerName(); w != "" {
					winnersPrev[w] = pm
				}
			}
		}

		guesses := map[int]int{} // match_id → guessed mid
		hasGuess := map[int]bool{}
		for _, m := range col.Matches {
			var anchors []int
			for _, name := range []string{m.Top.Name, m.Bottom.Name} {
				if f, ok := winnersPrev[name]; ok {
					if p, ok := pos[f.MatchID]; ok {
						anchors = append(anchors, p.mid)
					}
				}
			}
			if len(anchors) > 0 {
				sum := 0
				for _, a := range anchors {
					sum += a
				}
				guesses[m.MatchID] = sum / len(anchors)
				hasGuess[m.MatchID] = true
			}
		}

		order := append([]data.BracketMatch(nil), col.Matches...)
		idxOf := map[int]int{}
		for i, m := range col.Matches {
			idxOf[m.MatchID] = i
		}
		sort.SliceStable(order, func(a, b int) bool {
			ga, gb := hasGuess[order[a].MatchID], hasGuess[order[b].MatchID]
			if ga != gb {
				return ga // those with a guess first
			}
			if guesses[order[a].MatchID] != guesses[order[b].MatchID] {
				return guesses[order[a].MatchID] < guesses[order[b].MatchID]
			}
			return idxOf[order[a].MatchID] < idxOf[order[b].MatchID]
		})

		cur := 0
		for _, m := range order {
			mid := cur + 1
			if hasGuess[m.MatchID] && guesses[m.MatchID] > mid {
				mid = guesses[m.MatchID]
			}
			pos[m.MatchID] = rowPos{mid - 1, mid, mid + 1}
			cur = mid + 3
			if mid+2 > height {
				height = mid + 2
			}
		}
		prev = &columns[ci]
	}
	return pos, height
}

func bracketBox(slot data.BracketSlot, selected bool) (string, cellTag) {
	name := clipRunes(slot.Name, 11)
	if name == "" {
		name = "TBD"
	}
	score := "·"
	if slot.Score != nil {
		score = fmt.Sprint(*slot.Score)
	}
	txt := fmt.Sprintf("%-11s%4s", name, score)
	tag := tagMuted
	if slot.Winner {
		tag = tagWinner
	}
	if selected {
		tag = tagSelected
	}
	return txt, tag
}

func buildBracketGrid(b data.Bracket, selectedID int) *grid {
	if len(b.Sections) == 0 {
		g := newGrid(16, 1)
		g.put(0, 0, "no bracket data", tagMuted)
		return g
	}
	type secLayout struct {
		pos    map[int]rowPos
		height int
	}
	layouts := make([]secLayout, len(b.Sections))
	width := colW
	totalH := 0
	for i, s := range b.Sections {
		pos, h := layoutSection(s.Columns)
		layouts[i] = secLayout{pos, h}
		if w := len(s.Columns) * colW; w > width {
			width = w
		}
		totalH += h
	}
	totalH += 3 * len(b.Sections)
	g := newGrid(width+2, totalH+1)

	rowOff := 0
	for si, sec := range b.Sections {
		lay := layouts[si]
		g.put(rowOff, 0, strings.ToUpper(sec.Name), tagSection)
		base := rowOff + 2
		var prev *data.BracketColumn
		for ci := range sec.Columns {
			col := sec.Columns[ci]
			x0 := ci * colW
			g.put(rowOff+1, x0, clipRunes(col.Title, boxW+2), tagMuted)
			winnersPrev := map[string]data.BracketMatch{}
			if prev != nil {
				for _, pm := range prev.Matches {
					if w := pm.WinnerName(); w != "" {
						winnersPrev[w] = pm
					}
				}
			}
			for _, m := range col.Matches {
				p := lay.pos[m.MatchID]
				sel := m.MatchID == selectedID
				tTxt, tTag := bracketBox(m.Top, sel)
				bTxt, bTag := bracketBox(m.Bottom, sel)
				g.put(base+p.top, x0, tTxt, tTag)
				g.put(base+p.bot, x0, bTxt, bTag)
				if ci > 0 {
					drawConn(g, base, x0, ci, m, p.mid, winnersPrev, lay.pos)
				}
			}
			prev = &sec.Columns[ci]
		}
		rowOff += lay.height + 3
	}
	return g
}

func drawConn(g *grid, base, x0, ci int, m data.BracketMatch, mid int, winnersPrev map[string]data.BracketMatch, pos map[int]rowPos) {
	var rows []int
	for _, name := range []string{m.Top.Name, m.Bottom.Name} {
		if f, ok := winnersPrev[name]; ok {
			if p, ok := pos[f.MatchID]; ok {
				rows = append(rows, p.mid)
			}
		}
	}
	if len(rows) == 0 {
		return
	}
	chan_ := x0 - 4
	prevEnd := (ci-1)*colW + boxW

	if len(rows) == 1 && rows[0] == mid {
		g.put(base+mid, prevEnd, strings.Repeat("─", x0-prevEnd), tagDim)
		return
	}
	lo, hi := mid, mid
	for _, r := range rows {
		if r < lo {
			lo = r
		}
		if r > hi {
			hi = r
		}
	}
	for y := lo; y <= hi; y++ {
		if base+y >= 0 && base+y < g.h && g.ch[base+y][chan_] == ' ' {
			g.put(base+y, chan_, "│", tagDim)
		}
	}
	for _, fy := range rows {
		g.put(base+fy, prevEnd, strings.Repeat("─", chan_-prevEnd), tagDim)
		corner := "├"
		if fy < mid {
			corner = "┐"
		} else if fy > mid {
			corner = "┘"
		}
		g.put(base+fy, chan_, corner, tagDim)
	}
	g.put(base+mid, chan_, "├", tagDim)
	g.put(base+mid, chan_+1, strings.Repeat("─", x0-chan_-1), tagDim)
}

// Bracket is the [b] event sub-page: the playoff tree, navigable with h/j/k/l.
type Bracket struct {
	w, h    int
	bracket data.Bracket
	columns []data.BracketColumn // flattened for h/l navigation
	selCol  int
	selRow  int
	hasEvt  bool
}

func NewBracket(w, h int) Bracket { return Bracket{w: w, h: h} }

func (s *Bracket) SetSize(w, h int) { s.w, s.h = w, h }

func (s *Bracket) Load(eventID int) {
	s.hasEvt = eventID != 0
	if !s.hasEvt {
		s.bracket = data.Bracket{}
		s.columns = nil
		s.selCol, s.selRow = 0, 0
		return
	}
	s.bracket = data.BracketFor(eventID)
	s.columns = nil
	for _, sec := range s.bracket.Sections {
		s.columns = append(s.columns, sec.Columns...)
	}
	// Preserve the cursor across periodic reloads, clamped to the new bounds.
	s.selCol = clampIndex(s.selCol, len(s.columns))
	if s.selCol < len(s.columns) {
		s.selRow = clampIndex(s.selRow, len(s.columns[s.selCol].Matches))
	}
}

// clampIndex keeps i within [0, n-1], returning 0 for an empty collection.
func clampIndex(i, n int) int {
	if i < 0 || n == 0 {
		return 0
	}
	if i > n-1 {
		return n - 1
	}
	return i
}

func (s *Bracket) selectedMatch() (data.BracketMatch, bool) {
	if s.selCol >= len(s.columns) {
		return data.BracketMatch{}, false
	}
	col := s.columns[s.selCol]
	if s.selRow >= len(col.Matches) {
		return data.BracketMatch{}, false
	}
	return col.Matches[s.selRow], true
}

// SelectedMatchID is the match the cursor is on (0 if none).
func (s Bracket) SelectedMatchID() int {
	if m, ok := s.selectedMatch(); ok {
		return m.MatchID
	}
	return 0
}

func (s *Bracket) Focus() {}
func (s *Bracket) Blur()  {}

func (s Bracket) Update(msg tea.Msg) (Bracket, tea.Cmd) {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok || len(s.columns) == 0 {
		return s, nil
	}
	switch k.String() {
	case "l", "right":
		if s.selCol < len(s.columns)-1 {
			s.selCol++
		}
		if n := len(s.columns[s.selCol].Matches); s.selRow > n-1 {
			s.selRow = n - 1
		}
	case "h", "left":
		if s.selCol > 0 {
			s.selCol--
		}
		if n := len(s.columns[s.selCol].Matches); s.selRow > n-1 {
			s.selRow = n - 1
		}
	case "j", "down":
		if n := len(s.columns[s.selCol].Matches); s.selRow < n-1 {
			s.selRow++
		}
	case "k", "up":
		if s.selRow > 0 {
			s.selRow--
		}
	}
	return s, nil
}

func (s Bracket) View() string {
	header := title("bracket") + "\n" + hint("h/j/k/l move · enter → scoreboards") + "\n\n"
	if !s.hasEvt {
		return header + muted("select an event first")
	}
	if !s.bracket.HasData() {
		return header + muted("no playoff bracket for this event yet")
	}
	return header + buildBracketGrid(s.bracket, s.SelectedMatchID()).String()
}
