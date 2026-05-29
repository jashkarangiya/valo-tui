"""Match detail v2 — series header, map vetoes, and per-map sections with
win-bars, round-momentum sparklines, agent-role icons, and skeleton rows for
maps that haven't started.

The series fetch can hit the network (read-through cache), so it runs in a
worker thread to keep the UI responsive."""

from __future__ import annotations

from rich.text import Text

from textual import work
from textual.app import ComposeResult
from textual.binding import Binding
from textual.containers import Horizontal, VerticalScroll
from textual.screen import Screen
from textual.widgets import Footer, Label

from ..data import cache
from ..data.models import MapScore, SeriesDetail
from ..style import bars
from ..style.icons import agent_glyph, map_icon
from .widgets import LIVE, MUTED, TEXT, LiveDot, SkeletonRow, VimDataTable

_COLS = [
    ("player", 16),
    ("agent", 13),
    ("acs", 5),
    ("k", 4),
    ("d", 4),
    ("a", 4),
    ("adr", 5),
    ("hs%", 5),
    ("fk", 4),
    ("fd", 4),
]


class MatchDetailScreen(Screen):
    BINDINGS = [
        Binding("escape,q", "app.pop_screen", "Back"),
        Binding("j", "scroll_down", "Down", show=False),
        Binding("k", "scroll_up", "Up", show=False),
    ]

    def __init__(self, match_id: int) -> None:
        super().__init__()
        self.match_id = match_id

    def compose(self) -> ComposeResult:
        with VerticalScroll(id="detail-scroll"):
            yield SkeletonRow(cells=6)
            yield Label(f"[{MUTED}]loading match {self.match_id}…[/]")
        yield Footer()

    def on_mount(self) -> None:
        self._load()

    @work(thread=True, exclusive=True)
    def _load(self) -> None:
        detail = cache.series_detail(self.match_id)
        self.app.call_from_thread(self._render_detail, detail)

    def _render_detail(self, detail: SeriesDetail | None) -> None:
        body = self.query_one("#detail-scroll", VerticalScroll)
        body.remove_children()
        if detail is None:
            body.mount(Label(f"[{MUTED}]no detail available for this match[/]"))
            return

        self._mount_header(body, detail)
        if detail.vetoes:
            body.mount(Label(self._vetoes(detail), classes="veto"))

        maps = [m for m in detail.maps if not m.is_aggregate]
        if not maps:
            body.mount(Label(f"[{MUTED}]no map data yet[/]"))
            return

        for m in maps:
            self._mount_map(body, detail, m)

    # ── header ───────────────────────────────────────────────
    def _mount_header(self, body: VerticalScroll, d: SeriesDetail) -> None:
        s1 = d.team1.score if d.team1.score is not None else "–"
        s2 = d.team2.score if d.team2.score is not None else "–"
        title = (
            f"[bold {TEXT}]{d.team1.name}[/]  [{LIVE}]{s1} – {s2}[/]  "
            f"[bold {TEXT}]{d.team2.name}[/]"
        )
        bo = f"  ·  [{MUTED}]{d.best_of}[/]" if d.best_of else ""
        if d.is_live:
            status_widget = LiveDot("LIVE")
        elif d.is_completed:
            status_widget = Label(f"[{MUTED}]✓ final[/]")
        else:
            status = d.remaining or d.status_note or "upcoming"
            status_widget = Label(f"[{LIVE}]○ {status}[/]")
        body.mount(
            Horizontal(
                status_widget,
                Label(f"  {title}{bo}"),
                classes="series-header",
            )
        )
        body.mount(Label(f"[{MUTED}]{d.event} · {d.phase}[/]", classes="detail-sub"))

    def _vetoes(self, d: SeriesDetail) -> str:
        parts = []
        for v in d.vetoes:
            verb = v.action.lower()
            icon = map_icon(v.map)
            if verb == "ban":
                parts.append(f"[{MUTED}]✖ {v.team} ban {icon} {v.map}[/]")
            elif verb == "pick":
                parts.append(f"[{TEXT}]➤ {v.team} pick {icon} {v.map}[/]")
            else:
                parts.append(f"[{MUTED}]➤ {icon} {v.map} (decider)[/]")
        return "veto:  " + "   ".join(parts)

    # ── per-map section ──────────────────────────────────────
    def _mount_map(self, body: VerticalScroll, d: SeriesDetail, m: MapScore) -> None:
        body.mount(Label(self._map_header(d, m), classes="map-title"))
        if m.state == "pending":
            for _ in range(3):
                body.mount(SkeletonRow(cells=5))
            return
        if m.rounds:
            body.mount(
                Label(f"  {bars.momentum(m.rounds, m.team1_short)}", classes="momentum")
            )
        body.mount(self._map_table(m))

    def _map_header(self, d: SeriesDetail, m: MapScore) -> str:
        icon = map_icon(m.name)
        bar = bars.winbar(m.team1_score, m.team2_score)
        if m.has_score:
            score = f"[{TEXT}]{m.team1_score}–{m.team2_score}[/]"
        else:
            score = f"[{MUTED}]TBD[/]"
        pick = d.pick_label(m.name)
        pick_txt = f"   [{MUTED}]{pick}[/]" if pick else ""
        return f"{icon} [{TEXT}]{m.name:<9}[/] {bar}  {score}{pick_txt}"

    def _map_table(self, m: MapScore) -> VimDataTable:
        table = VimDataTable(cursor_type="row", zebra_stripes=False)
        for name, width in _COLS:
            table.add_column(name, width=width)
        for p in sorted(m.players, key=lambda p: (p.acs or 0), reverse=True):
            table.add_row(
                Text(p.name, style=TEXT),
                self._agent_cell(p.agents),
                _num(p.acs),
                _num(p.k),
                _num(p.d),
                _num(p.a),
                _num(p.adr),
                _pct(p.hs_pct),
                _num(p.fk),
                _num(p.fd),
            )
        table.styles.height = len(m.players) + 1
        return table

    def _agent_cell(self, agents: list[str]) -> Text:
        if not agents:
            return Text("—", style=MUTED)
        glyph, colour = agent_glyph(agents[0])
        cell = Text()
        cell.append(f"{glyph} ", style=colour)
        cell.append(", ".join(agents), style=MUTED)
        return cell


def _num(v) -> Text:
    return Text("–" if v is None else (f"{v:.0f}" if isinstance(v, float) else str(v)))


def _pct(v) -> Text:
    return Text("–" if v is None else f"{v:.0f}%")
