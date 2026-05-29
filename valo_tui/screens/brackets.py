"""[b] brackets — ASCII double-elimination trees.

The bracket is reconstructed in :mod:`valo_tui.data.bracket`; this module lays
it out on a character grid (midpoint positioning, box-drawing connectors) and
renders it as a selectable, navigable widget."""

from __future__ import annotations

from rich.markup import escape

from textual.app import ComposeResult
from textual.binding import Binding
from textual.containers import Horizontal, VerticalScroll
from textual.reactive import reactive
from textual.screen import Screen
from textual.widgets import Footer, Label, ListItem, ListView, Static

from ..data import cache
from ..data.bracket import Bracket, BracketMatch
from ..data.models import EventCard

# Grid geometry.
BOX_W = 15          # team box: name (11) + score (4)
CONN = 7            # connector zone width between columns
COLW = BOX_W + CONN

ACCENT = "#e87a5d"
TEXT = "#c8d8e8"
MUTED = "#6e8aa6"
DIM = "#2f4a5f"
SEL_BG = "#16344a"


# ── layout ───────────────────────────────────────────────────
def _layout_section(columns) -> tuple[dict[int, tuple[int, int, int]], int]:
    """Assign (top, mid, bot) rows to each match; return positions + height."""
    pos: dict[int, tuple[int, int, int]] = {}
    height = 0
    prev = None
    for col in columns:
        winners_prev: dict[str, BracketMatch] = {}
        if prev:
            for pm in prev.matches:
                if pm.winner_name:
                    winners_prev[pm.winner_name] = pm

        guesses: dict[int, int | None] = {}
        for m in col.matches:
            anchors = [
                pos[f.match_id][1]
                for f in (winners_prev.get(m.top.name), winners_prev.get(m.bottom.name))
                if f is not None and f.match_id in pos
            ]
            guesses[m.match_id] = sum(anchors) // len(anchors) if anchors else None

        order = sorted(
            col.matches,
            key=lambda m: (
                guesses[m.match_id] is None,
                guesses[m.match_id] or 0,
                col.matches.index(m),
            ),
        )
        cur = 0
        for m in order:
            g = guesses[m.match_id]
            mid = max(cur + 1, g if g is not None else cur + 1)
            pos[m.match_id] = (mid - 1, mid, mid + 1)
            cur = mid + 3
            height = max(height, mid + 2)
        prev = col
    return pos, height


class _Grid:
    """A character grid with a parallel style grid, emitted as Rich Text."""

    def __init__(self, w: int, h: int) -> None:
        self.w, self.h = w, h
        self.ch = [[" "] * w for _ in range(h)]
        self.st = [[""] * w for _ in range(h)]

    def put(self, y: int, x: int, text: str, style: str = "") -> None:
        if not (0 <= y < self.h):
            return
        for i, c in enumerate(text):
            if 0 <= x + i < self.w:
                self.ch[y][x + i] = c
                self.st[y][x + i] = style

    def to_markup(self) -> str:
        lines = []
        for y in range(self.h):
            parts, run, run_style = [], "", ""
            for x in range(self.w):
                c, s = self.ch[y][x], self.st[y][x]
                if s != run_style:
                    parts.append(_wrap(run, run_style))
                    run, run_style = c, s
                else:
                    run += c
            parts.append(_wrap(run, run_style))
            lines.append("".join(parts).rstrip())
        return "\n".join(lines)


def _wrap(text: str, style: str) -> str:
    if not text:
        return ""
    safe = escape(text)
    return f"[{style}]{safe}[/]" if style else safe


def _box(slot, selected: bool) -> tuple[str, str]:
    name = (slot.name or "TBD")[:11]
    score = "·" if slot.score is None else str(slot.score)
    text = f"{name:<11}{score:>4}"
    if slot.winner:
        style = f"bold {ACCENT}"
    else:
        style = MUTED
    if selected:
        style = f"bold {TEXT} on {SEL_BG}"
    return text, style


def render_bracket(bracket: Bracket, selected_id: int | None) -> str:
    grid = build_grid(bracket, selected_id)
    return grid.to_markup()


def build_grid(bracket: Bracket, selected_id: int | None) -> "_Grid":
    sections = bracket.sections
    if not sections:
        g = _Grid(16, 1)
        g.put(0, 0, "no bracket data", MUTED)
        return g

    # Pre-compute per-section layout + dimensions.
    layouts = [_layout_section(s.columns) for s in sections]
    width = max((len(s.columns) * COLW for s in sections), default=COLW)
    total_h = sum(h for _, h in layouts) + 3 * len(sections)
    grid = _Grid(width + 2, total_h + 1)

    row_off = 0
    for sec, (pos, sec_h) in zip(sections, layouts):
        grid.put(row_off, 0, sec.name.upper(), f"bold {ACCENT}")
        base = row_off + 2
        prev = None
        for ci, col in enumerate(sec.columns):
            x0 = ci * COLW
            # column header
            grid.put(row_off + 1, x0, col.title[: BOX_W + 2], MUTED)
            winners_prev: dict[str, BracketMatch] = {}
            if prev:
                for pm in prev.matches:
                    if pm.winner_name:
                        winners_prev[pm.winner_name] = pm
            for m in col.matches:
                top, mid, bot = pos[m.match_id]
                sel = m.match_id == selected_id
                t_txt, t_st = _box(m.top, sel)
                b_txt, b_st = _box(m.bottom, sel)
                grid.put(base + top, x0, t_txt, t_st)
                grid.put(base + bot, x0, b_txt, b_st)
                if ci > 0:
                    _draw_conn(grid, base, x0, ci, m, mid, winners_prev, pos)
            prev = col
        row_off += sec_h + 3

    return grid


