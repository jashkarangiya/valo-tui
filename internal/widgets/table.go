package widgets

import (
	"strings"
	"unicode/utf8"

	"charm.land/lipgloss/v2"

	"github.com/jashkarangiya/valo-tui/internal/styles"
)

// Column is a table column header + fixed width.
type Column struct {
	Title string
	Width int
}

// Cell is one styled table cell.
type Cell struct {
	Text  string
	Style lipgloss.Style
}

// Row is a table row keyed by id (the value drilled into on Enter).
type Row struct {
	Key   string
	Cells []Cell
}

// Table is a minimal DataTable analog: a cursor over keyed rows with j/k
// movement and a scroll window. Mirrors the VimDataTable behaviour from v1.
type Table struct {
	Cols    []Column
	rows    []Row
	cursor  int
	focused bool
	height  int // visible body rows; 0 = unbounded
}

var (
	tblHeader = lipgloss.NewStyle().Foreground(styles.Muted)
	tblCursor = lipgloss.NewStyle().Foreground(styles.BG).Background(styles.Accent).Bold(true)
)

// NewTable builds a table with the given columns.
func NewTable(cols ...Column) Table { return Table{Cols: cols} }

// SetRows replaces the rows, clamping the cursor into range.
func (t *Table) SetRows(rows []Row) {
	t.rows = rows
	if t.cursor > len(rows)-1 {
		t.cursor = len(rows) - 1
	}
	if t.cursor < 0 {
		t.cursor = 0
	}
}

// SetHeight sets the visible body-row count (for scrolling).
func (t *Table) SetHeight(h int) { t.height = h }

// MoveCursor moves the selection by delta, clamped.
func (t *Table) MoveCursor(delta int) {
	if len(t.rows) == 0 {
		return
	}
	t.cursor += delta
	if t.cursor < 0 {
		t.cursor = 0
	}
	if t.cursor > len(t.rows)-1 {
		t.cursor = len(t.rows) - 1
	}
}

// ClickVisual moves the cursor to the visually-nth body row (accounting for the
// scroll window) and returns its key. ok is false if the click missed a row.
func (t *Table) ClickVisual(visual int) (string, bool) {
	if visual < 0 || len(t.rows) == 0 {
		return "", false
	}
	start, end := t.window()
	row := start + visual
	if row < start || row >= end {
		return "", false
	}
	t.cursor = row
	return t.rows[row].Key, true
}

// SelectedKey returns the key of the cursor row, or "".
func (t Table) SelectedKey() string {
	if t.cursor < 0 || t.cursor >= len(t.rows) {
		return ""
	}
	return t.rows[t.cursor].Key
}

func (t *Table) Focus() { t.focused = true }
func (t *Table) Blur()  { t.focused = false }
func (t Table) Len() int { return len(t.rows) }

func (t Table) window() (int, int) {
	h := t.height
	if h <= 0 || h >= len(t.rows) {
		return 0, len(t.rows)
	}
	start := t.cursor - h/2
	if start < 0 {
		start = 0
	}
	if start+h > len(t.rows) {
		start = len(t.rows) - h
	}
	return start, start + h
}

func (t Table) View() string {
	var b strings.Builder

	// header
	var hdr strings.Builder
	for _, c := range t.Cols {
		hdr.WriteString(tblHeader.Render(padTrunc(c.Title, c.Width)) + " ")
	}
	b.WriteString(strings.TrimRight(hdr.String(), " "))
	b.WriteString("\n")

	start, end := t.window()
	for idx := start; idx < end; idx++ {
		row := t.rows[idx]
		cursor := idx == t.cursor && t.focused
		var line strings.Builder
		for ci, c := range t.Cols {
			text, stl := "", lipgloss.NewStyle()
			if ci < len(row.Cells) {
				text = row.Cells[ci].Text
				stl = row.Cells[ci].Style
			}
			padded := padTrunc(text, c.Width)
			if cursor {
				line.WriteString(tblCursor.Render(padded) + tblCursor.Render(" "))
			} else {
				line.WriteString(stl.Render(padded) + " ")
			}
		}
		b.WriteString(strings.TrimRight(line.String(), " "))
		if idx < end-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

// padTrunc left-aligns text to exactly w display columns.
func padTrunc(s string, w int) string {
	n := utf8.RuneCountInString(s)
	if n == w {
		return s
	}
	if n < w {
		return s + strings.Repeat(" ", w-n)
	}
	r := []rune(s)
	if w <= 1 {
		return string(r[:w])
	}
	return string(r[:w-1]) + "…"
}