def _draw_conn(grid, base, x0, ci, m, mid, winners_prev, pos) -> None:
    feeders = [
        winners_prev.get(m.top.name),
        winners_prev.get(m.bottom.name),
    ]
    rows = [pos[f.match_id][1] for f in feeders if f is not None and f.match_id in pos]
    if not rows:
        return
    chan = x0 - 4
    prev_end = (ci - 1) * COLW + BOX_W

    # single feeder, same row -> straight line
    if len(rows) == 1 and rows[0] == mid:
        grid.put(base + mid, prev_end, "─" * (x0 - prev_end), DIM)
        return

    lo, hi = min(rows + [mid]), max(rows + [mid])
    for y in range(lo, hi + 1):
        if grid.ch[base + y][chan] == " ":
            grid.put(base + y, chan, "│", DIM)
    for fy in rows:
        grid.put(base + fy, prev_end, "─" * (chan - prev_end), DIM)
        corner = "┐" if fy < mid else ("┘" if fy > mid else "├")
        grid.put(base + fy, chan, corner, DIM)
    grid.put(base + mid, chan, "├", DIM)
    grid.put(base + mid, chan + 1, "─" * (x0 - chan - 1), DIM)


# ── widget ───────────────────────────────────────────────────
class BracketWidget(Static):
    """Renders a bracket and tracks a selected match for navigation."""

    BINDINGS = [
        Binding("j,down", "move(0, 1)", "Down", show=False),
        Binding("k,up", "move(0, -1)", "Up", show=False),
        Binding("l,right", "move(1, 0)", "Right", show=False),
        Binding("h,left", "move(-1, 0)", "Left", show=False),
        Binding("enter", "open", "Open match", show=False),
    ]
    can_focus = True
    sel_col: reactive[int] = reactive(0)
    sel_row: reactive[int] = reactive(0)

    def __init__(self, bracket: Bracket, **kwargs) -> None:
        self.bracket = bracket
        # Flatten all columns across sections for h/l navigation.
        self.columns = [c for s in bracket.sections for c in s.columns]
        first = self.columns[0].matches[0].match_id if self.columns and self.columns[0].matches else None
        grid = build_grid(bracket, first)
        self._w, self._h = grid.w, grid.h
        super().__init__(grid.to_markup(), **kwargs)
        # Explicit size avoids Textual's auto-width content measurement.
        self.styles.width = self._w
        self.styles.height = self._h

    def _selected_match(self) -> BracketMatch | None:
        if not self.columns:
            return None
        col = self.columns[self.sel_col]
        if not col.matches:
            return None
        return col.matches[self.sel_row]

    def action_move(self, dc: int, dr: int) -> None:
        if not self.columns:
            return
        if dc:
            self.sel_col = max(0, min(len(self.columns) - 1, self.sel_col + dc))
            self.sel_row = min(self.sel_row, len(self.columns[self.sel_col].matches) - 1)
        if dr:
            n = len(self.columns[self.sel_col].matches)
            self.sel_row = max(0, min(n - 1, self.sel_row + dr))
        self._redraw()

    def action_open(self) -> None:
        m = self._selected_match()
        if m and m.match_id:
            from .match_detail import MatchDetailScreen

            self.app.push_screen(MatchDetailScreen(m.match_id))

    def _redraw(self) -> None:
        m = self._selected_match()
        self.update(render_bracket(self.bracket, m.match_id if m else None))


# ── screen ───────────────────────────────────────────────────
class BracketsScreen(Screen):
    BINDINGS = [Binding("escape,q", "app.pop_screen", "Back")]

    def __init__(self, event_id: int | None = None) -> None:
        super().__init__()
        self._event_id = event_id

    def compose(self) -> ComposeResult:
        with Horizontal():
            with VerticalScroll(id="bracket-events"):
                yield Label("events", classes="page-title")
                yield ListView(id="event-list")
            with VerticalScroll(id="bracket-body"):
                yield Label("select an event  ·  enter", id="bracket-hint", classes="hint")
        yield Footer()

    def on_mount(self) -> None:
        events = cache.active_events()
        lv = self.query_one("#event-list", ListView)
        self._events: list[EventCard] = events
        for e in events:
            lv.append(ListItem(Label(e.name[:30]), id=f"ev-{e.id}"))
        # Only fetch when the user explicitly picks an event (or one was passed
        # in). We never auto-ping the API just because the screen opened.
        if self._event_id is not None:
            self._load(self._event_id)

    def on_list_view_selected(self, event: ListView.Selected) -> None:
        if event.item.id and event.item.id.startswith("ev-"):
            self._load(int(event.item.id[3:]))

    def _load(self, event_id: int) -> None:
        body = self.query_one("#bracket-body", VerticalScroll)
        body.remove_children()
        bracket = cache.bracket(event_id)
        if not bracket.has_data:
            body.mount(Label("no playoff bracket for this event yet", classes="hint"))
            return
        widget = BracketWidget(bracket)
        body.mount(widget)
        self.call_after_refresh(widget.focus)
